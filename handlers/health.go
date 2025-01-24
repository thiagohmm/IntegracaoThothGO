package handlers

import (
	"context"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/streadway/amqp"
	"go.opentelemetry.io/otel"
)

var (
	rabbitConn  *amqp.Connection
	redisClient *redis.Client
	muRabbit    sync.Mutex // Para proteger o acesso ao RabbitMQ
	muRedis     sync.Mutex // Para proteger o acesso ao Redis
)

// InitRabbitMQ inicializa a conexão com o RabbitMQ
func InitRabbitMQ() {
	muRabbit.Lock()
	defer muRabbit.Unlock()

	rabbitURL := os.Getenv("ENV_RABBITMQ")
	if rabbitURL == "" {
		log.Fatal("A URL do RabbitMQ não está definida")
	}

	var err error
	rabbitConn, err = amqp.Dial(rabbitURL)
	if err != nil {
		log.Fatalf("Falha ao conectar no RabbitMQ: %v", err)
	}

	log.Println("Conexão com RabbitMQ estabelecida")
}

// InitRedis inicializa a conexão com o Redis
func InitRedis() {
	muRedis.Lock()
	defer muRedis.Unlock()

	redisClient = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("ENV_REDIS_ADDRESS"),
		Password: os.Getenv("ENV_REDIS_PASSWORD"),
	})

	_, err := redisClient.Ping(context.Background()).Result()
	if err != nil {
		log.Fatalf("Falha ao conectar no Redis: %v", err)
	}

	log.Println("Conexão com Redis estabelecida")
}

// ReconnectRabbitMQ tenta reconectar ao RabbitMQ com retries
func ReconnectRabbitMQ() {
	muRabbit.Lock()
	defer muRabbit.Unlock()

	for {
		if rabbitConn != nil && !rabbitConn.IsClosed() {
			rabbitConn.Close()
		}

		rabbitURL := os.Getenv("ENV_RABBITMQ")
		if rabbitURL == "" {
			log.Println("A URL do RabbitMQ não está definida")
			time.Sleep(5 * time.Second)
			continue
		}

		var err error
		rabbitConn, err = amqp.Dial(rabbitURL)
		if err != nil {
			log.Printf("Falha ao reconectar no RabbitMQ: %v. Tentando novamente em 5 segundos...", err)
			time.Sleep(5 * time.Second)
			continue
		}

		// Testar a conexão abrindo um canal
		channel, err := rabbitConn.Channel()
		if err != nil {
			log.Printf("Falha ao abrir canal após reconexão: %v. Tentando novamente em 5 segundos...", err)
			time.Sleep(5 * time.Second)
			continue
		}
		channel.Close()

		log.Println("Reconexão com RabbitMQ bem-sucedida")
		break
	}
}

// ReconnectRedis tenta reconectar ao Redis com retries
func ReconnectRedis() {
	muRedis.Lock()
	defer muRedis.Unlock()

	for {
		_, err := redisClient.Ping(context.Background()).Result()
		if err != nil {
			log.Printf("Falha ao reconectar no Redis: %v. Tentando novamente em 5 segundos...", err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Println("Reconexão com Redis bem-sucedida")
		break
	}
}

// HealthCheckHandler verifica o estado de RabbitMQ, Redis e Jaeger
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
		go ReconnectRabbitMQ()
		return
	}

	// Verifica Redis
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		http.Error(w, "Redis não está conectado", http.StatusInternalServerError)
		go ReconnectRedis()
		return
	}

	w.Write([]byte("ok"))
}
