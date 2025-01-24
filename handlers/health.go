package handlers

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/streadway/amqp"
	"go.opentelemetry.io/otel"
)

var (
	rabbitConn  *amqp.Connection
	redisClient *redis.Client
)

func InitRabbitMQ() {
	var err error
	rabbitConn, err = amqp.Dial(os.Getenv("ENV_RABBITMQ"))
	if err != nil {
		log.Fatalf("Falha ao conectar no RabbitMQ: %v", err)
	}
}

func InitRedis() {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("ENV_REDIS_ADDRESS"),
		Password: os.Getenv("ENV_REDIS_PASSWORD"),
	})
}

func ReconnectRabbitMQ() {
	for {
		var err error
		rabbitConn, err = amqp.Dial(os.Getenv("RABBITMQ_URL"))
		if err != nil {
			log.Printf("Falha ao reconectar no RabbitMQ: %v", err)
			time.Sleep(5 * time.Second) // Espera 5 segundos antes de tentar novamente
			continue
		}
		log.Println("Reconectado ao RabbitMQ com sucesso")
		break
	}
}

func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Verifica Jaeger
	tracer := otel.Tracer("health-check")
	if tracer == nil {
		http.Error(w, "Jaeger não está conectado", http.StatusInternalServerError)
		return
	}

	// Verifica RabbitMQ
	if rabbitConn == nil || rabbitConn.IsClosed() {
		http.Error(w, "RabbitMQ não está conectado", http.StatusInternalServerError)
		return
	}

	// Verifica Redis
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		http.Error(w, "Redis não está conectado", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("ok"))
}
