package internal

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/streadway/amqp"
)

var (
	connection *amqp.Connection
	channel    *amqp.Channel
)

func init() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}
}

// GetRabbitMQChannel ensures a connection and channel are available, reusing them if they already exist
func GetRabbitMQChannel() (*amqp.Channel, error) {
	if connection == nil {
		rabbitmqURL := os.Getenv("ENV_RABBITMQ")
		if rabbitmqURL == "" {
			return nil, fmt.Errorf("RabbitMQ URL is not defined")
		}

		var err error
		connection, err = amqp.Dial(rabbitmqURL)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
		}
	}

	if channel == nil {
		var err error
		channel, err = connection.Channel()
		if err != nil {
			return nil, fmt.Errorf("failed to create channel: %w", err)
		}
	}

	return channel, nil
}

// AssertQueue ensures the queue exists
func AssertQueue(queue string) error {
	channel, err := GetRabbitMQChannel()
	if err != nil {
		return err
	}

	_, err = channel.QueueDeclare(
		queue, // name
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	return nil
}

// SendToQueue sends a JSON message to the specified RabbitMQ queue
func SendToQueue(dadosQueue interface{}, ch chan error) error {
	const queue = "thothQueue"

	if err := AssertQueue(queue); err != nil {
		ch <- fmt.Errorf("failed to assert queue: %w", err)
		return nil
	}

	channel, err := GetRabbitMQChannel()
	if err != nil {
		ch <- fmt.Errorf("failed to get RabbitMQ channel: %w", err)
		return nil
	}

	messageBody, err := json.Marshal(dadosQueue)
	if err != nil {
		ch <- fmt.Errorf("failed to marshal JSON: %w", err)
		return nil
	}

	err = channel.Publish(
		"",    // exchange
		queue, // routing key
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        messageBody,
		},
	)
	ch <- err

	return nil
}
