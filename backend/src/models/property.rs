use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::FromRow;
use uuid::Uuid;
use validator::Validate;

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct Property {
    pub id: Uuid,
    pub organization_id: Uuid,
    pub reference: Option<String>,
    pub property_type: String,
    pub address: String,
    pub city: String,
    pub neighborhood: Option<String>,
    pub surface_m2: Option<f64>,
    pub rooms: Option<i16>,
    pub rent_amount: i64,
    pub charges_amount: i64,
    pub status: String,
    pub description: Option<String>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Deserialize, Validate)]
pub struct CreatePropertyRequest {
    pub reference: Option<String>,

    #[validate(length(min = 1, message = "Le type de bien est requis"))]
    pub property_type: String,

    #[validate(length(min = 1, message = "L'adresse est requise"))]
    pub address: String,

    pub city: Option<String>,
    pub neighborhood: Option<String>,
    pub surface_m2: Option<f64>,
    pub rooms: Option<i16>,

    #[validate(range(min = 0, message = "Le loyer doit être positif"))]
    pub rent_amount: i64,

    pub charges_amount: Option<i64>,
    pub description: Option<String>,
}

#[derive(Debug, Deserialize, Validate)]
pub struct UpdatePropertyRequest {
    pub reference: Option<String>,
    pub property_type: Option<String>,
    pub address: Option<String>,
    pub city: Option<String>,
    pub neighborhood: Option<String>,
    pub surface_m2: Option<f64>,
    pub rooms: Option<i16>,
    pub rent_amount: Option<i64>,
    pub charges_amount: Option<i64>,
    pub status: Option<String>,
    pub description: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct PropertyResponse {
    pub id: Uuid,
    pub reference: Option<String>,
    pub property_type: String,
    pub address: String,
    pub city: String,
    pub neighborhood: Option<String>,
    pub surface_m2: Option<f64>,
    pub rooms: Option<i16>,
    pub rent_amount: i64,
    pub charges_amount: i64,
    pub status: String,
    pub description: Option<String>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

impl From<Property> for PropertyResponse {
    fn from(p: Property) -> Self {
        Self {
            id: p.id,
            reference: p.reference,
            property_type: p.property_type,
            address: p.address,
            city: p.city,
            neighborhood: p.neighborhood,
            surface_m2: p.surface_m2,
            rooms: p.rooms,
            rent_amount: p.rent_amount,
            charges_amount: p.charges_amount,
            status: p.status,
            description: p.description,
            created_at: p.created_at,
            updated_at: p.updated_at,
        }
    }
}

#[derive(Debug, Deserialize, Default)]
pub struct PropertyFilter {
    pub status: Option<String>,
    pub city: Option<String>,
    pub property_type: Option<String>,
    pub min_rent: Option<i64>,
    pub max_rent: Option<i64>,
}
