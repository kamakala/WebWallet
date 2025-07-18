package main

import (
	"fmt"
	"log"
	"net/http"
	"webwallet/internal/handlers" // Importujemy nasz pakiet handlers
)

func main() {
	// Ustawienie handlera dla ścieżki głównej
	http.HandleFunc("/", handlers.HomeHandler) // Używamy handlera z pakietu handlers

	// Ustawienie handlera dla statycznych plików
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	fmt.Println("Serwer uruchomiony na porcie :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
