package server

import (
	"net/http"

	"github.com/choice404/vegas-protocol/vegas-tui/internal/config"
	"github.com/choice404/vegas-protocol/vegas-tui/internal/server/handlers"
	"github.com/choice404/vegas-protocol/vegas-tui/internal/server/middleware"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRouter(cfg *config.Config, pool *pgxpool.Pool) *chi.Mux {
	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	chat := &handlers.ChatHandler{
		OllamaURL: cfg.OllamaURL,
	}

	// Always available
	r.Post("/api/chat", chat.Chat)

	// NOTE: Health check
	// works with or without DB
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if pool != nil {
			if err := pool.Ping(r.Context()); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte(`{"status":"unhealthy","db":"disconnected","chat":"ok"}`))
				return
			}
			w.Write([]byte(`{"status":"ok","db":"connected","chat":"ok"}`))
		} else {
			w.Write([]byte(`{"status":"ok","db":"not configured","chat":"ok"}`))
		}
	})

	// NOTE: Auth + protected routes
	// only if Supabase is configured
	if cfg.HasSupabase {
		auth := &handlers.AuthHandler{
			SupabaseURL:     cfg.SupabaseURL,
			SupabaseAnonKey: cfg.SupabaseAnonKey,
		}
		r.Post("/auth/signup", auth.Signup)
		r.Post("/auth/login", auth.Login)

		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.SupabaseJWTSecret))
			// Add authenticated endpoints here
		})
	}

	return r
}
