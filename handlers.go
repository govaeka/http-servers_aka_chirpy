package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/govaeka/http-servers_aka_chirpy.git/internal/auth"
	"github.com/govaeka/http-servers_aka_chirpy.git/internal/database"

	_ "github.com/lib/pq"
)

func endpointHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) createUserHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	if err := decoder.Decode(&params); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	hashPw, err := auth.HashPassword(params.Password)
	if err != nil {
		log.Printf("Password hashing failed")
	}

	var userParams database.CreateUserParams
	userParams.Email = params.Email
	userParams.HashedPassword = hashPw
	ctx := r.Context()
	dbUser, err := cfg.database.CreateUser(ctx, userParams)
	if err != nil {
		log.Printf("CreateUser failed: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Could not create user")
		return
	}

	type User struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
	}

	user := User{
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
	}

	respondWithJSON(w, http.StatusCreated, user)
}

func (cfg *apiConfig) createChirpHandler(w http.ResponseWriter, r *http.Request) {

	/// DECODE /////
	type parameters struct {
		Body   string        `json:"body"`
		UserID uuid.NullUUID `json:"user_id"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		respondWithError(w, 500, "Something went wrong")
		return
	}

	//// CHECK TOKEN
	tokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		fmt.Printf("could not get token: %v", err)
		return
	}
	IdForUser, err := auth.ValidateJWT(tokenString, os.Getenv("JWTSecret"))
	if err != nil {
		fmt.Printf("validity check failed: %v", err)
		respondWithError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	/// VALIDATE / CLEAN ///////
	if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}
	fmt.Printf("decoded params: %v", params)

	type validationMsg struct {
		Valid       bool   `json:"valid"`
		CleanedBody string `json:"cleaned_body"`
	}

	cleanResponse := validationMsg{}
	cleanResponse.CleanedBody = profanityBleeped(params.Body)
	cleanResponse.Valid = true

	/// CREATE CHIRP FOR DATABASE ///////
	ctx := r.Context()
	var newChirp database.CreateChirpParams
	newChirp.Body = cleanResponse.CleanedBody
	newChirp.UserID = uuid.NullUUID{
		UUID:  IdForUser,
		Valid: true,
	}

	finalResponse, err := cfg.database.CreateChirp(ctx, newChirp)

	/// RESPOND WITHOUT JSON /////////
	if err != nil {
		respondWithError(w, 500, "Error saving Chirp to database.")
		return
	}

	/// BUILD JSON RESPONSE STRUCT ///////

	type fullChirp struct {
		ID        uuid.UUID     `json:"id"`
		CreatedAt time.Time     `json:"created_at"`
		UpdatedAt time.Time     `json:"updated_at"`
		Body      string        `json:"body"`
		UserID    uuid.NullUUID `json:"user_id"`
	}
	JSONChirp := fullChirp{
		ID:        finalResponse.ID,
		CreatedAt: finalResponse.CreatedAt,
		UpdatedAt: finalResponse.UpdatedAt,
		Body:      finalResponse.Body,
		UserID:    finalResponse.UserID,
	}

	respondWithJSON(w, http.StatusCreated, JSONChirp)

}

func (cfg *apiConfig) getChirpHandler(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("chirpID")
	id, err := uuid.Parse(idString)
	if err != nil {
		http.Error(w, "error parsing chirp ID", http.StatusBadRequest)
		return
	}

	type fullChirp struct {
		ID        uuid.UUID     `json:"id"`
		CreatedAt time.Time     `json:"created_at"`
		UpdatedAt time.Time     `json:"updated_at"`
		Body      string        `json:"body"`
		UserID    uuid.NullUUID `json:"user_id"`
	}
	ctx := r.Context()
	DbData, err := cfg.database.GetChirp(ctx, id)
	if DbData.ID == uuid.Nil {
		respondWithError(w, 404, "Chirp-ID not found.")
	}
	if err != nil {
		fmt.Printf("Error text: %v", err)
		respondWithError(w, 500, "Error retrieving data from DB")
	}

	var fullData fullChirp
	fullData = fullChirp{
		ID:        DbData.ID,
		CreatedAt: DbData.CreatedAt,
		UpdatedAt: DbData.UpdatedAt,
		Body:      DbData.Body,
		UserID:    DbData.UserID,
	}

	respondWithJSON(w, 200, fullData)
}

func (cfg *apiConfig) getChirpsHandler(w http.ResponseWriter, r *http.Request) {
	type fullChirp struct {
		ID        uuid.UUID     `json:"id"`
		CreatedAt time.Time     `json:"created_at"`
		UpdatedAt time.Time     `json:"updated_at"`
		Body      string        `json:"body"`
		UserID    uuid.NullUUID `json:"user_id"`
	}
	ctx := r.Context()
	DbData, err := cfg.database.GetChirps(ctx)
	if err != nil {
		respondWithError(w, 500, "Error retrieving data from DB")
		return
	}

	var results []fullChirp
	var newC fullChirp

	for i := range DbData {
		newC.ID = DbData[i].ID
		newC.CreatedAt = DbData[i].CreatedAt
		newC.UpdatedAt = DbData[i].UpdatedAt
		newC.Body = DbData[i].Body
		newC.UserID = DbData[i].UserID
		results = append(results, newC)
	}

	respondWithJSON(w, 200, results)
}

func (cfg *apiConfig) hitcountReportingHandler(w http.ResponseWriter, r *http.Request) {
	hits := cfg.fileserverHits.Load()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)
	fmt.Fprintf(w, "<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", hits)
}

func (cfg *apiConfig) hitcountResetHandler(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
}

func (cfg *apiConfig) loginHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	type params struct {
		Password         string `json:"password"`
		Email            string `json:"email"`
		ExpiresInSeconds int    `json:"expires_in_seconds"`
	}
	var creds params
	err := decoder.Decode(&creds)
	if err != nil {
		log.Printf("Error decoding credentials: %v", err)
		return
	}

	ctx := r.Context()
	usr, err := cfg.database.GetUser(ctx, creds.Email)
	if err != nil {
		log.Printf("Error retrieving user from DB.")
		respondWithError(w, 401, "Incorrect email or password.")
		return
	}

	match, err := auth.CheckPasswordHash(creds.Password, usr.HashedPassword)
	if err != nil {
		log.Printf("Error comparing password to hash.")
		respondWithError(w, 401, "Incorrect email or password.")
		return
	}

	if match == false {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	if creds.ExpiresInSeconds == 0 || creds.ExpiresInSeconds > 3600 {
		creds.ExpiresInSeconds = 3600
	}

	token, err := auth.MakeJWT(usr.ID, cfg.secret, time.Duration(creds.ExpiresInSeconds)*time.Second)
	if err != nil {
		log.Printf("error creating token: %v", err)
	}
	var newBody struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
		Token     string    `json:"token"`
	}

	newBody.ID = usr.ID
	newBody.CreatedAt = usr.CreatedAt
	newBody.UpdatedAt = usr.UpdatedAt
	newBody.Email = usr.Email
	newBody.Token = token

	respondWithJSON(w, 200, newBody)

}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// what needs to happen here, on every request?
		cfg.fileserverHits.Add(1)
		// and how do you make sure the original handler still runs?
		next.ServeHTTP(w, r)

	})
}

// Helpers //////////////////////////////

func profanityBleeped(body string) string {

	wordList := strings.Split(body, " ")

	for i := range wordList {
		lowerWord := strings.ToLower(wordList[i])

		if lowerWord == "kerfuffle" || lowerWord == "sharbert" || lowerWord == "fornax" {
			wordList[i] = "****"
		}
	}
	wordString := strings.Join(wordList, " ")
	return wordString
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type errMessage struct {
		Error string `json:"error"`
	}

	log.Printf("Responding with error: %s", msg)
	newResponse := errMessage{}
	newResponse.Error = msg
	data, _ := json.Marshal(newResponse)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(data)
}

func respondWithJSON(w http.ResponseWriter, code int, payload any) {

	data, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Problem converting to JSON: %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(data)

}
