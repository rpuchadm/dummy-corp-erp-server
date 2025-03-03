package main

import (
	"database/sql"
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

func initTables(connStr string) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		// Abre la conexión a la base de datos
		var err error
		db, err := openDatabaseConnection(connStr)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		err = initTablePersons(db)
		if err != nil {
			http.Error(w, fmt.Sprintf("{\"error\",\"%v\"}", err), http.StatusInternalServerError)
			return
		}
		err = initTableAuthClients(db)
		if err != nil {
			http.Error(w, fmt.Sprintf("{\"error\",\"%v\"}", err), http.StatusInternalServerError)
			return
		}

		err = initTableAuthProfiles(db)
		if err != nil {
			http.Error(w, fmt.Sprintf("{\"error\",\"%v\"}", err), http.StatusInternalServerError)
			return
		}

		// Responde json indicando que las tablas se crearon o ya existían
		w.Write([]byte(`{"message": "Tablas creadas o ya existentes"}`))
	}
}

// initTablePersons crea la tabla "persons" si no existe
func initTablePersons(db *sql.DB) error {

	// SQL para crear la tabla si no existe
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS PERSONS (
			id SERIAL PRIMARY KEY,
			dni VARCHAR(32) NOT NULL,
			nombre VARCHAR(255) NOT NULL,
			apellidos VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL,
			telefono VARCHAR(20),
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT email_check CHECK (position('@' IN email) > 0),
			CONSTRAINT telefono_check CHECK (telefono ~ '^[0-9]+$')
		);`

	// Ejecuta la creación de la tabla
	_, err := db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("error al crear la tabla persons: %v", err)
	}

	return nil
}

// initTableAuthClients crea la tabla "auth_clients" si no existe

// initTableAuthProfilea crea la tabla "auth_profiles" si no existe
// esta tabla se usa para almacenar los perfiles de autenticación
// a los que se accederá por token de manera reiterada
// después de que se acceda una vez por code
func initTableAuthProfiles(db *sql.DB) error {

	// SQL para crear la tabla si no existe
	// code y token son únicos
	// json_data es un campo jsonb que puede almacenar cualquier información adicional
	// created_at es la fecha de creación del registro
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS auth_profiles (
			id SERIAL PRIMARY KEY,
			code VARCHAR(32) UNIQUE,
			token VARCHAR(255) UNIQUE,
			json_data JSONB,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`

	// Ejecuta la creación de la tabla
	_, err := db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("error al crear la tabla auth_profiles: %v", err)
	}

	//CREATE UNIQUE INDEX unique_code ON auth_profiles (code) WHERE code IS NOT NULL;
	//CREATE UNIQUE INDEX unique_token ON auth_profiles (token) WHERE token IS NOT NULL;
	return nil
}

// initTableAuthClients crea la tabla "auth_clients" si no existe
// esta tabla se usa para almacenar los clientes que pueden acceder a la API
func initTableAuthClients(db *sql.DB) error {

	// SQL para crear la tabla si no existe
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS auth_clients (
			id SERIAL PRIMARY KEY,
			client_id VARCHAR(32) NOT NULL,
			client_url VARCHAR(255) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`

	// Ejecuta la creación de la tabla
	_, err := db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("error al crear la tabla auth_clients: %v", err)
	}

	return nil
}

func dropTables(connStr string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Abre la conexión a la base de datos
		var err error
		db, err := openDatabaseConnection(connStr)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		err = dropTablePersons(db)
		if err != nil {
			http.Error(w, fmt.Sprintf("{\"error\",\"%v\"}", err), http.StatusInternalServerError)
			return
		}

		err = dropTableAuthClients(db)
		if err != nil {
			http.Error(w, fmt.Sprintf("{\"error\",\"%v\"}", err), http.StatusInternalServerError)
			return
		}

		err = dropTableAuthProfiles(db)
		if err != nil {
			http.Error(w, fmt.Sprintf("{\"error\",\"%v\"}", err), http.StatusInternalServerError)
			return
		}

		// Responde json indicando que las tablas se eliminaron o no existían
		w.Write([]byte(`{"message": "Tablas eliminadas o no existían"}`))
	}
}

func dropTablePersons(db *sql.DB) error {

	// SQL para eliminar la tabla "persons"
	dropTableSQL := `DROP TABLE IF EXISTS persons;`

	// Ejecuta la eliminación de la tabla
	_, err := db.Exec(dropTableSQL)
	if err != nil {
		return fmt.Errorf("error al eliminar la tabla persons: %v", err)
	}
	return nil
}

func dropTableAuthClients(db *sql.DB) error {

	// SQL para eliminar la tabla "auth_clients"
	dropTableSQL := `DROP TABLE IF EXISTS auth_clients;`

	// Ejecuta la eliminación de la tabla
	_, err := db.Exec(dropTableSQL)
	if err != nil {
		return fmt.Errorf("error al eliminar la tabla auth_clients: %v", err)
	}
	return nil
}

func dropTableAuthProfiles(db *sql.DB) error {

	// SQL para eliminar la tabla "auth_profiles"
	dropTableSQL := `DROP TABLE IF EXISTS auth_profiles;`

	// Ejecuta la eliminación de la tabla
	_, err := db.Exec(dropTableSQL)
	if err != nil {
		return fmt.Errorf("error al eliminar la tabla auth_profiles: %v", err)
	}
	return nil
}

// checkTable verifica si la tabla "persons" existe
func checkTable(connStr string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Abre la conexión a la base de datos
		var err error
		db, err := openDatabaseConnection(connStr)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		// SQL para verificar si la tabla existe
		var exists bool
		query := `SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'persons');`
		err = db.QueryRow(query).Scan(&exists)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error al verificar la tabla: %v", err), http.StatusInternalServerError)
			return
		}

		// Responde json según si la tabla existe
		if exists {
			w.Write([]byte(`{"message": "La tabla 'persons' existe"}`))
		} else {
			w.Write([]byte(`{"message": "La tabla 'persons' no existe"}`))
		}
	}
}

func databaseConnString() (string, error) {
	dbUser := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB")
	dbService := os.Getenv("POSTGRES_SERVICE")

	// Si alguna variable de entorno no está definida, el programa falla
	if dbUser == "" || dbPassword == "" || dbName == "" {
		return "", fmt.Errorf("error: Las variables de entorno POSTGRES_USER, POSTGRES_PASSWORD y POSTGRES_DB deben estar definidas")
	}

	return fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", dbUser, dbPassword, dbService, dbName), nil
}

func openDatabaseConnection(connStr string) (*sql.DB, error) {

	// Conexión a PostgreSQL
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("error: Error al conectar a la base de datos: %v", err)
	}

	// Verifica que la base de datos se pueda acceder
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error: No se pudo conectar a la base de datos: %v", err)
	}

	return db, nil
}

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
	http.HandleFunc("/person", withLogging(corsMiddleware(withAuth(postPersonHandler(connStr), auth_token))))

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
		query := `INSERT INTO auth_profiles (code, json_data) VALUES ($1, $2) RETURNING id;`
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

		// SQL para obtener json_data de auth_profiles por token
		query := `SELECT json_data FROM auth_profiles where token = $1;`
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
		// se genera un token aleatorio y se almacena en auth_profiles
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

		// SQL para obtener el token de auth_profiles por code
		query := `SELECT id FROM auth_profiles WHERE code = $1;`
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

		// SQL para actualizar el token de auth_profiles por id
		query = `UPDATE auth_profiles SET token = $1, code = NULL WHERE id = $2;`
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
	for i := 0; i < length; i++ {
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
