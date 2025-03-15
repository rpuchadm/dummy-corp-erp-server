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

/*
	CREATE TABLE IF NOT EXISTS person_auth_client (
		id SERIAL PRIMARY KEY,
		person_id INT NOT NULL,
		auth_client_id INT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		profile JSONB,
		FOREIGN KEY (person_id) REFERENCES persons(id),
		FOREIGN KEY (auth_client_id) REFERENCES auth_clients(id),
		UNIQUE (person_id, auth_client_id)
	);
*/

type PersonApp struct {
	ID           int       `json:"id"`
	PersonID     int       `json:"person_id"`
	AuthClientId int       `json:"auth_client_id"`
	CreatedAt    time.Time `json:"created_at"`
	Profile      *string   `json:"profile"`
}

func personAppHandler(connStr string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// /personapp/1/2
		path := strings.TrimPrefix(r.URL.Path, "/personapp/")
		split := strings.Split(path, "/")
		if len(split) != 2 {
			errJsonStatus(w, `Se esperan dos parámetros en la URL`, http.StatusBadRequest)
			return
		}
		// Obtiene el ID de la persona
		idPer := split[0]
		// Obtiene el ID de la aplicación
		idApp := split[1]

		// parsear el idPer a int
		iidPer, err := strconv.Atoi(idPer)
		if err != nil {
			errJsonStatus(w, fmt.Sprintf(`Error al parsear el idPer: %v`, err), http.StatusBadRequest)
			return
		}

		// parsear el id a int
		iidApp, err := strconv.Atoi(idApp)
		if err != nil {
			errJsonStatus(w, fmt.Sprintf(`Error al parsear el idApp: %v`, err), http.StatusBadRequest)
			return
		}

		if iidPer == 0 {
			errJsonStatus(w, `El campo idPer es requerido`, http.StatusBadRequest)
			return
		}

		if iidApp == 0 {
			errJsonStatus(w, `El campo idApp es requerido`, http.StatusBadRequest)
			return
		}

		// Abre la conexión a la base de datos
		db, err := openDatabaseConnection(connStr)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		if r.Method == http.MethodGet {

			if iidPer == 0 {
				errJsonStatus(w, `El campo idPer es requerido`, http.StatusBadRequest)
				return
			}

			if iidApp == 0 {
				errJsonStatus(w, `El campo idApp es requerido`, http.StatusBadRequest)
				return
			}

			personApp, err := postgres_personapp_by_person_id_auth_client_id(db, iidPer, iidApp)
			if err != nil {
				errJsonStatus(w, fmt.Sprintf(`Error al obtener la personapp: %v`, err), http.StatusInternalServerError)
				return
			}

			person, err := postgres_person_by_id(db, iidPer)
			if err != nil {
				errJsonStatus(w, fmt.Sprintf(`Error al obtener la persona: %v`, err), http.StatusInternalServerError)
				return
			}

			app, err := postgres_auth_client_by_id(db, iidApp)
			if err != nil {
				errJsonStatus(w, fmt.Sprintf(`Error al obtener la app: %v`, err), http.StatusInternalServerError)
				return
			}

			data := make(map[string]any)
			data["personapp"] = personApp
			data["person"] = person
			data["app"] = app

			// Convierte la personapp a formato JSON
			jsonPersonApp, err := json.Marshal(data)
			if err != nil {
				errJsonStatus(w, fmt.Sprintf(`Error al convertir la personapp a JSON: %v`, err), http.StatusInternalServerError)
				return
			}

			// Responde con la personapp en formato JSON
			w.Header().Set("Content-Type", "application/json")
			w.Write(jsonPersonApp)

			return

			//} else if r.Method == http.MethodPost {

		} else if r.Method == http.MethodPut {

			var personApp PersonApp
			if err := json.NewDecoder(r.Body).Decode(&personApp); err != nil {
				errJsonStatus(w, fmt.Sprintf(`Error al decodificar la personapp: %v`, err), http.StatusBadRequest)
				return
			}

			if personApp.ID == 0 {
				errJsonStatus(w, `El campo personaapp id es requerido`, http.StatusBadRequest)
				return
			}

			if personApp.PersonID == 0 {
				errJsonStatus(w, `El campo person_id es requerido`, http.StatusBadRequest)
				return
			}

			if personApp.AuthClientId == 0 {
				errJsonStatus(w, `El campo auth_client_id es requerido`, http.StatusBadRequest)
				return
			}

			// Actualiza la personapp
			query := fmt.Sprintf(`
				UPDATE
					person_auth_client
				SET
					profile = $1
				WHERE
					person_id = %d AND auth_client_id = %d;`, personApp.PersonID, personApp.AuthClientId)

			stmt, err := db.Prepare(query)
			if err != nil {
				errJsonStatus(w, fmt.Sprintf(`Error al preparar la consulta: %v`, err), http.StatusInternalServerError)
				return
			}
			defer stmt.Close()

			_, err = stmt.Exec(personApp.Profile)
			if err != nil {
				errJsonStatus(w, fmt.Sprintf(`Error al ejecutar la consulta: %v`, err), http.StatusInternalServerError)
				return
			}

			// Responde con la personapp actualizada
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"message":"PersonApp actualizada"}`))

			return

		} else if r.Method == http.MethodDelete {

		} else {
			errJsonStatus(w, `Método no permitido`, http.StatusMethodNotAllowed)
			return
		}

	}
}

func postgres_personapp_by_person_id_auth_client_id(db *sql.DB, iidPer, iidApp int) (*PersonApp, error) {
	// SQL para obtener una PersonApp
	query := fmt.Sprintf(`
		SELECT
			id, person_id, auth_client_id, created_at, profile
		FROM
			person_auth_client
		WHERE
			person_id = %d AND auth_client_id = %d;`, iidPer, iidApp)
	stmt, err := db.Prepare(query)
	if err != nil {
		fmt.Printf(" Error al preparar la consulta: %v\n", err)
		return nil, err
	}
	defer stmt.Close()

	row, err := stmt.Query()
	if err != nil {
		fmt.Printf(" Error al ejecutar la consulta: %v\n", err)
		return nil, err
	}
	defer row.Close()

	// Estructura para almacenar la personapp
	var personApp PersonApp
	if row.Next() {
		if err := row.Scan(&personApp.ID, &personApp.PersonID, &personApp.AuthClientId, &personApp.CreatedAt, &personApp.Profile); err != nil {
			fmt.Printf(" Error al escanear la personapp: %v\n", err)
			return nil, err
		}
	} else {
		fmt.Printf(" PersonApp no encontrada iidPer:%d iidApp:%d\n", iidPer, iidApp)
		return nil, err
	}

	return &personApp, nil
}

func postgres_personapp_by_auth_client_id(db *sql.DB, id_app int) ([]PersonApp, error) {
	query := fmt.Sprintf(`
		SELECT
			id, person_id, auth_client_id, created_at, profile
		FROM
			person_auth_client
		WHERE
			auth_client_id = %d;`, id_app)
	stmt, err := db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []PersonApp
	for rows.Next() {
		var item PersonApp
		if err := rows.Scan(&item.ID, &item.PersonID, &item.AuthClientId, &item.CreatedAt, &item.Profile); err != nil {
			return nil, err
		}
		list = append(list, item)
	}

	return list, nil
}

func postgres_personapp_by_person_id(db *sql.DB, id_person int) ([]PersonApp, error) {
	query := fmt.Sprintf(`
		SELECT
			id, person_id, auth_client_id, created_at, profile
		FROM
			person_auth_client
		WHERE
			person_id = %d;`, id_person)
	stmt, err := db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []PersonApp
	for rows.Next() {
		var item PersonApp
		if err := rows.Scan(&item.ID, &item.PersonID, &item.AuthClientId, &item.CreatedAt, &item.Profile); err != nil {
			return nil, err
		}
		list = append(list, item)
	}

	return list, nil
}
