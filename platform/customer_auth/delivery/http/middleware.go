// Package http provides the HTTP delivery layer for the customer_auth platform.
package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/agile-factory/kernel/platform/customer_auth/usecase"
)

// contextKey is an unexported type for customer_auth context keys.
// Prevents collisions with the staff auth context keys.
type contextKey string

const (
	contextKeyEndCustomerID    contextKey = "end_customer_id"
	contextKeyCustomerTenantID contextKey = "customer_tenant_id"
)

// CustomerRequired is an HTTP middleware that validates the customer Bearer token.
// On success it stores end_customer_id and tenant_id in the request context via
// endCustomerIDFromContext / customerTenantIDFromContext.
func CustomerRequired(validator usecase.CustomerJWTValidator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		endCustomerID, tenantID, err := validator.ValidateToken(token)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		ctx := context.WithValue(r.Context(), contextKeyEndCustomerID, endCustomerID)
		ctx = context.WithValue(ctx, contextKeyCustomerTenantID, tenantID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// EndCustomerIDFromContext returns the end_customer_id stored by CustomerRequired.
func EndCustomerIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(contextKeyEndCustomerID).(string)
	return v
}

// CustomerTenantIDFromContext returns the tenant_id stored by CustomerRequired.
func CustomerTenantIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(contextKeyCustomerTenantID).(string)
	return v
}
