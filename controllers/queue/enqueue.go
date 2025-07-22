package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/spf13/viper"
	"golang.org/x/time/rate"
)

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

func Consumer() {
	conn, err := RabbitMqConnection()
	if err != nil {
		fmt.Printf("Failed to connect to RabbitMQ: %s\n", err)
		return
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		fmt.Printf("Failed to open a channel: %s\n", err)
		return
	}
	defer ch.Close()

	notifyConsumer, err := ch.QueueDeclare(
		"notification_queue",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		fmt.Printf("Failed to declare a queue: %s\n", err)
		return
	}
	err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)

	if err != nil {
		fmt.Printf("Failed to set QoS: %s\n", err)
		return
	}
	msgs, err := ch.Consume(
		notifyConsumer.Name, // queue name
		"",                  // consumer tag
		true,                // auto-acknowledge
		false,               // exclusive
		false,               // no-local
		false,               // no-wait
		nil,                 // arguments
	)
	if err != nil {
		fmt.Printf("Failed to register a consumer: %s\n", err)
		return
	}
	var forever chan struct{}

	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)
			dotCount := bytes.Count(d.Body, []byte("."))
			t := time.Duration(dotCount)
			time.Sleep(t * time.Second)
			log.Printf("Done")
			d.Ack(false)
		}
	}()
	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
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
	rateLimiter := rate.NewLimiter(rate.Every(1*time.Hour), 10)
	for i := 0; i < 10; i++ {
		err := rateLimiter.Wait(context.Background())
		if err != nil {
			log.Fatalf("Limiter error: %v", err)
		}
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
	}
	fmt.Printf("Message sent to queue %s: %s\n", queueName, body)
	return nil
}
