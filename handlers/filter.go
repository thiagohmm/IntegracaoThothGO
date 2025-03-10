package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
)

type Filtro struct {
	DtaEnvio string `json:"dtaenvio"`
	DtaVenda string `json:"dtavenda"`
	IBMS     string `json:"ibms"`
	App      string `json:"app"`
	QtdCupom int    `json:"qtdcupom"`
}

func (h *HandlerAtena) Filtrar(w http.ResponseWriter, r *http.Request) {
	// Verifica se o método é POST
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	// Decodifica o corpo da requisição para o struct Filtro
	var filtro Filtro

	if err := json.NewDecoder(r.Body).Decode(&filtro); err != nil {
		http.Error(w, "Erro ao decodificar JSON", http.StatusBadRequest)
		return
	}

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

	var resultJSON string
	for _, key := range keys {
		result, err := rdb.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var dados map[string]interface{}
		if err := json.Unmarshal([]byte(result), &dados); err != nil {
			continue
		}

		// Verifica se os dados correspondem ao filtro
		if dadosMap, ok := dados["dados"].(map[string]interface{}); ok {
			if vendas, ok := dadosMap["vendas"].(map[string]interface{}); ok {
				if vendas["dtaenvio"] == filtro.DtaEnvio ||
					vendas["dtavenda"] == filtro.DtaVenda {
					if ibmsList, ok := vendas["ibms"].([]interface{}); ok && len(ibmsList) > 0 {
						if ibms, ok := ibmsList[0].(map[string]interface{}); ok {
							if ibms["app"] == filtro.App ||
								ibms["nro"] == filtro.IBMS ||
								ibms["qtdcupom"] == float64(filtro.QtdCupom) {
								resultJSON = result
								break
							}
						}
					}
				}
			}
		}
	}

	if resultJSON == "" {
		http.Error(w, "JSON não encontrado", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(resultJSON))
}
