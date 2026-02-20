package server

import (
	"rebel-hacks-tui/internal/config"
	"rebel-hacks-tui/internal/server/handlers"
	"rebel-hacks-tui/internal/server/middleware"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRouter(cfg *config.Config, pool *pgxpool.Pool) *chi.Mux {
	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	auth := &handlers.AuthHandler{
		SupabaseURL:     cfg.SupabaseURL,
		SupabaseAnonKey: cfg.SupabaseAnonKey,
	}

	// Public routes
	r.Get("/health", handlers.Health(pool))
	r.Post("/auth/signup", auth.Signup)
	r.Post("/auth/login", auth.Login)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(cfg.SupabaseJWTSecret))
		// Add authenticated endpoints here
	})

	return r
}
