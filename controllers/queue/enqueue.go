package queue

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/spf13/viper"
)

type QueueInfo struct {
	Name                   string `json:"name"`
	Messages               int    `json:"messages"`
	MessagesReady          int    `json:"messages_ready"`
	MessagesUnacknowledged int    `json:"messages_unacknowledged"`
}

func RabbitMqConnection() (*amqp.Connection, error) {
	// Connect to RabbitMQ
	conn, err := amqp.Dial(viper.GetString("rabbitmq.url"))
	if err != nil {
		fmt.Printf("Failed to connect to RabbitMQ: %s\n", err)
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	fmt.Println("Successfully connected to RabbitMQ")
	// Return the connection
	return conn, nil
}

func GetTotalMessagesInQueue() int {
	url := viper.GetString("rabbitmq.api")
	username := viper.GetString("rabbitmq.username")
	password := viper.GetString("rabbitmq.password")
	if url == "" {
		return 0
	}
	req, err := http.NewRequest(
		"GET",
		url,
		nil,
	)
	if err != nil {
		return 0
	}
	auth := username + ":" + password
	encoded := base64.StdEncoding.EncodeToString([]byte(auth))
	req.Header.Set("Authorization", "Basic "+encoded)
	client := &http.Client{}
	request, err := client.Do(req)
	fmt.Println("request", request)
	if err != nil {
		return 0
	}
	defer request.Body.Close()
	var totalMessage QueueInfo
	if request.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(request.Body)
		if err != nil {
			return 0
		}
		fmt.Println("body", string(bodyBytes))
		if err := json.Unmarshal(bodyBytes, &totalMessage); err != nil {
			return 0
		}
		fmt.Printf("Total messages in queue: %d\n", totalMessage.Messages)
		return totalMessage.Messages
	}
	return 0
}

func Producer(body json.RawMessage) error {
	conn, err := RabbitMqConnection()
	if err != nil {
		// Error already printed inside RabbitMqConnection
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		fmt.Printf("Failed to open a channel: %s\n", err)
		return fmt.Errorf("failed to open a channel: %w", err)
	}
	defer ch.Close()

	fmt.Println("Channel opened successfully in Producer")

	err = ch.ExchangeDeclare(
		"dlx_exchange",
		"direct", // exchange type
		true,     // durable
		false,    // auto-delete
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare dead letter exchange: %w", err)
	}

	_, err = ch.QueueDeclare(
		"dlx_queue", // must match your dead-letter-routing-key
		true,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return fmt.Errorf("failed to declare dead letter queue: %w", err)
	}

	err = ch.QueueBind(
		"dlx_queue",    // queue name
		"dlx_queue",    // routing key (must match DL routing key)
		"dlx_exchange", // exchange name
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind dead letter queue: %w", err)
	}

	// dead letter queue
	args := amqp.Table{
		"x-dead-letter-exchange":    "dlx_exchange",
		"x-dead-letter-routing-key": "dlx_queue",
	}

	queueName := viper.GetString("rabbitmq.queue_name")

	_, err = ch.QueueDeclare(
		queueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		args,  // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare a queue: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Set up rate limiting
	totalMessage := GetTotalMessagesInQueue()
	if totalMessage <= 5 {
		err = ch.PublishWithContext(ctx,
			"",        // exchange
			queueName, // routing key
			false,     // mandatory
			false,     // immediate
			amqp.Publishing{
				DeliveryMode: amqp.Persistent,
				ContentType:  "application/json",
				Body:         body,
			},
		)
		if err != nil {
			return fmt.Errorf("failed to publish a message: %w", err)
		}

		fmt.Printf("Message sent to queue %s: %s\n", queueName, body)
	}

	return nil

}
