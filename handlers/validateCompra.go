package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/thiagohmm/producer/internal"
)

type HandlerCompra struct {
	Processa string                 `json:"processa"`
	Dados    map[string]interface{} `json:"dados"`
}

func NewControllerCompra(inf map[string]interface{}, url string) (*HandlerCompra, error) {
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

	return &HandlerCompra{
		Processa: tipoTransacao,
		Dados:    inf,
	}, nil
}

func (h *HandlerCompra) Salvar(w http.ResponseWriter, r *http.Request) {
	var compra map[string]interface{}

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
		compra, err = internal.ExtrairJsonAsync(dadosJSON)
		if err != nil {
			http.Error(w, "Erro ao extrair JSON", http.StatusInternalServerError)
			return
		}
	} else {
		err = json.Unmarshal(body, &compra)
		if err != nil {
			http.Error(w, "Erro ao converter JSON", http.StatusInternalServerError)
			return
		}

	}

	compraobj, err := NewControllerCompra(compra, r.URL.String())
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
		err := internal.SendToQueue(compraobj, errCh)
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
		return
	}

	// Defina o cabeçalho HTTP antes de escrever o corpo
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Escreva os dados serializados na resposta HTTP
	_, err = w.Write([]byte("Mensagem será processada"))
	if err != nil {
		fmt.Println("Erro ao escrever resposta:", err)
	}
}
