package httpserver

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"

	apiHandler "sourceink/backend/internal/api/handler"
	apiService "sourceink/backend/internal/api/service"
	"sourceink/backend/internal/articles"
	"sourceink/backend/internal/auth"
	"sourceink/backend/internal/config"
	"sourceink/backend/internal/database"
	"sourceink/backend/internal/github"
)

func New(cfg config.Config, db *sqlx.DB, logger *slog.Logger) (http.Handler, *articles.ArticlesWorker, error) {
	store := database.NewStore(db)

	authHandler, err := auth.NewHandler(store, logger, cfg.CookieSecure, cfg.SessionTTL)
	if err != nil {
		return nil, nil, err
	}

	githubClient, err := github.NewClient(github.ClientConfig{
		AppJWT:       cfg.GitHubAppJWT,
		APIBaseURL:   cfg.GitHubPublisherAPIBaseURL,
		ClientID:     cfg.GitHubPublisherClientID,
		ClientSecret: cfg.GitHubPublisherClientSecret,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("initialize github client: %w", err)
	}

	githubService := github.NewService(store, githubClient)
	githubHandler := github.NewHandler(
		githubService,
		cfg.GitHubPublisherAppSlug,
		cfg.AppOrigin,
		cfg.GitHubPublisherClientID,
		strings.TrimRight(cfg.AppOrigin, "/")+"/api/github/callback",
		cfg.CookieSecure,
		cfg.SessionTTL,
		logger,
	)

	articlesService := articles.NewService(store, githubClient)
	articlesWorker := articles.NewArticlesWorker(articlesService)
	githubWebhookHandler := github.NewWebhookHandler(cfg.GitHubWebhookSecret, store, articlesWorker.SyncRepository, logger)
	apiService := apiService.NewService(store, articlesService)
	apiHandler := apiHandler.NewHandler(&apiService, cfg.CookieSecure, logger)

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{\"status\":\"ok\"}\n"))
	})
	router.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte("{\"status\":\"ready\"}\n"))
	})
	router.Route("/api", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/email", authHandler.ConnectEmail)
			r.Get("/me", authHandler.Me)
			r.Post("/logout", authHandler.Logout)
			r.Get("/github", githubHandler.LoginRedirect)
		})
		r.Route("/github", func(r chi.Router) {
			r.Get("/connect", githubHandler.ConnectAccountRedirect)
			r.Get("/install", githubHandler.InstallationRedirect)
			r.Get("/setup", githubHandler.SetupInstallation)
			r.Get("/callback", githubHandler.ValidateOAuthCallback)
			r.Post("/webhook", githubWebhookHandler.ServeHTTP)
		})
		r.Get("/repos", apiHandler.ListRepositories)
		r.Get("/articles", apiHandler.ListArticles)
		r.Post("/articles/{draftID}/publish", apiHandler.PublishArticle)
		r.Get("/published-articles", apiHandler.ListPublishedArticles)
		r.Get("/published-articles/{articleID}/reviews", apiHandler.GetReviewSummary)
		r.Put("/published-articles/{articleID}/review", apiHandler.CreateReview)
		r.Delete("/published-articles/{articleID}/review", apiHandler.DeleteReview)
	})

	handler := securityHeaders(router)
	handler = enforceOrigin(cfg.AppOrigin, handler)
	handler = recoverPanics(logger, handler)
	handler = logRequests(logger, handler)
	return handler, articlesWorker, nil
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func enforceOrigin(appOrigin string, next http.Handler) http.Handler {
	allowed := strings.TrimRight(appOrigin, "/")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			origin := strings.TrimRight(r.Header.Get("Origin"), "/")
			if origin != "" && origin != allowed {
				http.Error(w, "forbidden origin", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func recoverPanics(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if value := recover(); value != nil {
				logger.Error("request panic", "value", value, "stack", string(debug.Stack()))
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func logRequests(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}
		started := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(started))
	})
}
