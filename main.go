package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
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
	http.HandleFunc("/personapp-session/", withLogging(corsMiddleware(withAuth(personAppSessionHandler(connStr), auth_token))))
	http.HandleFunc("/authini/", withLogging(corsMiddleware(withAuth(authIniHandler(connStr), auth_token))))

	http.HandleFunc("/init", withLogging(initTablesHandler(connStr)))
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

	// Inicializa las tablas de la base de datos
	// TODO: comentar cuando no se necesite
	go func() {
		initTables(connStr)
	}()
}

func getAuthHandler(w http.ResponseWriter, r *http.Request) {
	// la cache en el cliente podría ser de dos minutos
	// w.Header().Set("Cache-Control", "public, max-age=120")
	w.Write([]byte(`{"status": "success"}`))
}

// middleware para autenticación
func withAuth(handler http.HandlerFunc, auth_token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verificar si el token de autenticación es correcto
		if r.Header.Get("Authorization") != "Bearer "+auth_token {
			//fmt.Println("withAuth No autorizado")
			http.Error(w, `{"error": "No autorizado"}`, http.StatusUnauthorized)
			return
		}

		// Ejecutar el manejador original
		handler(w, r)
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

func errJsonStatus(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	data := map[string]string{"error": msg}
	json.NewEncoder(w).Encode(data)
}
