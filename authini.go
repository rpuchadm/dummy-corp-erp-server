package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

func authIniHandler(connStr string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Verifica que el método sea GET
		if r.Method != http.MethodGet {
			errJsonStatus(w, `Método no permitido`, http.StatusMethodNotAllowed)
			return
		}

		// Obtiene el ID de la aplicación
		path := strings.TrimPrefix(r.URL.Path, "/authini/")
		id := strings.Split(path, "/")[0]

		// Abre la conexión a la base de datos
		var err error
		db, err := openDatabaseConnection(connStr)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		app, err := postgres_auth_client_by_client_id(db, id)
		if err != nil {
			errJsonStatus(w, fmt.Sprintf(`Error al obtener la aplicación: %v`, err), http.StatusInternalServerError)
			return
		}
		if app == nil {
			errJsonStatus(w, fmt.Sprintf(`No se encontró la aplicación con el id: %s`, id), http.StatusNotFound)
			return
		}

		if app.ClientUrlCallback == nil || *app.ClientUrlCallback == "" {
			errJsonStatus(w, `La aplicación no tiene definido un URL de callback`, http.StatusInternalServerError)
			return
		}

		lper, err := postgres_persons_all(db)
		if err != nil {
			errJsonStatus(w, fmt.Sprintf(`Error al obtener las personas: %v`, err), http.StatusInternalServerError)
			return
		}

		if len(lper) == 0 {
			errJsonStatus(w, `No hay personas registradas`, http.StatusNotFound)
			return
		}

		lpersonapp, err := postgres_personapp_by_auth_client_id(db, app.ID)
		if err != nil {
			errJsonStatus(w, fmt.Sprintf(`Error al obtener la personaapp: %v`, err), http.StatusInternalServerError)
			return
		}

		data := make(map[string]any)
		data["application"] = app
		data["lper"] = lper

		if len(lpersonapp) > 0 {
			data["lpersonapp"] = lpersonapp
		}

		jsonData, err := json.Marshal(data)
		if err != nil {
			errJsonStatus(w, fmt.Sprintf(`Error al serializar los datos: %v`, err), http.StatusInternalServerError)
			return
		}

		// Responde con los clientes en formato JSON
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)

	}

}
