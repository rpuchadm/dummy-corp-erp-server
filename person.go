package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
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
			errJsonStatus(w, `Método no permitido`, http.StatusMethodNotAllowed)
			return
		}

		// SQL para obtener todas las personas
		query := `SELECT id, dni, nombre, apellidos, email, telefono, created_at FROM persons;`
		rows, err := db.Query(query)
		if err != nil {
			errJsonStatus(w, fmt.Sprintf(`Error al obtener las personas: %v`, err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Estructura para almacenar las personas
		var list []PersonData
		for rows.Next() {
			var item PersonData
			if err := rows.Scan(&item.ID, &item.Dni, &item.Nombre, &item.Apellidos, &item.Email, &item.Telefono, &item.CreatedAt); err != nil {
				errJsonStatus(w, fmt.Sprintf(`Error al escanear la persona: %v`, err), http.StatusInternalServerError)
				return
			}
			list = append(list, item)
		}

		// Convierte las personas a formato JSON
		jsonList, err := json.Marshal(list)
		if err != nil {
			errJsonStatus(w, fmt.Sprintf(`Error al convertir las personas a JSON: %v`, err), http.StatusInternalServerError)
			return
		}

		// Responde con las personas en formato JSON
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonList)
	}
}

func personHandler(connStr string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Obtiene el ID de la persona
		path := strings.TrimPrefix(r.URL.Path, "/person/")
		id := strings.Split(path, "/")[0]

		// parsear el id a int
		iid, err := strconv.Atoi(id)
		if err != nil {
			errJsonStatus(w, fmt.Sprintf(`Error al parsear el id: %v`, err), http.StatusBadRequest)
			return
		}

		// Abre la conexión a la base de datos
		db, err := openDatabaseConnection(connStr)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		if r.Method == http.MethodGet {

			if iid == 0 {
				errJsonStatus(w, `El campo id es requerido`, http.StatusBadRequest)
				return
			}

			person, err := postgres_person_by_id(db, iid)
			if err != nil {
				errJsonStatus(w, fmt.Sprintf(`Error al obtener la persona: %v`, err), http.StatusInternalServerError)
				return
			}

			lpersonapp, err := postgres_personapp_by_person_id(db, iid)
			if err != nil {
				errJsonStatus(w, fmt.Sprintf(`Error al obtener la personaapp: %v`, err), http.StatusInternalServerError)
				return
			}

			lapp, err := postgres_auth_client_by_person_id(db, iid)
			if err != nil {
				errJsonStatus(w, fmt.Sprintf(`Error al obtener la app: %v`, err), http.StatusInternalServerError)
				return
			}

			data := make(map[string]any)
			data["person"] = person
			if len(lpersonapp) > 0 {
				data["lpersonapp"] = lpersonapp
			}
			if len(lapp) > 0 {
				data["lapp"] = lapp
			}

			// Convierte data a formato JSON
			jsonPerson, err := json.Marshal(data)

			if err != nil {
				errJsonStatus(w, fmt.Sprintf(`Error al convertir la persona a JSON: %v`, err), http.StatusInternalServerError)
				return
			}

			// Responde con la persona en formato JSON
			w.Header().Set("Content-Type", "application/json")
			w.Write(jsonPerson)

			return

		} else if r.Method == http.MethodPost {

			if iid != 0 {
				errJsonStatus(w, `El campo id no puede ser diferente de 0 para insertar`, http.StatusBadRequest)
				return
			}

			// Parsea el cuerpo de la solicitud en json
			var person PersonPostSent
			if err := json.NewDecoder(r.Body).Decode(&person); err != nil {
				errJsonStatus(w, fmt.Sprintf(`Error al parsear el cuerpo de la solicitud: %v`, err), http.StatusBadRequest)
				return
			}

			// Verifica que los campos no estén vacíos
			if person.Dni == "" || person.Nombre == "" || person.Apellidos == "" || person.Email == "" {
				errJsonStatus(w, `Los campos dni, nombre, apellidos y email son requeridos`, http.StatusBadRequest)
				return
			}

			// SQL para insertar un mensaje
			var id int
			query := `INSERT INTO persons (dni, nombre, apellidos, email, telefono) VALUES ($1, $2, $3, $4, $5) RETURNING id;`
			err = db.QueryRow(query, person.Dni, person.Nombre, person.Apellidos, person.Email, person.Telefono).Scan(&id)
			if err != nil {
				errJsonStatus(w, fmt.Sprintf(`Error al insertar la persona: %v`, err), http.StatusInternalServerError)
				return
			}

			//tiempo de espera de 2 segundos para poner drama
			time.Sleep(2 * time.Second)

			// Responde con un mensaje en formato JSON
			w.Write([]byte(`{"message": "Persona creada", "id": ` + fmt.Sprintf("%d", id) + `}`))

			return

		} else if r.Method == http.MethodPut {

			if iid == 0 {
				errJsonStatus(w, `El campo id es requerido para actualizar`, http.StatusBadRequest)
				return
			}

			// Parsea el cuerpo de la solicitud en json
			var person PersonData
			if err := json.NewDecoder(r.Body).Decode(&person); err != nil {
				errJsonStatus(w, fmt.Sprintf(`Error al parsear el cuerpo de la solicitud: %v`, err), http.StatusBadRequest)
				return
			}

			// Verifica que el ID no sea cero
			if person.ID == 0 {
				errJsonStatus(w, `El campo id es requerido`, http.StatusBadRequest)
				return
			}

			// SQL para actualizar una persona
			query := `UPDATE persons SET dni = $1, nombre = $2, apellidos = $3, email = $4, telefono = $5 WHERE id = $6;`
			_, err = db.Exec(query, person.Dni, person.Nombre, person.Apellidos, person.Email, person.Telefono, person.ID)
			if err != nil {
				errJsonStatus(w, fmt.Sprintf(`Error al actualizar la persona: %v`, err), http.StatusInternalServerError)
				return
			}

			// Responde con un mensaje en formato JSON
			w.Write([]byte(`{"message": "Persona actualizada"}`))

			return

		} else if r.Method == http.MethodDelete {

			if iid == 0 {
				errJsonStatus(w, `El campo id es requerido para eliminar`, http.StatusBadRequest)
				return
			}

			// Parsea el ID de la persona
			id := r.URL.Query().Get("id")
			if id == "" {
				errJsonStatus(w, `El campo id es requerido`, http.StatusBadRequest)
				return
			}

			// SQL para eliminar una persona
			query := `DELETE FROM persons WHERE id = $1;`
			_, err = db.Exec(query, id)
			if err != nil {
				errJsonStatus(w, fmt.Sprintf(`Error al eliminar la persona: %v`, err), http.StatusInternalServerError)
				return
			}

			// Responde con un mensaje en formato JSON
			w.Write([]byte(`{"message": "Persona eliminada"}`))

			return

		}
	}
}

func postgres_person_by_id(db *sql.DB, id int) (*PersonData, error) {

	query := fmt.Sprintf(`SELECT id, dni, nombre, apellidos, email, telefono, created_at FROM persons WHERE id = %d;`, id)
	stmt, err := db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	row, err := stmt.Query()
	if err != nil {
		return nil, err
	}
	defer row.Close()

	person := &PersonData{}
	if row.Next() {
		if err := row.Scan(&person.ID, &person.Dni, &person.Nombre, &person.Apellidos, &person.Email, &person.Telefono, &person.CreatedAt); err != nil {
			return person, err
		}
	} else {
		return nil, fmt.Errorf("persona no encontrada con id %d", id)
	}
	return person, nil
}

type PersonPostSent struct {
	Dni       string `json:"dni"`
	Nombre    string `json:"nombre"`
	Apellidos string `json:"apellidos"`
	Email     string `json:"email"`
	Telefono  string `json:"telefono"`
}
