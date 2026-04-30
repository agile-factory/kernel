package bootstrap

import (
	"log/slog"
	"net/http"

	deliveryhttp "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/delivery/http"
	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/config"
	authhttp "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/auth/delivery/http"
	authmock "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/auth/infrastructure"
	authusecase "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/auth/usecase"
)

// Container holds the application's dependencies
type Container struct {
	cfg *config.Config
	log *slog.Logger
	mux *http.ServeMux
}

// NewContainer initializes a new Container with all dependencies.
func NewContainer(cfg *config.Config, log *slog.Logger) *Container {
	c := &Container{
		cfg: cfg,
		log: log,
		mux: deliveryhttp.NewRouter(log),
	}

	c.setupRoutes()

	return c
}

// Router returns the configured HTTP multiplexer wrapped with global middlewares.
func (c *Container) Router() http.Handler {
	return deliveryhttp.ApplyGlobalMiddlewares(c.mux, c.log)
}

// setupRoutes initializes all the application routes.
func (c *Container) setupRoutes() {
	// Simple health check endpoint
	c.mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Auth Module Wire-up
	// Note: these dependencies (repo, hasher, jwt) would normally be instantiated once
	// and passed down, possibly cached in the container struct itself.
	authRepo := authmock.NewPostgresUserRepository()
	hasher := authmock.NewBcryptHasher(0) // Default cost
	jwtService := authmock.NewJWTService(c.cfg.JWTSecret, c.cfg.TokenLifespan)

	registerUC := authusecase.NewRegisterUseCase(authRepo, hasher)
	loginUC := authusecase.NewLoginUseCase(authRepo, hasher, jwtService)

	authHandler := authhttp.NewAuthHandler(registerUC, loginUC)
	authhttp.RegisterRoutes(c.mux, authHandler)

	// TODO: Register modules and platform routes here
}
