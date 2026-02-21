package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL       string
	SupabaseURL       string
	SupabaseAnonKey   string
	SupabaseJWTSecret string
	ServerPort        string
	OllamaURL         string

	// Derived flags
	HasDatabase bool
	HasSupabase bool
}

func Load() (*Config, error) {
	// NOTE: .env is optional
	// production uses real env vars
	_ = godotenv.Load()

	cfg := &Config{
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		SupabaseURL:       os.Getenv("SUPABASE_URL"),
		SupabaseAnonKey:   os.Getenv("SUPABASE_ANON_KEY"),
		SupabaseJWTSecret: os.Getenv("SUPABASE_JWT_SECRET"),
		ServerPort:        os.Getenv("SERVER_PORT"),
		OllamaURL:         os.Getenv("OLLAMA_URL"),
	}

	if cfg.ServerPort == "" {
		cfg.ServerPort = "8080"
	}
	if cfg.OllamaURL == "" {
		cfg.OllamaURL = "http://localhost:11434"
	}

	// NOTE: Database is optional
	// server runs in chat-only mode without it
	cfg.HasDatabase = cfg.DatabaseURL != ""

	// NOTE: Supabase auth is optional
	cfg.HasSupabase = cfg.SupabaseURL != "" && cfg.SupabaseAnonKey != "" && cfg.SupabaseJWTSecret != ""

	if !cfg.HasDatabase {
		log.Println("DATABASE_URL not set — running in chat-only mode (no DB)")
	}
	if !cfg.HasSupabase {
		log.Println("Supabase credentials not set — auth endpoints disabled")
	}

	return cfg, nil
}
