package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"
)

type PropertyType string
type ListingStatus string

const (
	PropertyApartment  PropertyType = "apartment"
	PropertyHouse      PropertyType = "house"
	PropertyStudio     PropertyType = "studio"
	PropertyVilla      PropertyType = "villa"
	PropertyCommercial PropertyType = "commercial"

	ListingDraft     ListingStatus = "draft"
	ListingPublished ListingStatus = "published"
	ListingArchived  ListingStatus = "archived"
	ListingRented    ListingStatus = "rented"
)

type Listing struct {
	ID             uuid.UUID     `json:"id" db:"id"`
	LandlordID     uuid.UUID     `json:"landlord_id" db:"landlord_id"`
	Title          string        `json:"title" db:"title"`
	Description    string        `json:"description" db:"description"`
	PropertyType   PropertyType  `json:"property_type" db:"property_type"`
	Status         ListingStatus `json:"status" db:"status"`
	Address        string        `json:"address" db:"address"`
	City           string        `json:"city" db:"city"`
	PostalCode     *string       `json:"postal_code,omitempty" db:"postal_code"`
	Country        string        `json:"country" db:"country"`
	Location       orb.Point     `json:"location" db:"location"` // [lon, lat]
	Latitude       float64       `json:"latitude"`
	Longitude      float64       `json:"longitude"`
	PricePerMonth  float64       `json:"price_per_month" db:"price_per_month"`
	DepositAmount  float64       `json:"deposit_amount" db:"deposit_amount"`
	SurfaceArea    float64       `json:"surface_area" db:"surface_area"`
	Bedrooms       int           `json:"bedrooms" db:"bedrooms"`
	Bathrooms      int           `json:"bathrooms" db:"bathrooms"`
	Floor          *int          `json:"floor,omitempty" db:"floor"`
	TotalFloors    *int          `json:"total_floors,omitempty" db:"total_floors"`
	Amenities      []string      `json:"amenities" db:"amenities"`
	Images         []ListingImage `json:"images" db:"images"`
	AvailableFrom  time.Time     `json:"available_from" db:"available_from"`
	MinLeaseMonths int           `json:"min_lease_months" db:"min_lease_months"`
	MaxLeaseMonths *int          `json:"max_lease_months,omitempty" db:"max_lease_months"`
	Slug           string        `json:"slug" db:"slug"`
	MetaDescription *string      `json:"meta_description,omitempty" db:"meta_description"`
	ViewCount      int           `json:"view_count" db:"view_count"`
	FavoriteCount  int           `json:"favorite_count" db:"favorite_count"`
	CreatedAt      time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at" db:"updated_at"`
	DeletedAt      *time.Time    `json:"-" db:"deleted_at"`
}

type ListingImage struct {
	URL       string `json:"url"`
	Order     int    `json:"order"`
	IsPrimary bool   `json:"is_primary"`
}

// DTOs
type CreateListingRequest struct {
	Title          string        `json:"title" validate:"required,min=10,max=255"`
	Description    string        `json:"description" validate:"required,min=50"`
	PropertyType   PropertyType  `json:"property_type" validate:"required"`
	Address        string        `json:"address" validate:"required"`
	City           string        `json:"city" validate:"required"`
	PostalCode     *string       `json:"postal_code,omitempty"`
	Latitude       float64       `json:"latitude" validate:"required,min=-90,max=90"`
	Longitude      float64       `json:"longitude" validate:"required,min=-180,max=180"`
	PricePerMonth  float64       `json:"price_per_month" validate:"required,gt=0"`
	DepositAmount  float64       `json:"deposit_amount" validate:"required,gte=0"`
	SurfaceArea    float64       `json:"surface_area" validate:"required,gt=0"`
	Bedrooms       int           `json:"bedrooms" validate:"required,gte=0"`
	Bathrooms      int           `json:"bathrooms" validate:"required,gte=0"`
	Floor          *int          `json:"floor,omitempty"`
	TotalFloors    *int          `json:"total_floors,omitempty"`
	Amenities      []string      `json:"amenities"`
	AvailableFrom  string        `json:"available_from" validate:"required"` // ISO date
	MinLeaseMonths int           `json:"min_lease_months" validate:"required,min=1"`
	MaxLeaseMonths *int          `json:"max_lease_months,omitempty"`
}

type UpdateListingRequest struct {
	Title          *string       `json:"title,omitempty" validate:"omitempty,min=10,max=255"`
	Description    *string       `json:"description,omitempty" validate:"omitempty,min=50"`
	Status         *ListingStatus `json:"status,omitempty"`
	PricePerMonth  *float64      `json:"price_per_month,omitempty" validate:"omitempty,gt=0"`
	DepositAmount  *float64      `json:"deposit_amount,omitempty" validate:"omitempty,gte=0"`
	Amenities      *[]string     `json:"amenities,omitempty"`
	AvailableFrom  *string       `json:"available_from,omitempty"`
	MinLeaseMonths *int          `json:"min_lease_months,omitempty" validate:"omitempty,min=1"`
	MaxLeaseMonths *int          `json:"max_lease_months,omitempty"`
}

type ListingSearchQuery struct {
	City           *string       `query:"city"`
	MinPrice       *float64      `query:"min_price"`
	MaxPrice       *float64      `query:"max_price"`
	MinSurface     *float64      `query:"min_surface"`
	MaxSurface     *float64      `query:"max_surface"`
	Bedrooms       *int          `query:"bedrooms"`
	Bathrooms      *int          `query:"bathrooms"`
	PropertyType   *PropertyType `query:"property_type"`
	Amenities      []string      `query:"amenities"`
	Latitude       *float64      `query:"lat"`
	Longitude      *float64      `query:"lon"`
	RadiusKM       *float64      `query:"radius_km"`
	Page           int           `query:"page" validate:"min=1"`
	Limit          int           `query:"limit" validate:"min=1,max=100"`
	SortBy         string        `query:"sort_by" validate:"omitempty,oneof=price_asc price_desc date_desc date_asc"`
}

type ListingSearchResponse struct {
	Listings   []Listing `json:"listings"`
	TotalCount int       `json:"total_count"`
	Page       int       `json:"page"`
	Limit      int       `json:"limit"`
	TotalPages int       `json:"total_pages"`
}
