use std::sync::Arc;

use axum::{
    extract::{Path, Query, State},
    Json,
};
use uuid::Uuid;

use crate::errors::AppError;
use crate::middleware::auth::AuthUser;
use crate::models::lease::{CreateLeaseRequest, LeaseResponse, LeaseWithDetails, UpdateLeaseRequest};
use crate::services::leases as lease_service;
use crate::utils::pagination::PaginationParams;
use crate::utils::response::ApiResponse;
use crate::AppState;

pub async fn list(
    State(state): State<Arc<AppState>>,
    auth_user: AuthUser,
    Query(pagination): Query<PaginationParams>,
) -> Result<Json<ApiResponse<Vec<LeaseWithDetails>>>, AppError> {
    let (leases, meta) = lease_service::list_leases(
        &state.db,
        auth_user.organization_id,
        &pagination,
    )
    .await?;

    Ok(Json(ApiResponse::with_pagination(leases, meta)))
}

pub async fn get_by_id(
    State(state): State<Arc<AppState>>,
    auth_user: AuthUser,
    Path(id): Path<Uuid>,
) -> Result<Json<ApiResponse<LeaseWithDetails>>, AppError> {
    let lease = lease_service::get_lease(&state.db, id, auth_user.organization_id).await?;
    Ok(Json(ApiResponse::success(lease)))
}

pub async fn create(
    State(state): State<Arc<AppState>>,
    auth_user: AuthUser,
    Json(body): Json<CreateLeaseRequest>,
) -> Result<Json<ApiResponse<LeaseResponse>>, AppError> {
    let lease = lease_service::create_lease(&state.db, auth_user.organization_id, body).await?;
    Ok(Json(ApiResponse::success(lease)))
}

pub async fn update(
    State(state): State<Arc<AppState>>,
    auth_user: AuthUser,
    Path(id): Path<Uuid>,
    Json(body): Json<UpdateLeaseRequest>,
) -> Result<Json<ApiResponse<LeaseResponse>>, AppError> {
    let lease = lease_service::update_lease(&state.db, id, auth_user.organization_id, body).await?;
    Ok(Json(ApiResponse::success(lease)))
}

pub async fn terminate(
    State(state): State<Arc<AppState>>,
    auth_user: AuthUser,
    Path(id): Path<Uuid>,
) -> Result<Json<ApiResponse<LeaseResponse>>, AppError> {
    let lease = lease_service::terminate_lease(&state.db, id, auth_user.organization_id).await?;
    Ok(Json(ApiResponse::success(lease)))
}
