package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
)

// AuthProfile representa la estructura del perfil de autenticación
type AuthProfileData struct {
	ID         int               `json:"id"`
	ClientID   string            `json:"client_id"`
	UserID     int               `json:"user_id"`
	Attributes map[string]string `json:"attributes"`
}

// authProfile realiza la solicitud para obtener el perfil de autenticación
func AuthProfile(token string) (*AuthProfileData, error) {
	authProfileURL := os.Getenv("AUTH_PROFILE_URL")
	if authProfileURL == "" {
		return nil, errors.New("la variable de entorno AUTH_PROFILE_URL no está definida")
	}

	// Crear la solicitud HTTP
	req, err := http.NewRequest("GET", authProfileURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creando la solicitud: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	// Realizar la solicitud
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		// Manejar errores de conexión
		log.Printf("Error obteniendo perfil: %v", err)
		return nil, errors.New("error de conexión con el servicio de autenticación")
	}
	defer resp.Body.Close()

	// Procesar la respuesta según el código de estado
	switch resp.StatusCode {
	case http.StatusOK:
		var profile AuthProfileData
		if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
			log.Printf("Error parseando perfil: %v", err)
			return nil, errors.New("error procesando la respuesta del servidor")
		}
		return &profile, nil

	case http.StatusUnauthorized:
		log.Println("auth_profile response status: 401 Unauthorized")
		return nil, errors.New("no autorizado")

	default:
		log.Printf("auth_profile response status: %d", resp.StatusCode)
		return nil, errors.New("error interno del servidor")
	}
}
