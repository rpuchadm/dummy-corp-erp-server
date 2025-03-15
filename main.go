package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

func main() {

	// Cadena de conexión a la base de datos
	connStr, err := databaseConnString()
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println("Cadena de conexión a la base de datos:", connStr)
	}

	// carga el token de autenticación desde una variable de entorno
	auth_token := os.Getenv("AUTH_TOKEN")
	if auth_token == "" {
		log.Fatal("error: La variable de entorno AUTH_TOKEN debe estar definida")
		//} else {
		//	fmt.Println("Token de autenticación:", token)
	}

	//healz check
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// Manejadores de las rutas
	http.HandleFunc("/auth", withLogging(corsMiddleware(withAuth(getAuthHandler, auth_token))))
	http.HandleFunc("/persons", withLogging(corsMiddleware(withAuth(getPersonsHandler(connStr), auth_token))))
	http.HandleFunc("/person/", withLogging(corsMiddleware(withAuth(personHandler(connStr), auth_token))))
	http.HandleFunc("/applications", withLogging(corsMiddleware(withAuth(getAuthClientsHandler(connStr), auth_token))))
	http.HandleFunc("/application/", withLogging(corsMiddleware(withAuth(authClientHandler(connStr), auth_token))))
	http.HandleFunc("/personapp/", withLogging(corsMiddleware(withAuth(personAppHandler(connStr), auth_token))))

	// post json con client_id y client_url
	http.HandleFunc("/authinit", withLogging(corsMiddleware(withAuth(postAuthInitHandler(connStr), auth_token))))
	// token?code=123
	http.HandleFunc("/token", withLogging(corsMiddleware(getAuthTokenHandler(connStr))))
	http.HandleFunc("/profile", withLogging(corsMiddleware(getAuthProfileHandler(connStr))))

	http.HandleFunc("/init", withLogging(initTables(connStr)))
	http.HandleFunc("/clean", withLogging(dropTables(connStr)))
	http.HandleFunc("/status", withLogging(checkTable(connStr)))
	//manejador por defecto 404
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Ruta no encontrada: %s %s", r.Method, r.URL.Path)
		http.Error(w, "Ruta no encontrada", http.StatusNotFound)
	})

	// Inicia el servidor en el puerto 8080
	fmt.Println("Servidor iniciado en :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func getAuthHandler(w http.ResponseWriter, r *http.Request) {
	// la cache en el cliente podría ser de dos minutos
	// w.Header().Set("Cache-Control", "public, max-age=120")
	w.Write([]byte(`{"status": "success"}`))
}

// middleware para autenticación
func withAuth(handler http.HandlerFunc, token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verificar si el token de autenticación es correcto
		if r.Header.Get("Authorization") != "Bearer "+token {
			//fmt.Println("withAuth No autorizado")
			http.Error(w, `{"error": "No autorizado"}`, http.StatusUnauthorized)
			return
		}

		// Ejecutar el manejador original
		handler(w, r)
	}
}

func authHeader(r *http.Request) string {
	authorization := r.Header.Get("Authorization")
	if authorization != "" {
		// quitamos el prefijo Bearer
		return authorization[7:]
	}
	return ""
}

