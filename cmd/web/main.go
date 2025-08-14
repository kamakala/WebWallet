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
	"webwallet/internal/middleware"
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
	// Tworzymy multiplexer (mux), który będzie zarządzał routingiem.
	mux := http.NewServeMux()

	// Ustawienie handlera dla ścieżki głównej (teraz używamy metody z mainHandler)
	mux.HandleFunc("/", mainHandler.HomeHandler)

	mux.HandleFunc("/add-asset", mainHandler.AddAssetHandler) // Rejestracja dla GET i POST

	mux.HandleFunc("/delete-asset", mainHandler.DeleteAssetHandler)

	mux.HandleFunc("/update-asset", mainHandler.UpdateAssetHandler)
	mux.HandleFunc("/update-price", mainHandler.UpdateAssetPriceHandler)
	mux.HandleFunc("/add-subscription", mainHandler.AddSubscriptionHandler)
	mux.HandleFunc("/delete-subscription", mainHandler.DeleteSubscriptionHandler)
	mux.HandleFunc("/update-subscription", mainHandler.UpdateSubscriptionHandler)
	mux.HandleFunc("/update-wallet-type", mainHandler.UpdateWalletTypeHandler)

	mux.HandleFunc("/visualizations", mainHandler.VisualizationsHandler)            // Nowa podstrona
	mux.HandleFunc("/visualizations/data", mainHandler.GetVisualizationDataHandler) // Endpoint HTMX
	mux.HandleFunc("/toggle-theme", mainHandler.ThemeToggleHandler)

	// Ustawienie handlera dla statycznych plików
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	themedMux := middleware.ThemeMiddleware(mux)

	// Graceful shutdown (kontrolowane wyłączanie serwera)
	server := &http.Server{Addr: ":8080", Handler: themedMux}
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
