use sqlx::PgPool;
use uuid::Uuid;

use crate::errors::AppError;
use crate::models::property::{
    CreatePropertyRequest, PropertyFilter, PropertyResponse, UpdatePropertyRequest,
};
use crate::repositories::property_repo;
use crate::utils::pagination::{PaginationMeta, PaginationParams};
use crate::utils::validator::validate_request;

pub async fn list_properties(
    pool: &PgPool,
    org_id: Uuid,
    pagination: &PaginationParams,
    filters: &PropertyFilter,
) -> Result<(Vec<PropertyResponse>, PaginationMeta), AppError> {
    let (properties, total) = property_repo::find_all_by_org(pool, org_id, pagination, filters).await?;
    let meta = PaginationMeta::new(pagination.page(), pagination.per_page(), total);
    let responses: Vec<PropertyResponse> = properties.into_iter().map(PropertyResponse::from).collect();
    Ok((responses, meta))
}

pub async fn get_property(
    pool: &PgPool,
    id: Uuid,
    org_id: Uuid,
) -> Result<PropertyResponse, AppError> {
    let property = property_repo::find_by_id(pool, id, org_id)
        .await?
        .ok_or_else(|| AppError::NotFound("Bien non trouve".to_string()))?;
    Ok(PropertyResponse::from(property))
}

pub async fn create_property(
    pool: &PgPool,
    org_id: Uuid,
    data: CreatePropertyRequest,
) -> Result<PropertyResponse, AppError> {
    validate_request(&data)?;

    let valid_types = ["apartment", "house", "land", "commercial", "studio", "villa"];
    if !valid_types.contains(&data.property_type.as_str()) {
        return Err(AppError::Validation(format!(
            "Type de bien invalide. Types valides: {}",
            valid_types.join(", ")
        )));
    }

    let property = property_repo::create(pool, org_id, &data).await?;
    Ok(PropertyResponse::from(property))
}

pub async fn update_property(
    pool: &PgPool,
    id: Uuid,
    org_id: Uuid,
    data: UpdatePropertyRequest,
) -> Result<PropertyResponse, AppError> {
    // Verify property exists
    property_repo::find_by_id(pool, id, org_id)
        .await?
        .ok_or_else(|| AppError::NotFound("Bien non trouve".to_string()))?;

    if let Some(ref status) = data.status {
        let valid_statuses = ["available", "rented", "maintenance"];
        if !valid_statuses.contains(&status.as_str()) {
            return Err(AppError::Validation(format!(
                "Statut invalide. Statuts valides: {}",
                valid_statuses.join(", ")
            )));
        }
    }

    let property = property_repo::update(pool, id, org_id, &data).await?;
    Ok(PropertyResponse::from(property))
}

pub async fn delete_property(
    pool: &PgPool,
    id: Uuid,
    org_id: Uuid,
) -> Result<(), AppError> {
    property_repo::find_by_id(pool, id, org_id)
        .await?
        .ok_or_else(|| AppError::NotFound("Bien non trouve".to_string()))?;
    property_repo::delete(pool, id, org_id).await?;
    Ok(())
}
