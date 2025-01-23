package internal

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

type Dados struct {
	Processo       string                 `json:"processo"`
	Processa       string                 `json:"processa"`
	StatusProcesso string                 `json:"statusProcesso"`
	Dados          map[string]interface{} `json:"dados"`
}

// GravarCompraEmRedis grava o objeto dados no Redis
func GravarObjEmRedis(dados *Dados) error {
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("ENV_REDIS_ADDRESS"),  // Endereço do Redis
		Password: os.Getenv("ENV_REDIS_PASSWORD"), // Senha do Redis
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	compraJSON, err := json.Marshal(dados)
	if err != nil {
		return err
	}

	// Grava o JSON no Redis com uma chave baseada no Processo
	expireHours, err := strconv.Atoi(os.Getenv("ENV_REDIS_EXPIRE"))
	if err != nil {
		return err
	}
	expiracao := time.Duration(expireHours) * 24 * time.Hour
	err = rdb.Set(ctx, dados.Processo, compraJSON, expiracao).Err()
	return err
}
