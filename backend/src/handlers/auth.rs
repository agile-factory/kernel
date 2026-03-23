use std::sync::Arc;

use axum::{extract::State, Json};
use uuid::Uuid;

use crate::errors::AppError;
use crate::middleware::auth::AuthUser;
use crate::models::user::{AuthResponse, CreateUserRequest, LoginRequest, UserResponse};
use crate::repositories::user_repo;
use crate::services::auth as auth_service;
use crate::utils::response::ApiResponse;
use crate::utils::validator::validate_request;
use crate::AppState;

pub async fn register(
    State(state): State<Arc<AppState>>,
    Json(body): Json<CreateUserRequest>,
) -> Result<Json<ApiResponse<AuthResponse>>, AppError> {
    validate_request(&body)?;

    // Check if user already exists
    if user_repo::find_by_email(&state.db, &body.email).await?.is_some() {
        return Err(AppError::Conflict("Un utilisateur avec cet email existe deja".to_string()));
    }

    let password_hash = auth_service::hash_password(&body.password)?;

    // Create organization
    let org = sqlx::query_as::<_, OrgRow>(
        "INSERT INTO organizations (name, org_type) VALUES ($1, $2) RETURNING id, name"
    )
    .bind(&body.organization_name)
    .bind(&body.organization_type)
    .fetch_one(&state.db)
    .await?;

    // Create user
    let user = user_repo::create(
        &state.db,
        org.id,
        &body.email,
        &password_hash,
        "admin",
        &body.first_name,
        &body.last_name,
        body.phone.as_deref(),
    )
    .await?;

    let token = auth_service::generate_token(
        &user.id,
        &user.organization_id,
        &state.config.jwt_secret,
        state.config.jwt_expiration_hours,
    )?;

    let response = AuthResponse {
        token,
        user: UserResponse::from(user),
    };

    Ok(Json(ApiResponse::success(response)))
}

pub async fn login(
    State(state): State<Arc<AppState>>,
    Json(body): Json<LoginRequest>,
) -> Result<Json<ApiResponse<AuthResponse>>, AppError> {
    validate_request(&body)?;

    let user = user_repo::find_by_email(&state.db, &body.email)
        .await?
        .ok_or_else(|| AppError::Unauthorized("Email ou mot de passe incorrect".to_string()))?;

    if !user.is_active {
        return Err(AppError::Unauthorized("Compte desactive".to_string()));
    }

    let valid = auth_service::verify_password(&body.password, &user.password_hash)?;
    if !valid {
        return Err(AppError::Unauthorized("Email ou mot de passe incorrect".to_string()));
    }

    let token = auth_service::generate_token(
        &user.id,
        &user.organization_id,
        &state.config.jwt_secret,
        state.config.jwt_expiration_hours,
    )?;

    let response = AuthResponse {
        token,
        user: UserResponse::from(user),
    };

    Ok(Json(ApiResponse::success(response)))
}

pub async fn refresh(
    State(state): State<Arc<AppState>>,
    auth_user: AuthUser,
) -> Result<Json<ApiResponse<AuthResponse>>, AppError> {
    let user = user_repo::find_by_id(&state.db, auth_user.user_id)
        .await?
        .ok_or_else(|| AppError::NotFound("Utilisateur non trouve".to_string()))?;

    let token = auth_service::generate_token(
        &user.id,
        &user.organization_id,
        &state.config.jwt_secret,
        state.config.jwt_expiration_hours,
    )?;

    let response = AuthResponse {
        token,
        user: UserResponse::from(user),
    };

    Ok(Json(ApiResponse::success(response)))
}

pub async fn me(
    State(state): State<Arc<AppState>>,
    auth_user: AuthUser,
) -> Result<Json<ApiResponse<UserResponse>>, AppError> {
    let user = user_repo::find_by_id(&state.db, auth_user.user_id)
        .await?
        .ok_or_else(|| AppError::NotFound("Utilisateur non trouve".to_string()))?;

    Ok(Json(ApiResponse::success(UserResponse::from(user))))
}

#[derive(Debug, sqlx::FromRow)]
struct OrgRow {
    id: Uuid,
    #[allow(dead_code)]
    name: String,
}
