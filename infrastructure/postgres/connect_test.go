package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/johnfarrell/runeplan/infrastructure/postgres"
)

func TestConnect_MissingURL(t *testing.T) {
	_, err := postgres.Connect(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestConnect_Live(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	pool, err := postgres.Connect(context.Background(), url)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}
