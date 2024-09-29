# Build Stage
FROM golang:1.21-alpine AS builder
WORKDIR /scholarize_backend
COPY go.mod go.sum ./
RUN apk update && apk add --no-cache tzdata
RUN go mod tidy
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o scholarize_backend main.go

# Run Stage
FROM alpine:latest
RUN apk update && apk add --no-cache tzdata
ENV TZ=Asia/Bangkok
WORKDIR /scholarize_backend

# Copy the entire application directory
COPY --from=builder /scholarize_backend /scholarize_backend

EXPOSE 2812

CMD ["./scholarize_backend"]
