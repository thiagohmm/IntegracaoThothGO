package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

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

// connect establishes a connection and a channel to RabbitMQ, with automatic reconnection.
func connect() error {
	rabbitmqURL := os.Getenv("ENV_RABBITMQ")
	if rabbitmqURL == "" {
		return fmt.Errorf("RabbitMQ URL is not defined")
	}

	for { // Loop infinito para tentativas de reconexão
		var err error
		connection, err = amqp.Dial(rabbitmqURL)
		if err != nil {
			fmt.Printf("Failed to connect to RabbitMQ: %v. Retrying in 5 seconds...\n", err)
			time.Sleep(5 * time.Second)
			continue // Volta para o início do loop para tentar novamente
		}

		channel, err = connection.Channel()
		if err != nil {
			fmt.Printf("Failed to create channel: %v. Retrying in 5 seconds...\n", err)
			connection.Close() // Fecha a conexão se o canal não for criado
			connection = nil   // Reseta a conexão
			time.Sleep(5 * time.Second)
			continue // Volta para o início do loop para tentar novamente
		}

		fmt.Println("Successfully connected to RabbitMQ and created channel.")
		return nil // Conexão e canal estabelecidos com sucesso
	}
}

// GetRabbitMQChannel ensures a connection and channel are available, reusing them if they already exist
func GetRabbitMQChannel() (*amqp.Channel, error) {
	if connection == nil || channel == nil || connection.IsClosed() {
		if err := connect(); err != nil {
			return nil, fmt.Errorf("failed to establish connection: %w", err)
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
		// Verifica se o erro é relacionado a conexão fechada
		if err, ok := err.(*amqp.Error); ok && err.Code == amqp.ChannelError {
			fmt.Printf("Channel error: %v. Reconnecting...\n", err)
			if err := connect(); err != nil {
				return fmt.Errorf("failed to reconnect after channel error: %w", err)
			}

			// Tenta declarar a fila novamente após reconexão
			return AssertQueue(queue)
		}
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
	if err != nil {
		// Verifica se o erro é relacionado a conexão fechada
		if err, ok := err.(*amqp.Error); ok && err.Code == amqp.ChannelError {
			fmt.Printf("Connection or channel error: %v. Reconnecting...\n", err)
			if err := connect(); err != nil {
				ch <- fmt.Errorf("failed to reconnect after connection/channel error: %w", err)
				return nil
			}
			// Tenta reenviar a mensagem após reconexão
			if err := SendToQueue(dadosQueue, ch); err != nil {
				ch <- fmt.Errorf("failed to send message after reconnecting: %w", err)
				return nil
			}
			ch <- nil
			return nil
		}

		ch <- fmt.Errorf("failed to publish message: %w", err)
		return nil
	}

	ch <- err
	return nil
}
