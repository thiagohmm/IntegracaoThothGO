package internal

import (
	"context"
	"encoding/json"
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
func GravarCompraEmRedis(dados *Dados) error {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Endereço do Redis
		Password: "eYVX7EwVmmxKPCDmwMtyKVge8oLd2t81",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	compraJSON, err := json.Marshal(dados)
	if err != nil {
		return err
	}

	// Grava o JSON no Redis com uma chave baseada no Processo
	expiracao := 5 * 24 * time.Hour
	err = rdb.Set(ctx, dados.Processo, compraJSON, expiracao).Err()
	return err
}
