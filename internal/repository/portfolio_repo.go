package repository

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"webwallet/internal/models" // Importujemy nasze modele
)

// RepoConfiguration przechowuje konfigurację połączenia z bazą danych
type RepoConfiguration struct {
	URI        string
	Database   string
	Collection string
}

// PortfolioRepo implementuje operacje CRUD dla InvestmentPortfolio.
type PortfolioRepo struct {
	client     *mongo.Client
	collection *mongo.Collection
}

// NewPortfolioRepo tworzy nową instancję PortfolioRepo i łączy się z MongoDB.
func NewPortfolioRepo(config RepoConfiguration) (*PortfolioRepo, error) {
	// Kontekst z timeoutem na połączenie
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel() // Upewnij się, że kontekst zostanie anulowany po zakończeniu funkcji

	// Opcje połączenia
	clientOptions := options.Client().ApplyURI(config.URI)

	// Łączenie z MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Pingowanie bazy danych, aby sprawdzić połączenie
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	log.Println("Connected to MongoDB!")

	collection := client.Database(config.Database).Collection(config.Collection)

	return &PortfolioRepo{
		client:     client,
		collection: collection,
	}, nil
}

// Disconnect zamyka połączenie z bazą danych.
func (r *PortfolioRepo) Disconnect(ctx context.Context) error {
	if r.client == nil {
		return nil
	}
	return r.client.Disconnect(ctx)
}

// SavePortfolio zapisuje lub aktualizuje portfel w bazie danych.
// Na razie zakładamy, że mamy tylko jeden portfel (lub będziemy używać stałego ID).
func (r *PortfolioRepo) SavePortfolio(ctx context.Context, portfolio *models.InvestmentPortfolio) error {
	// Przykład: zawsze zapisujemy portfel z ID "main_portfolio"
	// W prawdziwej aplikacji ID portfela byłoby związane z użytkownikiem
	filter := bson.M{"_id": "main_portfolio"}

	// Opcje aktualizacji: Upsert = true oznacza "wstaw, jeśli nie istnieje"
	opts := options.Update().SetUpsert(true)

	// Aktualizacja dokumentu
	// Używamy $set, aby zaktualizować tylko podane pola, nie nadpisując całego dokumentu
	// (choć w tym przypadku nadpisanie całego dokumentu też byłoby ok).
	// Zauważ, że InvestmentPortfolio jest zapisywany bezpośrednio, MongoDB zajmie się mapowaniem.
	_, err := r.collection.UpdateOne(ctx, filter, bson.M{"$set": portfolio}, opts)
	if err != nil {
		return fmt.Errorf("failed to save portfolio: %w", err)
	}
	return nil
}

// LoadPortfolio ładuje portfel z bazy danych.
func (r *PortfolioRepo) LoadPortfolio(ctx context.Context) (*models.InvestmentPortfolio, error) {
	var portfolio models.InvestmentPortfolio
	filter := bson.M{"_id": "main_portfolio"} // Ładujemy portfel o stałym ID

	err := r.collection.FindOne(ctx, filter).Decode(&portfolio)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Println("No existing portfolio found. Creating a new one.")
			return models.NewInvestmentPortfolio(), nil // Zwróć nowy, pusty portfel
		}
		return nil, fmt.Errorf("failed to load portfolio: %w", err)
	}
	return &portfolio, nil
}
