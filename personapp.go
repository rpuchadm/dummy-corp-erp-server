package main

import (
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
	ID        int       `json:"id"`
	PersonID  int       `json:"person_id"`
	AppID     int       `json:"app_id"`
	CreatedAt time.Time `json:"created_at"`
	Profile   *string   `json:"profile"`
}

func personAppHandler(connStr string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		///personapp/1/2

		// Obtiene el ID de la persona
		path := strings.TrimPrefix(r.URL.Path, "/personapp/")
		idPer := strings.Split(path, "/")[0]

		// Obtiene el ID de la aplicación
		idApp := strings.Split(path, "/")[1]

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

			// SQL para obtener una PersonApp
			query := fmt.Sprintf(`
				SELECT
					id, person_id, app_id, created_at, profile
				FROM
					person_auth_client
				WHERE
					person_id = %d AND app_id = %d;`, iidPer, iidApp)
			stmt, err := db.Prepare(query)
			if err != nil {
				errJsonStatus(w, fmt.Sprintf(`Error al preparar la consulta %v %v`, query, err), http.StatusInternalServerError)
				return
			}
			defer stmt.Close()

			row, err := stmt.Query()
			if err != nil {
				errJsonStatus(w, fmt.Sprintf(`Error al obtener la personapp: %v`, err), http.StatusInternalServerError)
				return
			}
			defer row.Close()

			// Estructura para almacenar la personapp
			var personApp PersonApp
			if row.Next() {
				if err := row.Scan(&personApp.ID, &personApp.PersonID, &personApp.AppID, &personApp.CreatedAt, &personApp.Profile); err != nil {
					errJsonStatus(w, fmt.Sprintf(`Error al obtener la personapp: %v`, err), http.StatusInternalServerError)
					return
				}
			} else {
				errJsonStatus(w, `PersonApp no encontrada`, http.StatusNotFound)
				return
			}

			// Convierte la personapp a formato JSON
			jsonPersonApp, err := json.Marshal(personApp)
			if err != nil {
				errJsonStatus(w, fmt.Sprintf(`Error al convertir la personapp a JSON: %v`, err), http.StatusInternalServerError)
				return
			}

			// Responde con la personapp en formato JSON
			w.Header().Set("Content-Type", "application/json")
			w.Write(jsonPersonApp)

			return

			//} else if r.Method == http.MethodPost {
		}
	}
}
