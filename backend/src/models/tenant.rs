use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::FromRow;
use uuid::Uuid;
use validator::Validate;

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct Tenant {
    pub id: Uuid,
    pub organization_id: Uuid,
    pub first_name: String,
    pub last_name: String,
    pub phone: String,
    pub email: Option<String>,
    pub cni_number: Option<String>,
    pub profession: Option<String>,
    pub employer: Option<String>,
    pub emergency_contact: Option<String>,
    pub emergency_phone: Option<String>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Deserialize, Validate)]
pub struct CreateTenantRequest {
    #[validate(length(min = 1, message = "Le prénom est requis"))]
    pub first_name: String,

    #[validate(length(min = 1, message = "Le nom est requis"))]
    pub last_name: String,

    #[validate(length(min = 1, message = "Le téléphone est requis"))]
    pub phone: String,

    pub email: Option<String>,
    pub cni_number: Option<String>,
    pub profession: Option<String>,
    pub employer: Option<String>,
    pub emergency_contact: Option<String>,
    pub emergency_phone: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UpdateTenantRequest {
    pub first_name: Option<String>,
    pub last_name: Option<String>,
    pub phone: Option<String>,
    pub email: Option<String>,
    pub cni_number: Option<String>,
    pub profession: Option<String>,
    pub employer: Option<String>,
    pub emergency_contact: Option<String>,
    pub emergency_phone: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct TenantResponse {
    pub id: Uuid,
    pub first_name: String,
    pub last_name: String,
    pub phone: String,
    pub email: Option<String>,
    pub cni_number: Option<String>,
    pub profession: Option<String>,
    pub employer: Option<String>,
    pub emergency_contact: Option<String>,
    pub emergency_phone: Option<String>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

impl From<Tenant> for TenantResponse {
    fn from(t: Tenant) -> Self {
        Self {
            id: t.id,
            first_name: t.first_name,
            last_name: t.last_name,
            phone: t.phone,
            email: t.email,
            cni_number: t.cni_number,
            profession: t.profession,
            employer: t.employer,
            emergency_contact: t.emergency_contact,
            emergency_phone: t.emergency_phone,
            created_at: t.created_at,
            updated_at: t.updated_at,
        }
    }
}
