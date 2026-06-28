// Package http exposes the white_label platform's HTTP surface.
package http

import (
	"encoding/json"
	"net/http"
	"time"

	kerrors "github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/kernel/response"
	"github.com/agile-factory/kernel/platform/white_label/domain"
	"github.com/agile-factory/kernel/platform/white_label/usecase"
)

// BrandHandler handles HTTP requests for the white_label surface.
type BrandHandler struct {
	getUC      *usecase.GetBrandSettingsUseCase
	updateUC   *usecase.UpdateBrandSettingsUseCase
	verifyUC   *usecase.VerifyCustomDomainUseCase
	publicUC   *usecase.GetPublicBrandUseCase
}

// NewBrandHandler wires the handler.
func NewBrandHandler(
	getUC *usecase.GetBrandSettingsUseCase,
	updateUC *usecase.UpdateBrandSettingsUseCase,
	verifyUC *usecase.VerifyCustomDomainUseCase,
	publicUC *usecase.GetPublicBrandUseCase,
) *BrandHandler {
	return &BrandHandler{
		getUC:    getUC,
		updateUC: updateUC,
		verifyUC: verifyUC,
		publicUC: publicUC,
	}
}

// ── DTOs ─────────────────────────────────────────────────────────────────────

type brandResponse struct {
	TenantID               string  `json:"tenantId"`
	AppName                string  `json:"appName"`
	LogoURL                string  `json:"logoUrl,omitempty"`
	FaviconURL             string  `json:"faviconUrl,omitempty"`
	PrimaryColor           string  `json:"primaryColor"`
	SecondaryColor         string  `json:"secondaryColor"`
	AccentColor            string  `json:"accentColor"`
	FontFamily             string  `json:"fontFamily"`
	CustomFontURL          string  `json:"customFontUrl,omitempty"`
	CustomDomain           string  `json:"customDomain,omitempty"`
	CustomDomainStatus     string  `json:"customDomainStatus"`
	CustomDomainToken      string  `json:"customDomainToken,omitempty"`
	CustomDomainVerifiedAt *string `json:"customDomainVerifiedAt,omitempty"`
	EmailFooterHTML        string  `json:"emailFooterHtml,omitempty"`
	EmailLogoURL           string  `json:"emailLogoUrl,omitempty"`
	TermsURL               string  `json:"termsUrl,omitempty"`
	PrivacyURL             string  `json:"privacyUrl,omitempty"`
	SupportEmail           string  `json:"supportEmail,omitempty"`
	SupportPhone           string  `json:"supportPhone,omitempty"`
	UpdatedAt              string  `json:"updatedAt"`
}

func toBrandResponse(b *domain.BrandSettings) brandResponse {
	resp := brandResponse{
		TenantID:           b.TenantID,
		AppName:            b.AppName,
		LogoURL:            b.LogoURL,
		FaviconURL:         b.FaviconURL,
		PrimaryColor:       b.PrimaryColor,
		SecondaryColor:     b.SecondaryColor,
		AccentColor:        b.AccentColor,
		FontFamily:         b.FontFamily,
		CustomFontURL:      b.CustomFontURL,
		CustomDomain:       b.CustomDomain,
		CustomDomainStatus: string(b.CustomDomainStatus),
		CustomDomainToken:  b.CustomDomainToken,
		EmailFooterHTML:    b.EmailFooterHTML,
		EmailLogoURL:       b.EmailLogoURL,
		TermsURL:           b.TermsURL,
		PrivacyURL:         b.PrivacyURL,
		SupportEmail:       b.SupportEmail,
		SupportPhone:       b.SupportPhone,
		UpdatedAt:          b.UpdatedAt.Format(time.RFC3339),
	}
	if b.CustomDomainVerifiedAt != nil {
		s := b.CustomDomainVerifiedAt.Format(time.RFC3339)
		resp.CustomDomainVerifiedAt = &s
	}
	return resp
}

