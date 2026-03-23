use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::FromRow;
use uuid::Uuid;
use validator::Validate;

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct User {
    pub id: Uuid,
    pub organization_id: Uuid,
    pub email: String,
    pub password_hash: String,
    pub role: String,
    pub first_name: String,
    pub last_name: String,
    pub phone: Option<String>,
    pub is_active: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Deserialize, Validate)]
pub struct CreateUserRequest {
    #[validate(email(message = "Adresse email invalide"))]
    pub email: String,

    #[validate(length(min = 8, message = "Le mot de passe doit contenir au moins 8 caractères"))]
    pub password: String,

    #[validate(length(min = 1, message = "Le prénom est requis"))]
    pub first_name: String,

    #[validate(length(min = 1, message = "Le nom est requis"))]
    pub last_name: String,

    pub phone: Option<String>,

    #[validate(length(min = 1, message = "Le nom de l'organisation est requis"))]
    pub organization_name: String,

    #[validate(length(min = 1, message = "Le type d'organisation est requis"))]
    pub organization_type: String,
}

#[derive(Debug, Deserialize, Validate)]
pub struct LoginRequest {
    #[validate(email(message = "Adresse email invalide"))]
    pub email: String,

    #[validate(length(min = 1, message = "Le mot de passe est requis"))]
    pub password: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct UserResponse {
    pub id: Uuid,
    pub email: String,
    pub role: String,
    pub first_name: String,
    pub last_name: String,
    pub phone: Option<String>,
    pub organization_id: Uuid,
    pub is_active: bool,
}

impl From<User> for UserResponse {
    fn from(u: User) -> Self {
        Self {
            id: u.id,
            email: u.email,
            role: u.role,
            first_name: u.first_name,
            last_name: u.last_name,
            phone: u.phone,
            organization_id: u.organization_id,
            is_active: u.is_active,
        }
    }
}

#[derive(Debug, Serialize)]
pub struct AuthResponse {
    pub token: String,
    pub user: UserResponse,
}
