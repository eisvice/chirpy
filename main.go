package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/eisvice/chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	database *database.Queries
	platform string
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserId	  uuid.NullUUID	`json:"user_id"`
}

const port = "8080"
const filepathRoot = "."

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	apiCfg := apiConfig{fileserverHits: atomic.Int32{}, database: database.New(db), platform: platform}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/healthz", healthHandler)
	mux.HandleFunc("GET /admin/metrics", apiCfg.hitsHandler)
	mux.HandleFunc("POST /admin/reset", apiCfg.resetHandler)
	mux.HandleFunc("POST /api/users", apiCfg.newUserHandler)
	mux.HandleFunc("POST /api/chirps", apiCfg.validateChirpHandler)
	mux.HandleFunc("GET /api/chirps", apiCfg.listChirpsHandler)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.getChirpHandler)

	fs := http.FileServer(http.Dir(filepathRoot))
	handler := http.StripPrefix("/app", fs)
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(handler))

	server := &http.Server{
		Addr: ":" + port,
		Handler: mux,
	}

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(server.ListenAndServe())
}

func healthHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(http.StatusOK)
	_, err := writer.Write([]byte("OK"))
	if err != nil {
		log.Println(err)
	}
}

func (cfg *apiConfig) hitsHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.WriteHeader(http.StatusOK)
	_, err := fmt.Fprintf(writer, `<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileserverHits.Load())
	if err != nil {
		log.Fatal("hitsHandler issues: ", err)
	}
}

func (cfg *apiConfig) resetHandler(writer http.ResponseWriter, request *http.Request) {
	if cfg.platform != "DEV" {
		writer.WriteHeader(http.StatusForbidden)
		return
	}

	err := cfg.database.DeleteUsers(request.Context())
	if err != nil {
		log.Fatalf("couldn't delete users: %v", err)
	}
	writer.WriteHeader(http.StatusOK)
	cfg.fileserverHits.Store(0)
}

func (cfg *apiConfig) newUserHandler(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	type requestBody struct {
		Email string `json:"email"`
	}

	dat, err := io.ReadAll(request.Body)
	if err != nil {
		respondWithError(writer, 500, "couldn't read request!")
		return
	}

	params := requestBody{}
	err = json.Unmarshal(dat, &params)
	if err != nil {
		respondWithError(writer, 500, "couldn't unmarshal parameters")
		return
	}

	user, err := cfg.database.CreateUser(request.Context(), params.Email)
	if err != nil {
		respondWithError(writer, 500, fmt.Sprintf("error while creating a user: %v", err))
		return
	}

	respondWithJSON(writer, 201, &User{
		ID: user.ID, 
		CreatedAt: user.CreatedAt, 
		UpdatedAt: user.UpdatedAt,
		Email: user.Email,
	})
}

func (cfg *apiConfig) validateChirpHandler(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	type requestBody struct {
		Body string `json:"body"`
		UserId uuid.NullUUID `json:"user_id"`
	}

	dat, err := io.ReadAll(request.Body)
	if err != nil {
		respondWithError(writer, 500, "couldn't read request!")
		return
	}

	params := requestBody{}
	err = json.Unmarshal(dat, &params)
	if err != nil {
		respondWithError(writer, 500, "couldn't unmarshal parameters")
		return
	}

	if len(params.Body) > 140 {
		respondWithError(writer, 400, "Chirp is too long")
		return
	}

	chirp, err := cfg.database.CreateChirp(
		request.Context(), 
		database.CreateChirpParams{
			Body: params.Body,
			UserID: params.UserId,
		},
	)

	if err != nil {
		respondWithError(writer, 500, fmt.Sprintf("error while creating a chirp: %v", err))
		return
	}

	respondWithJSON(
		writer,
		201,
		&Chirp{
			ID: chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt, 
			Body: chirp.Body, 
			UserId: chirp.UserID,
		},
	)
}

func (cfg *apiConfig) listChirpsHandler(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	chirps, err := cfg.database.ListChirps(request.Context())
	if err != nil {
		respondWithError(writer, 500, fmt.Sprintf("error while listing chirps: %v", err))
		return
	}

	chirpsMap := make([]Chirp, len(chirps));
	for i, chirp := range chirps {
		chirpsMap[i] = Chirp{
			ID: chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt, 
			Body: chirp.Body, 
			UserId: chirp.UserID,
		}
	}

	respondWithJSON(writer, http.StatusOK, chirpsMap)
}

func (cfg *apiConfig) getChirpHandler(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	chirpUUID, err := uuid.Parse(request.PathValue("chirpID"))
	if err != nil {
		respondWithError(writer, http.StatusInternalServerError, fmt.Sprintf("invalid uuid: %v", err))
	}

	chirp, err := cfg.database.GetChirp(request.Context(), chirpUUID)
	if err != nil {
		respondWithError(writer, http.StatusNotFound, fmt.Sprintf("error while finding a chirp: %v", err))
		return
	}

	respondWithJSON(writer, http.StatusOK, Chirp{
			ID: chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt, 
			Body: chirp.Body, 
			UserId: chirp.UserID,
		},
	)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) error {
    response, err := json.Marshal(payload)
    if err != nil {
        return err
    }
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.WriteHeader(code)
    w.Write(response)
    return nil
}

func respondWithError(w http.ResponseWriter, code int, msg string) error {
	return respondWithJSON(w, code, map[string]string{"error": msg})
}

func bodyCleaning(bodyText string) string {
	profanes := []string{
		"kerfuffle",
		"sharbert",
		"fornax",
	}

	bodySplit := strings.Split(bodyText, " ")

	for i, split := range bodySplit {
		for _, profane := range profanes {
			if strings.ToLower(split) == profane {
				bodySplit[i] = "****"
			}
		}
	}

	return strings.Join(bodySplit, " ")
}
