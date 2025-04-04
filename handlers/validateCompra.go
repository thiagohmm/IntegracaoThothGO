package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"time"

	"github.com/google/uuid"

	"github.com/thiagohmm/producer/internal"
)

type HandlerAtena struct {
	Processo       string                 `json:"processo"`
	Processa       string                 `json:"processa"`
	StatusProcesso string                 `json:"statusProcesso"`
	DtRecebimento  string                 `json:"dt_recebimento"`
	Dados          map[string]interface{} `json:"dados"`
}

func (h *HandlerAtena) ToInternalDados() *internal.Dados {
	return &internal.Dados{
		Processo:       h.Processo,
		Processa:       h.Processa,
		StatusProcesso: h.StatusProcesso,
		DtRecebimento:  h.DtRecebimento,
		Dados:          h.Dados,
	}
}

func NewControllerAtena(inf map[string]interface{}, url string) (*HandlerAtena, error) {
	var tipoTransacao string
	if url == "/v1/EnviaDadosCompra" {
		tipoTransacao = "compra"
	} else if url == "/v1/EnviaDadosVendas" {
		tipoTransacao = "venda"
	} else {
		tipoTransacao = "estoque"
	}
	if inf == nil {
		return nil, errors.New("inf is nil")
	}

	return &HandlerAtena{
		Processo:       uuid.New().String(),
		StatusProcesso: "processando",
		Processa:       tipoTransacao,
		DtRecebimento:  time.Now().Format("02-01-2006 15:04:05"),
		Dados:          inf,
	}, nil
}

func (h *HandlerAtena) Salvar(w http.ResponseWriter, r *http.Request) {
	var objAtena map[string]interface{}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Erro ao ler o corpo da requisição", http.StatusBadRequest)
		return
	}
	//fmt.Println("Body:", string(body))

	values, _ := url.ParseQuery(string(body))

	fmt.Println("Values:", values)

	// Get the jsonCompactado value
	dadosJSON := values.Get("jsonCompactado")
	fmt.Println("jsonCompactado:", dadosJSON)
	if dadosJSON != "" {
		// Decode the jsonCompactado value
		objAtena, err = internal.ExtrairJsonAsync(dadosJSON)
		if err != nil {
			http.Error(w, "Erro ao extrair JSON", http.StatusInternalServerError)
			return
		}
	} else {
		err = json.Unmarshal(body, &objAtena)
		if err != nil {
			http.Error(w, "Erro ao converter JSON", http.StatusInternalServerError)
			return
		}

	}

	atenaobj, err := NewControllerAtena(objAtena, r.URL.String())
	if err != nil {
		http.Error(w, "Erro ao criar Controller", http.StatusInternalServerError)
		return
	}

	err = internal.GravarObjEmRedis(atenaobj.ToInternalDados())
	if err != nil {
		http.Error(w, "Erro ao serializar objeto", http.StatusInternalServerError)
		return
	}

	errCh := make(chan error)

	go func() {
		err = internal.GravarObjEmRedis(atenaobj.ToInternalDados())
		if err != nil {
			log.Printf("Erro ao gravar no Redis: %v", err)
			ReconnectRedis()
			errCh <- fmt.Errorf("Erro ao gravar no Redis: %w", err)
			return
		}

		err = internal.SendToQueue(atenaobj, errCh)
		if err != nil {
			log.Printf("Erro ao enviar para a fila: %v", err)
			ReconnectRabbitMQ()
			errCh <- fmt.Errorf("Erro ao enviar para a fila: %w", err)
			return
		}

		errCh <- nil // Indica sucesso
	}()

	// Espera o resultado da goroutine
	err = <-errCh
	if err != nil {
		log.Printf("Erro no processamento: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Println("Mensagem enviada com sucesso!")
	w.WriteHeader(http.StatusOK)
	// Defina o cabeçalho HTTP antes de escrever o corpo
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Escreva os dados serializados na resposta HTTP
	_, err = w.Write([]byte("Processo: " + atenaobj.Processo + "\nMensagem será processada"))

	if err != nil {
		fmt.Println("Erro ao escrever resposta:", err)
	}
}
