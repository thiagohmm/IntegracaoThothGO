package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/thiagohmm/producer/internal"
)

type HandlerAtena struct {
	Processo       string                 `json:"processo"`
	Processa       string                 `json:"processa"`
	StatusProcesso string                 `json:"statusProcesso"`
	Dados          map[string]interface{} `json:"dados"`
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

	values, err := url.ParseQuery(string(body))
	if err != nil {
		http.Error(w, "Erro ao analisar o corpo da requisição", http.StatusBadRequest)
		return
	}

	//fmt.Println("Values:", values)

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

		http.Error(w, "Erro ao criar ControllerCompra", http.StatusInternalServerError)
		return
	}

	if err != nil {
		http.Error(w, "Erro ao serializar objeto", http.StatusInternalServerError)
		return
	}

	if err != nil {
		http.Error(w, "Erro ao serializar objeto", http.StatusInternalServerError)
		return
	}

	errCh := make(chan error)

	go func() {
		err = internal.GravarObjEmRedis((*internal.Dados)(atenaobj))
		if err != nil {
			http.Error(w, "Erro de gravacao de processo", http.StatusInternalServerError)
			log.Print(err)
			return
		}
		err := internal.SendToQueue(atenaobj, errCh)
		errCh <- err
	}()

	err = <-errCh
	if err != nil {
		fmt.Println("Error sending to queue:", err)
	} else {
		fmt.Println("Message sent successfully")
	}
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println("Erro ao enviar para a fila:", err)
		http.Error(w, "Erro ao enviar para a fila", http.StatusInternalServerError)
		ReconnectRabbitMQ()
		return
	}

	// Defina o cabeçalho HTTP antes de escrever o corpo
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Escreva os dados serializados na resposta HTTP
	_, err = w.Write([]byte("Processo: " + atenaobj.Processo + "\nMensagem será processada"))

	if err != nil {
		fmt.Println("Erro ao escrever resposta:", err)
	}
}
