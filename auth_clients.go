package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
)

type AuthClient struct {
	ID                int     `json:"id"`
	ClientID          string  `json:"client_id"`
	ClientUrl         string  `json:"client_url"`
	ClientUrlCallback *string `json:"client_url_callback,omitempty"`
	ClientSecret      *string `json:"client_secret,omitempty"`
	CreatedAt         string  `json:"created_at"`
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
		query := `
			SELECT 
				id, client_id, 
				client_url, client_url_callback,
				client_secret,
				created_at 
			FROM auth_clients;`
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
			if err := rows.Scan(&item.ID, &item.ClientID,
				&item.ClientUrl,
				&item.ClientUrlCallback,
				&item.ClientSecret,
				&item.CreatedAt); err != nil {
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
	ClientID          string  `json:"client_id"`
	ClientUrl         string  `json:"client_url"`
	ClientUrlCallback *string `json:"client_url_callback,omitempty"`
}

func authClientHandler(connStr string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Obtiene el ID de la persona
		path := strings.TrimPrefix(r.URL.Path, "/application/")
		id := strings.Split(path, "/")[0]

		// parsear el id a int
		iid, err := strconv.Atoi(id)
		if err != nil {
			errJsonStatus(w, fmt.Sprintf(`Error al parsear el id: %v`, err), http.StatusBadRequest)
			return
		}

		//fmt.Printf("method:%v iid: %d\n", r.Method, iid)

		switch r.Method {
		case http.MethodGet:
			getAuthClientHandler(connStr, iid)(w, r)
		case http.MethodPost:
			postAuthClientHandler(connStr, iid)(w, r)
		case http.MethodPut:
			putAuthClientHandler(connStr, iid)(w, r)
		case http.MethodDelete:
			deleteAuthClientHandler(connStr, iid)(w, r)
		default:
			errJsonStatus(w, `Método no permitido`, http.StatusMethodNotAllowed)
		}
	}
}

func getAuthClientHandler(connStr string, iid int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if iid == 0 {
			errJsonStatus(w, `IEl campo id es requerido`, http.StatusBadRequest)
			return
			//} else {
			//	fmt.Printf("getAuthClientHandler iid:%d\n", iid)
		}

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

		// SQL para obtener un cliente
		query := fmt.Sprintf(`
			SELECT
				id, client_id,
				client_url, client_url_callback,
				client_secret,
				created_at
			FROM auth_clients
				WHERE id = %d;`, iid)
		stmt, err := db.Prepare(query)
		if err != nil {
			errJsonStatus(w, fmt.Sprintf(`Error al preparar la consulta %v %v`, query, err), http.StatusInternalServerError)
			return
		}
		defer stmt.Close()

		row, err := stmt.Query()
		if err != nil {
			errJsonStatus(w, fmt.Sprintf(`Error al obtener el cliente: %v`, err), http.StatusInternalServerError)
			return
		}
		defer row.Close()

		// Estructura para almacenar el cliente
		var item AuthClient
		if row.Next() {
			err = row.Scan(&item.ID, &item.ClientID,
				&item.ClientUrl,
				&item.ClientUrlCallback,
				&item.ClientSecret,
				&item.CreatedAt)
			if err != nil {
				errJsonStatus(w, fmt.Sprintf(`Error al obtener el cliente: %v`, err), http.StatusInternalServerError)
				return
			}
		} else {
			errJsonStatus(w, `Cliente no encontrado`, http.StatusNotFound)
			return
		}

		// Convierte el cliente a formato JSON
		jsonItem, err := json.Marshal(item)
		if err != nil {
			errJsonStatus(w, fmt.Sprintf(`Error al convertir el cliente a JSON: %v`, err), http.StatusInternalServerError)
			return
		}

		// Responde con el cliente en formato JSON
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonItem)
	}
}

func postAuthClientHandler(connStr string, iid int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if iid != 0 {
			errJsonStatus(w, `El campo id no puede ser diferente de 0 para insertar`, http.StatusBadRequest)
			return
		}

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
		query := `
			INSERT INTO auth_clients (client_id, client_url)
			VALUES ($1, $2) RETURNING id, created_at;`
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

func putAuthClientHandler(connStr string, iid int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if iid == 0 {
			errJsonStatus(w, `El campo id es requerido para actualizar`, http.StatusBadRequest)
			return
		}

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

		if sent.ClientUrlCallback != nil {
			token := tokenCreate(96)
			sent.ClientSecret = &token
		}

		// SQL para actualizar un cliente
		query := `
			UPDATE
				auth_clients
			SET
				client_id = $1, client_url = $2,
				client_url_callback = $3, client_secret = $4
			WHERE id = $5;`
		_, err = db.Exec(
			query,
			sent.ClientID, sent.ClientUrl,
			sent.ClientUrlCallback, sent.ClientSecret,
			sent.ID)
		if err != nil {
			errJsonStatus(w, fmt.Sprintf(`Error al actualizar el cliente: %v`, err), http.StatusInternalServerError)
			return
		}

		// Convierte el cliente actualizado a formato JSON
		jsonItem, err := json.Marshal(sent)
		if err != nil {
			errJsonStatus(w, fmt.Sprintf(`Error al convertir el cliente a JSON: %v`, err), http.StatusInternalServerError)
			return
		}

		// Responde con el cliente insertado en formato JSON
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonItem)
	}
}

func deleteAuthClientHandler(connStr string, iid int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if iid == 0 {
			errJsonStatus(w, `El campo id es requerido para eliminar`, http.StatusBadRequest)
			return
		}

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
		query := fmt.Sprintf(`DELETE FROM auth_clients where id=%d;`, iid)
		_, err = db.Exec(query)
		if err != nil {
			errJsonStatus(w, fmt.Sprintf(`Error al eliminar los clientes: %v`, err), http.StatusInternalServerError)
			return
		}

		// Responde con un mensaje de éxito
		w.Write([]byte(`{"message": "Cliente eliminado"}`))
	}
}

func tokenCreate(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
