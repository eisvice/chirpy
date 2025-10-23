package main

import (
	"log"
	"net/http"
)

const port = "8080"
const filepathRoot = "."

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", healthHandler)

	fs := http.FileServer(http.Dir(filepathRoot))
	mux.Handle("/app/", http.StripPrefix("/app", fs))

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
	num, err := writer.Write([]byte("OK"))
	if err != nil {
		log.Println(err)
	}
	log.Printf("Some number: %d", num)
	log.Println(request)
}