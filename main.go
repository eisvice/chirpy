package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

const port = "8080"
const filepathRoot = "."

func main() {
	apiCfg := apiConfig{fileserverHits: atomic.Int32{}}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/healthz", healthHandler)
	mux.HandleFunc("GET /admin/metrics", apiCfg.hitsHandler)
	mux.HandleFunc("POST /admin/reset", apiCfg.resetHandler)
	mux.HandleFunc("POST /api/validate_chirp", validateChirpHandler)

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
	writer.WriteHeader(http.StatusOK)
	cfg.fileserverHits.Store(0)
}

func validateChirpHandler(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	type requestBody struct {
		Body string `json:"body"`
	}

	type responseBody struct {
		CleanedBody string `json:"cleaned_body"`
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

	respondWithJSON(writer, 200, responseBody{
		CleanedBody: bodyCleaning(params.Body),
	})
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