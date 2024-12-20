package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// initTable crea la tabla "persons" si no existe
func initTable(connStr string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Abre la conexión a la base de datos
		var err error
		db, err := openDatabaseConnection(connStr)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

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
		_, err = db.Exec(createTableSQL)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error al crear la tabla: %v", err), http.StatusInternalServerError)
			return
		}

		// Responde json indicando que la tabla se creó o ya existe
		w.Write([]byte(`{"message": "Tabla 'persons' creada o ya existe"}`))
	}
}

func dropTable(connStr string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Abre la conexión a la base de datos
		var err error
		db, err := openDatabaseConnection(connStr)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		// SQL para eliminar la tabla "persons"
		dropTableSQL := `DROP TABLE IF EXISTS persons;`

		// Ejecuta la eliminación de la tabla
		_, err = db.Exec(dropTableSQL)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error al eliminar la tabla: %v", err), http.StatusInternalServerError)
			return
		}

		// Responde json indicando que la tabla se eliminó o no existía
		w.Write([]byte(`{"message": "Tabla 'persons' eliminada o no existía"}`))
	}
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
	token := os.Getenv("AUTH_TOKEN")
	if token == "" {
		log.Fatal("error: La variable de entorno AUTH_TOKEN debe estar definida")
		//} else {
		//	fmt.Println("Token de autenticación:", token)
	}

	//healz check
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// Manejadores de las rutas
	http.HandleFunc("/auth", withLogging(corsMiddleware(withAuth(getAuthHandler, token))))
	http.HandleFunc("/persons", withLogging(corsMiddleware(withAuth(getPersonsHandler(connStr), token))))
	http.HandleFunc("/person", withLogging(corsMiddleware(withAuth(postPersonHandler(connStr), token))))
	http.HandleFunc("/init", withLogging(initTable(connStr)))
	http.HandleFunc("/clean", withLogging(dropTable(connStr)))
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
	// la cache en el cliente puede ser de dos minutos
	//w.Header().Set("Cache-Control", "public, max-age=120")
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