func postAuthInitHandler(connStr string) http.HandlerFunc {
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
		var data map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "Error al parsear el cuerpo de la solicitud: %v"}`, err), http.StatusBadRequest)
			return
		}

		// Verifica que los campos no estén vacíos
		if data["client_id"] == "" || data["client_url"] == "" {
			http.Error(w, `{"error": "Los campos client_id y client_url son requeridos"}`, http.StatusBadRequest)
			return
		}
		if data["user_id"] == "" || data["user_type"] == "" {
			http.Error(w, `{"error": "Los campos user_id y user_type son requeridos"}`, http.StatusBadRequest)
			return
		}

		// se genera un code aleatorio
		code := generaCode()

		// SQL para insertar un mensaje
		var id int
		query := `INSERT INTO auth_sessions (code, json_data) VALUES ($1, $2) RETURNING id;`
		jsonData, err := json.Marshal(data)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "Error al convertir el json: %v"}`, err), http.StatusInternalServerError)
			return
		}
		err = db.QueryRow(query, code, jsonData).Scan(&id)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "Error al insertar el perfil de autenticación: %v"}`, err), http.StatusInternalServerError)
			return
		}

		// Responde con un mensaje en formato JSON
		w.Write([]byte(`{"message": "Perfil de autenticación creado", "code": "` + code + `"}`))
	}
}

func getAuthProfileHandler(connStr string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		authToken := authHeader(r)
		if authToken == "" {
			http.Error(w, `{"error": "No autorizado"}`, http.StatusUnauthorized)
			return
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
			http.Error(w, `{"error": "Método no permitido"}`, http.StatusMethodNotAllowed)
			return
		}

		// SQL para obtener json_data de auth_sessions por token
		query := `SELECT json_data FROM auth_sessions where token = $1;`
		rows, err := db.Query(query, authToken)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "Error al obtener el perfil de autenticación: %v"}`, err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Estructura para almacenar los perfiles de autenticación
		var jsonData string
		if rows.Next() {
			if err := rows.Scan(&jsonData); err != nil {
				http.Error(w, fmt.Sprintf(`{"error": "Error al escanear el perfil de autenticación: %v"}`, err), http.StatusInternalServerError)
				return
			}
		}

		// Responde con los perfiles de autenticación en formato JSON
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(jsonData))
	}
}

func getAuthTokenHandler(connStr string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// a partir del parámetro code se obtiene el id de auth_profile de la URL
		// se genera un token aleatorio y se almacena en auth_sessions
		// y se pone a null el campo code

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

		// se obtiene el parámetro code de la URL
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, `{"error": "El parámetro code es requerido"}`, http.StatusBadRequest)
			return
		}

		// SQL para obtener el token de auth_sessions por code
		query := `SELECT id FROM auth_sessions WHERE code = $1;`
		rows, err := db.Query(query, code)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "Error al obtener el token de autenticación: %v"}`, err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Estructura para almacenar los perfiles de autenticación
		var id int
		if rows.Next() {
			if err := rows.Scan(&id); err != nil {
				http.Error(w, fmt.Sprintf(`{"error": "Error al escanear el token de autenticación: %v"}`, err), http.StatusInternalServerError)
				return
			}
		}

		// si no se encuentra el id se responde con error
		if id == 0 {
			http.Error(w, `{"error": "No se encontró el token de autenticación"}`, http.StatusNotFound)
			return
		}

		// se genera un token aleatorio
		token := generaToken()

		// SQL para actualizar el token de auth_sessions por id
		query = `UPDATE auth_sessions SET token = $1, code = NULL WHERE id = $2;`
		_, err = db.Exec(query, token, id)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "Error al actualizar el token de autenticación: %v"}`, err), http.StatusInternalServerError)
			return
		}

		// Responde con el token de autenticación en formato JSON
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"token": "` + token + `"}`))
	}
}

func generaCode() string {
	return generaRand(32)
}

func generaToken() string {
	return generaRand(255)
}

func generaRand(length int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	sb := strings.Builder{}
	sb.Grow(length)
	for range length {
		sb.WriteByte(letterBytes[r.Intn(len(letterBytes))])
	}
	return sb.String()
}

/*
type AuthProfileData struct {
	ID        int       `json:"id"`
	Code      string    `json:"code"`
	Token     string    `json:"token"`
	JsonData  string    `json:"json_data"`
	CreatedAt time.Time `json:"created_at"`
}
*/

// Middleware para registrar solicitudes HTTP
func withLogging(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Registrar información de la solicitud
		log.Printf("Started %s %s", r.Method, r.URL.Path)

		// Ejecutar el manejador original
		handler(w, r)

		// Registrar información adicional (tiempo de respuesta)
		log.Printf("Completed %s %s in %v", r.Method, r.URL.Path, time.Since(start))
	}
}

func corsMiddleware(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Permitir cualquier origen
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Permitir los métodos GET, POST, PUT, DELETE
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")

		// Permitir los encabezados Authorization y Content-Type
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

		// Si la solicitud es de tipo OPTIONS, terminar aquí
		if r.Method == http.MethodOptions {
			//fmt.Println("corsMiddleware OPTIONS")
			//w.WriteHeader(http.StatusOK)
			return
		}

		// Ejecutar el manejador original
		handler(w, r)
	}
}

func errJsonStatus(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	data := map[string]string{"error": msg}
	json.NewEncoder(w).Encode(data)
}
