package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type AuthClient struct {
	ID        int    `json:"id"`
	ClientID  string `json:"client_id"`
	ClientUrl string `json:"client_url"`
	CreatedAt string `json:"created_at"`
}

func getAuthClientsHandler(connStr string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Abre la conexión a la base de datos
		var err error
		db, err := openDatabaseConnection(connStr)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		// Verifica que el método sea GET
		if r.Method != http.MethodGet {
			errJsonStatus(w, `Método no permitido`, http.StatusMethodNotAllowed)
			return
		}

		// SQL para obtener todos los clientes
		query := `SELECT id, client_id, client_url, created_at FROM auth_clients;`
		rows, err := db.Query(query)
		if err != nil {
			errJsonStatus(w, fmt.Sprintf(`Error al obtener los clientes: %v`, err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Estructura para almacenar los clientes
		var list []AuthClient
		for rows.Next() {
			var item AuthClient
			if err := rows.Scan(&item.ID, &item.ClientID, &item.ClientUrl, &item.CreatedAt); err != nil {
				errJsonStatus(w, fmt.Sprintf(`Error al escanear el cliente: %v`, err), http.StatusInternalServerError)
				return
			}
			list = append(list, item)
		}

		// Convierte los clientes a formato JSON
		jsonList, err := json.Marshal(list)
		if err != nil {
			errJsonStatus(w, fmt.Sprintf(`Error al convertir los clientes a JSON: %v`, err), http.StatusInternalServerError)
			return
		}

		// Responde con los clientes en formato JSON
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonList)
	}
}

type AuthClientPostSent struct {
	ClientID  string `json:"client_id"`
	ClientUrl string `json:"client_url"`
}

func authClientHandler(connStr string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			postAuthClientHandler(connStr)(w, r)
		case http.MethodPut:
			putAuthClientHandler(connStr)(w, r)
		case http.MethodDelete:
			deleteAuthClientHandler(connStr)(w, r)
		default:
			errJsonStatus(w, `Método no permitido`, http.StatusMethodNotAllowed)
		}
	}
}

func postAuthClientHandler(connStr string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Abre la conexión a la base de datos
		var err error
		db, err := openDatabaseConnection(connStr)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		// Verifica que el método sea POST
		if r.Method != http.MethodPost {
			errJsonStatus(w, `Método no permitido`, http.StatusMethodNotAllowed)
			return
		}

		// Decodifica el JSON recibido
		var sent AuthClientPostSent
		err = json.NewDecoder(r.Body).Decode(&sent)
		if err != nil {
			errJsonStatus(w, fmt.Sprintf(`Error al decodificar el JSON: %v`, err), http.StatusBadRequest)
			return
		}

		// SQL para insertar un nuevo cliente
		query := `INSERT INTO auth_clients (client_id, client_url) VALUES ($1, $2) RETURNING id, created_at;`
		row := db.QueryRow(query, sent.ClientID, sent.ClientUrl)

		// Estructura para almacenar el cliente insertado
		var item AuthClient
		err = row.Scan(&item.ID, &item.CreatedAt)
		if err != nil {
			errJsonStatus(w, fmt.Sprintf(`Error al insertar el cliente: %v`, err), http.StatusInternalServerError)
			return
		}
		item.ClientID = sent.ClientID
		item.ClientUrl = sent.ClientUrl

		// Convierte el cliente insertado a formato JSON
		jsonItem, err := json.Marshal(item)
		if err != nil {
			errJsonStatus(w, fmt.Sprintf(`Error al convertir el cliente a JSON: %v`, err), http.StatusInternalServerError)
			return
		}

		// Responde con el cliente insertado en formato JSON
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonItem)
	}
}

func putAuthClientHandler(connStr string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Abre la conexión a la base de datos
		var err error
		db, err := openDatabaseConnection(connStr)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		// Verifica que el método sea PUT
		if r.Method != http.MethodPut {
			errJsonStatus(w, `Método no permitido`, http.StatusMethodNotAllowed)
			return
		}

		// Decodifica el JSON recibido
		var sent AuthClient
		err = json.NewDecoder(r.Body).Decode(&sent)
		if err != nil {
			errJsonStatus(w, fmt.Sprintf(`Error al decodificar el JSON: %v`, err), http.StatusBadRequest)
			return
		}

		// SQL para actualizar un cliente
		query := `UPDATE auth_clients SET client_id = $1, client_url = $2 WHERE id = $3 RETURNING created_at;`
		row := db.QueryRow(query, sent.ClientID, sent.ClientUrl, sent.ID)

		// Estructura para almacenar la fecha de creación
		var createdAt string
		err = row.Scan(&createdAt)
		if err != nil {
			errJsonStatus(w, fmt.Sprintf(`Error al actualizar el cliente: %v`, err), http.StatusInternalServerError)
			return
		}

		// Convierte la fecha de creación a formato JSON
		jsonCreatedAt, err := json.Marshal(createdAt)
		if err != nil {
			errJsonStatus(w, fmt.Sprintf(`Error al convertir la fecha de creación a JSON: %v`, err), http.StatusInternalServerError)
			return
		}

		// Responde con la fecha de creación en formato JSON
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonCreatedAt)
	}
}

func deleteAuthClientHandler(connStr string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Abre la conexión a la base de datos
		var err error
		db, err := openDatabaseConnection(connStr)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		// Verifica que el método sea DELETE
		if r.Method != http.MethodDelete {
			errJsonStatus(w, `Método no permitido`, http.StatusMethodNotAllowed)
			return
		}

		// SQL para eliminar todos los clientes
		query := `DELETE FROM auth_clients;`
		_, err = db.Exec(query)
		if err != nil {
			errJsonStatus(w, fmt.Sprintf(`Error al eliminar los clientes: %v`, err), http.StatusInternalServerError)
			return
		}

		// Responde con un mensaje de éxito
		w.Write([]byte(`{"message": "Clientes eliminados"}`))
	}
}