// publicBrandResponse omits sensitive fields (token, footer HTML).
type publicBrandResponse struct {
	TenantID       string `json:"tenantId"`
	AppName        string `json:"appName"`
	LogoURL        string `json:"logoUrl,omitempty"`
	FaviconURL     string `json:"faviconUrl,omitempty"`
	PrimaryColor   string `json:"primaryColor"`
	SecondaryColor string `json:"secondaryColor"`
	AccentColor    string `json:"accentColor"`
	FontFamily     string `json:"fontFamily"`
	CustomFontURL  string `json:"customFontUrl,omitempty"`
}

func toPublicBrandResponse(b *domain.BrandSettings) publicBrandResponse {
	return publicBrandResponse{
		TenantID:       b.TenantID,
		AppName:        b.AppName,
		LogoURL:        b.LogoURL,
		FaviconURL:     b.FaviconURL,
		PrimaryColor:   b.PrimaryColor,
		SecondaryColor: b.SecondaryColor,
		AccentColor:    b.AccentColor,
		FontFamily:     b.FontFamily,
		CustomFontURL:  b.CustomFontURL,
	}
}

// ── Handlers ─────────────────────────────────────────────────────────────────

// GetBrand handles GET /api/v1/tenants/{id}/brand
func (h *BrandHandler) GetBrand(w http.ResponseWriter, r *http.Request) {
	out, err := h.getUC.Execute(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, toBrandResponse(out.Brand))
}

type patchBrandRequest struct {
	AppName         string `json:"appName"`
	LogoURL         string `json:"logoUrl"`
	FaviconURL      string `json:"faviconUrl"`
	PrimaryColor    string `json:"primaryColor"`
	SecondaryColor  string `json:"secondaryColor"`
	AccentColor     string `json:"accentColor"`
	FontFamily      string `json:"fontFamily"`
	CustomFontURL   string `json:"customFontUrl"`
	CustomDomain    string `json:"customDomain"`
	EmailFooterHTML string `json:"emailFooterHtml"`
	EmailLogoURL    string `json:"emailLogoUrl"`
	TermsURL        string `json:"termsUrl"`
	PrivacyURL      string `json:"privacyUrl"`
	SupportEmail    string `json:"supportEmail"`
	SupportPhone    string `json:"supportPhone"`
}

// UpdateBrand handles PATCH /api/v1/tenants/{id}/brand
func (h *BrandHandler) UpdateBrand(w http.ResponseWriter, r *http.Request) {
	var body patchBrandRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, kerrors.BadRequest("invalid JSON body"))
		return
	}

	brand, err := h.updateUC.Execute(r.Context(), usecase.UpdateBrandSettingsInput{
		AppName:         body.AppName,
		LogoURL:         body.LogoURL,
		FaviconURL:      body.FaviconURL,
		PrimaryColor:    body.PrimaryColor,
		SecondaryColor:  body.SecondaryColor,
		AccentColor:     body.AccentColor,
		FontFamily:      body.FontFamily,
		CustomFontURL:   body.CustomFontURL,
		CustomDomain:    body.CustomDomain,
		EmailFooterHTML: body.EmailFooterHTML,
		EmailLogoURL:    body.EmailLogoURL,
		TermsURL:        body.TermsURL,
		PrivacyURL:      body.PrivacyURL,
		SupportEmail:    body.SupportEmail,
		SupportPhone:    body.SupportPhone,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toBrandResponse(brand))
}

// VerifyDomain handles POST /api/v1/tenants/{id}/brand/verify-domain
func (h *BrandHandler) VerifyDomain(w http.ResponseWriter, r *http.Request) {
	if err := h.verifyUC.Execute(r.Context()); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusNoContent, nil)
}

// GetPublicBrand handles GET /public/v1/brand (unauthenticated)
// Lookup: Host header → ?tenant_id= query param.
func (h *BrandHandler) GetPublicBrand(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	if host != "" {
		out, err := h.publicUC.ExecuteByDomain(r.Context(), host)
		if err == nil {
			response.JSON(w, http.StatusOK, toPublicBrandResponse(out.Brand))
			return
		}
	}

	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		response.Error(w, kerrors.BadRequest("tenant_id query parameter or Host header required"))
		return
	}

	out, err := h.publicUC.ExecuteByTenantID(r.Context(), tenantID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, toPublicBrandResponse(out.Brand))
}
