package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
)

func init() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}
}

// Função para pegar o status do processo no Redis
func (h *HandlerAtena) PegarStatusProcesso(w http.ResponseWriter, r *http.Request) {

	// Obtenha o ID do processo da URL
	processo := r.PathValue("processo")

	fmt.Println("Processo ID:", processo)

	// Configuração do Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("ENV_REDIS_ADDRESS"),
		Password: os.Getenv("ENV_REDIS_PASSWORD"),
	})
	defer rdb.Close()

	// Contexto com timeout para operações Redis
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := rdb.Get(ctx, processo).Result()

	if err == redis.Nil {
		http.Error(w, fmt.Sprintf("Status do processo %s não encontrado", processo), http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Deserializar o JSON armazenado
	var dados map[string]interface{}
	if err := json.Unmarshal([]byte(result), &dados); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Extrair o campo statusProcesso
	statusProcesso, ok := dados["statusProcesso"].(string)
	if !ok {
		http.Error(w, "Campo statusProcesso não encontrado", http.StatusInternalServerError)
		return
	}

	// Escreva o status como resposta
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("StatusProcesso: " + statusProcesso))
}
