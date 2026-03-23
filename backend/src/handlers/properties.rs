use std::sync::Arc;

use axum::{
    extract::{Path, Query, State},
    Json,
};
use uuid::Uuid;

use crate::errors::AppError;
use crate::middleware::auth::AuthUser;
use crate::models::property::{CreatePropertyRequest, PropertyFilter, PropertyResponse, UpdatePropertyRequest};
use crate::services::properties as property_service;
use crate::utils::pagination::PaginationParams;
use crate::utils::response::ApiResponse;
use crate::AppState;

pub async fn list(
    State(state): State<Arc<AppState>>,
    auth_user: AuthUser,
    Query(pagination): Query<PaginationParams>,
    Query(filters): Query<PropertyFilter>,
) -> Result<Json<ApiResponse<Vec<PropertyResponse>>>, AppError> {
    let (properties, meta) = property_service::list_properties(
        &state.db,
        auth_user.organization_id,
        &pagination,
        &filters,
    )
    .await?;

    Ok(Json(ApiResponse::with_pagination(properties, meta)))
}

pub async fn get_by_id(
    State(state): State<Arc<AppState>>,
    auth_user: AuthUser,
    Path(id): Path<Uuid>,
) -> Result<Json<ApiResponse<PropertyResponse>>, AppError> {
    let property = property_service::get_property(&state.db, id, auth_user.organization_id).await?;
    Ok(Json(ApiResponse::success(property)))
}

pub async fn create(
    State(state): State<Arc<AppState>>,
    auth_user: AuthUser,
    Json(body): Json<CreatePropertyRequest>,
) -> Result<Json<ApiResponse<PropertyResponse>>, AppError> {
    let property = property_service::create_property(&state.db, auth_user.organization_id, body).await?;
    Ok(Json(ApiResponse::success(property)))
}

pub async fn update(
    State(state): State<Arc<AppState>>,
    auth_user: AuthUser,
    Path(id): Path<Uuid>,
    Json(body): Json<UpdatePropertyRequest>,
) -> Result<Json<ApiResponse<PropertyResponse>>, AppError> {
    let property = property_service::update_property(&state.db, id, auth_user.organization_id, body).await?;
    Ok(Json(ApiResponse::success(property)))
}

pub async fn delete(
    State(state): State<Arc<AppState>>,
    auth_user: AuthUser,
    Path(id): Path<Uuid>,
) -> Result<Json<ApiResponse<()>>, AppError> {
    property_service::delete_property(&state.db, id, auth_user.organization_id).await?;
    Ok(Json(ApiResponse::success(())))
}
