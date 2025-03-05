package handlers

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "time"

    "github.com/go-redis/redis/v8"
    "github.com/thiagohmm/producer/internal"
)

func (h *HandlerAtena) Reenqueue(w http.ResponseWriter, r *http.Request) {
    // Verifica se o RabbitMQ está ativo
    channel, err := internal.GetRabbitMQChannel()
    if err != nil {
        http.Error(w, "RabbitMQ não está conectado", http.StatusInternalServerError)
        return
    }
    defer channel.Close()

    // Configuração do Redis
    rdb := redis.NewClient(&redis.Options{
        Addr:     os.Getenv("ENV_REDIS_ADDRESS"),
        Password: os.Getenv("ENV_REDIS_PASSWORD"),
    })
    defer rdb.Close()

    // Contexto com timeout para operações Redis
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // Obtenha todas as chaves
    keys, err := rdb.Keys(ctx, "*").Result()
    if err != nil {
        http.Error(w, "Erro ao obter chaves do Redis", http.StatusInternalServerError)
        return
    }

    errCh := make(chan error)

    for _, key := range keys {
        result, err := rdb.Get(ctx, key).Result()
        if err != nil {
            http.Error(w, fmt.Sprintf("Erro ao obter valor para a chave %s", key), http.StatusInternalServerError)
            return
        }

        var dados map[string]interface{}
        if err := json.Unmarshal([]byte(result), &dados); err != nil {
            http.Error(w, fmt.Sprintf("Erro ao deserializar JSON para a chave %s", key), http.StatusInternalServerError)
            return
        }

        status, ok := dados["statusProcesso"].(string)
        if !ok || status == "processando" {
            go func(dados map[string]interface{}) {
                err := internal.SendToQueue(dados, errCh)
                if err != nil {
                    http.Error(w, fmt.Sprintf("Erro ao enviar para a fila: %v", err), http.StatusInternalServerError)
                    return
                }
            }(dados)
        }
    }

    // Espera o resultado da goroutine
    err = <-errCh
    if err != nil {
        http.Error(w, fmt.Sprintf("Erro no processamento: %v", err), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Reenqueued successfully"))
}