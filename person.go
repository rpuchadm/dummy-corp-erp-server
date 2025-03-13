package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type PersonData struct {
	ID        int       `json:"id"`
	Dni       string    `json:"dni"`
	Nombre    string    `json:"nombre"`
	Apellidos string    `json:"apellidos"`
	Email     string    `json:"email"`
	Telefono  *string   `json:"telefono,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func getPersonsHandler(connStr string) http.HandlerFunc {
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
			http.Error(w, `{"error": "Método no permitido"}`, http.StatusMethodNotAllowed)
			return
		}

		// SQL para obtener todas las personas
		query := `SELECT id, dni, nombre, apellidos, email, telefono, created_at FROM persons;`
		rows, err := db.Query(query)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "Error al obtener las personas: %v"}`, err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Estructura para almacenar las personas
		var list []PersonData
		for rows.Next() {
			var item PersonData
			if err := rows.Scan(&item.ID, &item.Dni, &item.Nombre, &item.Apellidos, &item.Email, &item.Telefono, &item.CreatedAt); err != nil {
				http.Error(w, fmt.Sprintf(`{"error": "Error al escanear la persona: %v"}`, err), http.StatusInternalServerError)
				return
			}
			list = append(list, item)
		}

		// Convierte las personas a formato JSON
		jsonList, err := json.Marshal(list)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "Error al convertir las personas a JSON: %v"}`, err), http.StatusInternalServerError)
			return
		}

		// Responde con las personas en formato JSON
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonList)
	}
}

type PersonPostSent struct {
	Dni       string `json:"dni"`
	Nombre    string `json:"nombre"`
	Apellidos string `json:"apellidos"`
	Email     string `json:"email"`
	Telefono  string `json:"telefono"`
}

func postPersonHandler(connStr string) http.HandlerFunc {
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
			http.Error(w, `{"error": "Método no permitido"}`, http.StatusMethodNotAllowed)
			return
		}

		// Parsea el cuerpo de la solicitud en json
		var person PersonPostSent
		if err := json.NewDecoder(r.Body).Decode(&person); err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "Error al parsear el cuerpo de la solicitud: %v"}`, err), http.StatusBadRequest)
			return
		}

		// Verifica que los campos no estén vacíos
		if person.Dni == "" || person.Nombre == "" || person.Apellidos == "" || person.Email == "" {
			http.Error(w, `{"error": "Los campos dni, nombre, apellidos y email son requeridos"}`, http.StatusBadRequest)
			return
		}

		// SQL para insertar un mensaje
		var id int
		query := `INSERT INTO persons (dni, nombre, apellidos, email, telefono) VALUES ($1, $2, $3, $4, $5) RETURNING id;`
		err = db.QueryRow(query, person.Dni, person.Nombre, person.Apellidos, person.Email, person.Telefono).Scan(&id)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "Error al insertar la persona: %v"}`, err), http.StatusInternalServerError)
			return
		}

		//tiempo de espera de 2 segundos para poner drama
		time.Sleep(2 * time.Second)

		// Responde con un mensaje en formato JSON
		w.Write([]byte(`{"message": "Persona creada", "id": ` + fmt.Sprintf("%d", id) + `}`))
	}
}
