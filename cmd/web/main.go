package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"webwallet/internal/handlers"
	"webwallet/internal/repository" // Importujemy pakiet repository
)

func main() {
	// Konfiguracja repozytorium MongoDB
	repoConfig := repository.RepoConfiguration{
		URI:        "mongodb://admin:adminpassword@localhost:27017", // Użyj poświadczeń z docker-compose
		Database:   "portfolio_db",
		Collection: "portfolios",
	}

	// Inicjalizacja repozytorium
	portfolioRepo, err := repository.NewPortfolioRepo(repoConfig)
	if err != nil {
		log.Fatalf("Failed to initialize portfolio repository: %v", err)
	}
	defer func() {
		// Zamykanie połączenia z DB przy wyłączaniu aplikacji
		log.Println("Disconnecting from MongoDB...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err = portfolioRepo.Disconnect(ctx); err != nil {
			log.Printf("Error disconnecting from MongoDB: %v", err)
		} else {
			log.Println("Disconnected from MongoDB successfully.")
		}
	}()

	// Przekazanie repozytorium do handlera
	// Tworzymy nową instancję handlera z wstrzykniętym repozytorium
	mainHandler := handlers.NewAppHandler(portfolioRepo)

	// Ustawienie handlera dla ścieżki głównej (teraz używamy metody z mainHandler)
	http.HandleFunc("/", mainHandler.HomeHandler)

	http.HandleFunc("/add-asset", mainHandler.AddAssetHandler) // Rejestracja dla GET i POST

	http.HandleFunc("/delete-asset", mainHandler.DeleteAssetHandler)

	http.HandleFunc("/update-asset", mainHandler.UpdateAssetHandler)

	http.HandleFunc("/add-subscription", mainHandler.AddSubscriptionHandler)
	http.HandleFunc("/delete-subscription", mainHandler.DeleteSubscriptionHandler)
	http.HandleFunc("/update-subscription", mainHandler.UpdateSubscriptionHandler)

	// Ustawienie handlera dla statycznych plików
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Graceful shutdown (kontrolowane wyłączanie serwera)
	server := &http.Server{Addr: ":8080"}
	go func() {
		fmt.Println("Serwer uruchomiony na porcie :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on :8080: %v\n", err)
		}
	}()

	// Czekaj na sygnał zakończenia (Ctrl+C)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exiting.")
}
