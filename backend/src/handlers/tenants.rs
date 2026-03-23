use std::sync::Arc;

use axum::{
    extract::{Path, Query, State},
    Json,
};
use uuid::Uuid;

use crate::errors::AppError;
use crate::middleware::auth::AuthUser;
use crate::models::tenant::{CreateTenantRequest, TenantResponse, UpdateTenantRequest};
use crate::services::tenants as tenant_service;
use crate::utils::pagination::PaginationParams;
use crate::utils::response::ApiResponse;
use crate::AppState;

pub async fn list(
    State(state): State<Arc<AppState>>,
    auth_user: AuthUser,
    Query(pagination): Query<PaginationParams>,
) -> Result<Json<ApiResponse<Vec<TenantResponse>>>, AppError> {
    let (tenants, meta) = tenant_service::list_tenants(
        &state.db,
        auth_user.organization_id,
        &pagination,
    )
    .await?;

    Ok(Json(ApiResponse::with_pagination(tenants, meta)))
}

pub async fn get_by_id(
    State(state): State<Arc<AppState>>,
    auth_user: AuthUser,
    Path(id): Path<Uuid>,
) -> Result<Json<ApiResponse<TenantResponse>>, AppError> {
    let tenant = tenant_service::get_tenant(&state.db, id, auth_user.organization_id).await?;
    Ok(Json(ApiResponse::success(tenant)))
}

pub async fn create(
    State(state): State<Arc<AppState>>,
    auth_user: AuthUser,
    Json(body): Json<CreateTenantRequest>,
) -> Result<Json<ApiResponse<TenantResponse>>, AppError> {
    let tenant = tenant_service::create_tenant(&state.db, auth_user.organization_id, body).await?;
    Ok(Json(ApiResponse::success(tenant)))
}

pub async fn update(
    State(state): State<Arc<AppState>>,
    auth_user: AuthUser,
    Path(id): Path<Uuid>,
    Json(body): Json<UpdateTenantRequest>,
) -> Result<Json<ApiResponse<TenantResponse>>, AppError> {
    let tenant = tenant_service::update_tenant(&state.db, id, auth_user.organization_id, body).await?;
    Ok(Json(ApiResponse::success(tenant)))
}

pub async fn delete(
    State(state): State<Arc<AppState>>,
    auth_user: AuthUser,
    Path(id): Path<Uuid>,
) -> Result<Json<ApiResponse<()>>, AppError> {
    tenant_service::delete_tenant(&state.db, id, auth_user.organization_id).await?;
    Ok(Json(ApiResponse::success(())))
}
