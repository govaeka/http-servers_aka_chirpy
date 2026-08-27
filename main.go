package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/govaeka/http-servers_aka_chirpy.git/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	database       *database.Queries
	secret         string
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
	dbURL := os.Getenv("DB_URL")
	log.Println("Connecting with DB_URL:", dbURL)
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}

	dbQueries := database.New(db)
	cfg := apiConfig{
		fileserverHits: atomic.Int32{},
		database:       dbQueries,
		secret:         os.Getenv("JWTSecret"),
	}

	mux := http.NewServeMux()

	fileServDir := http.Dir(".")
	handlerFileServ := http.FileServer(fileServDir)

	strippedHandler := http.StripPrefix("/app", handlerFileServ)

	mux.Handle("/app/", cfg.middlewareMetricsInc(strippedHandler))

	chirpServer := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	mux.HandleFunc("GET /admin/metrics", cfg.hitcountReportingHandler)
	mux.HandleFunc("GET /api/chirps/{chirpID}", cfg.getChirpHandler)
	mux.HandleFunc("GET /api/chirps", cfg.getChirpsHandler)
	mux.HandleFunc("GET /api/healthz", endpointHandler)
	mux.HandleFunc("POST /admin/reset", cfg.hitcountResetHandler)
	mux.HandleFunc("POST /api/chirps", cfg.createChirpHandler)
	mux.HandleFunc("POST /api/login", cfg.loginHandler)
	mux.HandleFunc("POST /api/refresh", cfg.refreshHandler)
	mux.HandleFunc("POST /api/revoke", cfg.revokeHandler)
	mux.HandleFunc("POST /api/users", cfg.createUserHandler)
	mux.HandleFunc("PUT /api/users", cfg.updateUserHandler)

	chirpServer.ListenAndServe()
}
