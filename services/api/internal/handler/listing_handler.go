package handler

import (
	"immobilier-saas/api/internal/domain"
	"immobilier-saas/api/internal/middleware"
	"immobilier-saas/api/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type ListingHandler struct {
	listingService *service.ListingService
}

func NewListingHandler(listingService *service.ListingService) *ListingHandler {
	return &ListingHandler{listingService: listingService}
}

// GetListings godoc
// @Summary      Get all listings
// @Description  Get paginated list of listings with optional filters
// @Tags         listings
// @Accept       json
// @Produce      json
// @Param        city query string false "Filter by city"
// @Param        min_price query number false "Minimum price"
// @Param        max_price query number false "Maximum price"
// @Param        min_surface query number false "Minimum surface area"
// @Param        max_surface query number false "Maximum surface area"
// @Param        bedrooms query integer false "Number of bedrooms"
// @Param        bathrooms query integer false "Number of bathrooms"
// @Param        property_type query string false "Property type" Enums(apartment, house, studio, villa, commercial)
// @Param        lat query number false "Latitude for geo search"
// @Param        lon query number false "Longitude for geo search"
// @Param        radius_km query number false "Search radius in kilometers"
// @Param        page query integer false "Page number" default(1)
// @Param        limit query integer false "Items per page" default(20)
// @Param        sort_by query string false "Sort by" Enums(price_asc, price_desc, date_desc, date_asc)
// @Success      200 {object} domain.ListingSearchResponse
// @Failure      400 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /listings [get]
func (h *ListingHandler) GetListings(c *fiber.Ctx) error {
	var query domain.ListingSearchQuery
	if err := c.QueryParser(&query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid query parameters",
		})
	}

	// Set defaults
	if query.Page == 0 {
		query.Page = 1
	}
	if query.Limit == 0 {
		query.Limit = 20
	}

	response, err := h.listingService.SearchListings(c.Context(), &query)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(response)
}

// GetListingByID godoc
// @Summary      Get listing by ID
// @Description  Get detailed information about a specific listing
// @Tags         listings
// @Accept       json
// @Produce      json
// @Param        id path string true "Listing ID"
// @Success      200 {object} domain.Listing
// @Failure      404 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /listings/{id} [get]
func (h *ListingHandler) GetListingByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid listing ID",
		})
	}

	listing, err := h.listingService.GetListingByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Listing not found",
		})
	}

	// Increment view count asynchronously
	go h.listingService.IncrementViewCount(c.Context(), id)

	return c.JSON(listing)
}

// SearchListings godoc
// @Summary      Search listings
// @Description  Advanced search for listings with filters and geo-location
// @Tags         listings
// @Accept       json
// @Produce      json
// @Param        query query domain.ListingSearchQuery false "Search query"
// @Success      200 {object} domain.ListingSearchResponse
// @Failure      400 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /listings/search [get]
func (h *ListingHandler) SearchListings(c *fiber.Ctx) error {
	return h.GetListings(c)
}

// CreateListing godoc
// @Summary      Create listing
// @Description  Create a new property listing (landlord only)
// @Tags         listings
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body domain.CreateListingRequest true "Create listing request"
// @Success      201 {object} domain.Listing
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /listings [post]
func (h *ListingHandler) CreateListing(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return err
	}

	var req domain.CreateListingRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	listing, err := h.listingService.CreateListing(c.Context(), userID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(listing)
}

// UpdateListing godoc
// @Summary      Update listing
// @Description  Update an existing listing (landlord only)
// @Tags         listings
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Listing ID"
// @Param        request body domain.UpdateListingRequest true "Update listing request"
// @Success      200 {object} domain.Listing
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /listings/{id} [put]
func (h *ListingHandler) UpdateListing(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return err
	}

	listingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid listing ID",
		})
	}

	var req domain.UpdateListingRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	listing, err := h.listingService.UpdateListing(c.Context(), userID, listingID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(listing)
}

// DeleteListing godoc
// @Summary      Delete listing
// @Description  Delete a listing (landlord only)
// @Tags         listings
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Listing ID"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /listings/{id} [delete]
func (h *ListingHandler) DeleteListing(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return err
	}

	listingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid listing ID",
		})
	}

	err = h.listingService.DeleteListing(c.Context(), userID, listingID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Listing deleted successfully",
	})
}
