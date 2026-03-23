use sqlx::PgPool;
use uuid::Uuid;

use crate::errors::AppError;
use crate::models::tenant::{CreateTenantRequest, TenantResponse, UpdateTenantRequest};
use crate::repositories::tenant_repo;
use crate::utils::pagination::{PaginationMeta, PaginationParams};
use crate::utils::validator::validate_request;

pub async fn list_tenants(
    pool: &PgPool,
    org_id: Uuid,
    pagination: &PaginationParams,
) -> Result<(Vec<TenantResponse>, PaginationMeta), AppError> {
    let (tenants, total) = tenant_repo::find_all_by_org(pool, org_id, pagination).await?;
    let meta = PaginationMeta::new(pagination.page(), pagination.per_page(), total);
    let responses: Vec<TenantResponse> = tenants.into_iter().map(TenantResponse::from).collect();
    Ok((responses, meta))
}

pub async fn get_tenant(
    pool: &PgPool,
    id: Uuid,
    org_id: Uuid,
) -> Result<TenantResponse, AppError> {
    let tenant = tenant_repo::find_by_id(pool, id, org_id)
        .await?
        .ok_or_else(|| AppError::NotFound("Locataire non trouve".to_string()))?;
    Ok(TenantResponse::from(tenant))
}

pub async fn create_tenant(
    pool: &PgPool,
    org_id: Uuid,
    data: CreateTenantRequest,
) -> Result<TenantResponse, AppError> {
    validate_request(&data)?;
    let tenant = tenant_repo::create(pool, org_id, &data).await?;
    Ok(TenantResponse::from(tenant))
}

pub async fn update_tenant(
    pool: &PgPool,
    id: Uuid,
    org_id: Uuid,
    data: UpdateTenantRequest,
) -> Result<TenantResponse, AppError> {
    tenant_repo::find_by_id(pool, id, org_id)
        .await?
        .ok_or_else(|| AppError::NotFound("Locataire non trouve".to_string()))?;
    let tenant = tenant_repo::update(pool, id, org_id, &data).await?;
    Ok(TenantResponse::from(tenant))
}

pub async fn delete_tenant(
    pool: &PgPool,
    id: Uuid,
    org_id: Uuid,
) -> Result<(), AppError> {
    tenant_repo::find_by_id(pool, id, org_id)
        .await?
        .ok_or_else(|| AppError::NotFound("Locataire non trouve".to_string()))?;
    tenant_repo::delete(pool, id, org_id).await?;
    Ok(())
}
