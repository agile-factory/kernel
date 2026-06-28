// Package user is the user-profile platform. It manages UserProfile records
// (display name, avatar, locale, timezone) separately from auth.User (identity
// + credentials), following RFC-0009 single-responsibility separation.
//
// Wiring:
//
//	userRepo := userinfra.NewPostgresUserProfileRepository(db)
//	checker  := permission.New(memberRepo)
//	userMod  := user.New(user.Config{Repo: userRepo, Checker: checker, Logger: log})
//	userMod.Subscribe(bus)              // creates profiles on auth.signup_owner events
//	userMod.RegisterRoutes(mux)
package user

import (
	"context"
	"embed"
	"errors"
	"log/slog"
	"net/http"

	"github.com/agile-factory/kernel/kernel/events"
	"github.com/agile-factory/kernel/kernel/permission"
	userhttp "github.com/agile-factory/kernel/platform/user/delivery/http"
	"github.com/agile-factory/kernel/platform/user/domain"
	"github.com/agile-factory/kernel/platform/user/usecase"
)

// Migrations holds the SQL files this platform owns. Consumed by migrator.Run.
//
//go:embed migrations/*.sql
var Migrations embed.FS

// Config captures the dependencies of the user Module.
type Config struct {
	Repo    domain.UserProfileRepository
	Checker permission.Checker
	Logger  *slog.Logger
}

// Module is the self-contained user profile platform.
type Module struct {
	cfg      Config
	getUC    *usecase.GetProfileUseCase
	updateUC *usecase.UpdateProfileUseCase
	handler  *userhttp.UserHandler
}

// New wires the user module.
func New(cfg Config) (*Module, error) {
	if cfg.Repo == nil {
		return nil, errors.New("user.Config: Repo is required")
	}
	if cfg.Checker == nil {
		return nil, errors.New("user.Config: Checker is required")
	}
	if cfg.Logger == nil {
		return nil, errors.New("user.Config: Logger is required")
	}

	getUC := usecase.NewGetProfileUseCase(cfg.Repo)
	updateUC := usecase.NewUpdateProfileUseCase(cfg.Repo)

	return &Module{
		cfg:      cfg,
		getUC:    getUC,
		updateUC: updateUC,
		handler:  userhttp.NewUserHandler(getUC, updateUC, cfg.Checker),
	}, nil
}

// Subscribe registers handlers on the outbox bus. Call before bus.Run.
//
// Handled events:
//   - auth.signup_owner → creates the owner's profile with email-derived DisplayName
func (m *Module) Subscribe(bus events.Subscriber) {
	bus.Subscribe("auth.signup_owner", m.handleSignupOwner)
}

// RegisterRoutes mounts the user profile endpoints on mux.
//
//	GET  /api/v1/users/{userId}/profile
//	PATCH /api/v1/users/{userId}/profile
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/users/{userId}/profile", m.handler.GetProfile)
	mux.HandleFunc("PATCH /api/v1/users/{userId}/profile", m.handler.UpdateProfile)
}

// handleSignupOwner creates a UserProfile when a new owner registers.
// This ensures every user has a profile row from the moment they sign up,
// so GET /profile never returns 404 for the owner.
func (m *Module) handleSignupOwner(ctx context.Context, ev events.Event) error {
	userID, _ := ev.Payload["user_id"].(string)
	email, _ := ev.Payload["email"].(string)
	if userID == "" || email == "" {
		m.cfg.Logger.Warn("user: signup_owner event missing user_id or email, skipping profile creation",
			"event_type", ev.Type)
		return nil
	}

	_, err := m.getUC.Execute(ctx, usecase.GetProfileInput{
		UserID: userID,
		Email:  email,
	})
	if err != nil {
		m.cfg.Logger.Warn("user: failed to create profile on signup", "user_id", userID, "error", err)
		// Return nil so the outbox doesn't retry indefinitely for a non-critical operation.
		return nil
	}
	return nil
}
