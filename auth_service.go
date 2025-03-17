package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type AuthServicePostSession struct {
	ClientId     string                 `json:"client_id"`
	UserId       int                    `json:"user_id"`
	ExpiresInMin int                    `json:"expires_in_min"`
	Attributes   map[string]interface{} `json:"attributes"`
}

type AuthServicePostSessionResponse struct {
	ClientId   string `json:"client_id"`
	Code       string `json:"code"`
	UserId     int    `json:"user_id"`
	ExpiresAt  any    `json:"expires_at"`
	Attributes any    `json:"attributes"`
}

// constante con la url del servicio de autenticación
const AUTH_SERVICE_URL = "http://localhost:8080"

func auth_service_post_session(client_id string, user_id int, expires_in_min int, attributes map[string]any) (string, error) {

	super_secret_token := os.Getenv("AUTH_SUPER_SECRET_TOKEN")
	if super_secret_token == "" {
		return "", fmt.Errorf("AUTH_SUPER_SECRET_TOKEN not set")
	}

	auth_service_url := os.Getenv("AUTH_SERVICE_URL")
	if auth_service_url == "" {
		auth_service_url = AUTH_SERVICE_URL
	}

	// crear la estructura con los datos a enviar
	data := AuthServicePostSession{
		ClientId:     client_id,
		UserId:       user_id,
		ExpiresInMin: expires_in_min,
		Attributes:   attributes,
	}

	// Convertir los datos a JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("error al convertir los datos a JSON: %v", err)
	}

	// enviar los datos al servicio de autenticación
	req, err := http.NewRequest("POST", auth_service_url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error al crear la solicitud HTTP: %v", err)
	}
	// Agregar el token de autenticación en el header
	req.Header.Set("Authorization", "Bearer "+super_secret_token)
	req.Header.Set("Content-Type", "application/json")

	// Realizar la solicitud HTTP
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error al realizar la solicitud HTTP: %v", err)
	}
	defer resp.Body.Close()

	// Leer la respuesta del servidor
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error al leer la respuesta del servidor: %v", err)
	}

	// Verificar el código de estado de la respuesta
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("código de estado inesperado: %d, respuesta: %s", resp.StatusCode, string(body))
	}

	// Decodificar la respuesta JSON en la estructura ResponseData
	var response AuthServicePostSessionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("error al decodificar la respuesta JSON: %v", err)
	}

	return response.Code, nil
}
