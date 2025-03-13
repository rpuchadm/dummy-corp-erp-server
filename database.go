package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
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
