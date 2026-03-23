use sqlx::PgPool;
use uuid::Uuid;

use crate::errors::AppError;
use crate::models::lease::{CreateLeaseRequest, LeaseResponse, LeaseWithDetails, UpdateLeaseRequest};
use crate::repositories::lease_repo;
use crate::utils::pagination::{PaginationMeta, PaginationParams};
use crate::utils::validator::validate_request;

pub async fn list_leases(
    pool: &PgPool,
    org_id: Uuid,
    pagination: &PaginationParams,
) -> Result<(Vec<LeaseWithDetails>, PaginationMeta), AppError> {
    let (leases, total) = lease_repo::find_all_by_org(pool, org_id, pagination).await?;
    let meta = PaginationMeta::new(pagination.page(), pagination.per_page(), total);
    Ok((leases, meta))
}

pub async fn get_lease(
    pool: &PgPool,
    id: Uuid,
    org_id: Uuid,
) -> Result<LeaseWithDetails, AppError> {
    let lease = lease_repo::find_with_details(pool, id, org_id)
        .await?
        .ok_or_else(|| AppError::NotFound("Bail non trouve".to_string()))?;
    Ok(lease)
}

pub async fn create_lease(
    pool: &PgPool,
    org_id: Uuid,
    data: CreateLeaseRequest,
) -> Result<LeaseResponse, AppError> {
    validate_request(&data)?;

    // Check that property belongs to org
    let _property = crate::repositories::property_repo::find_by_id(pool, data.property_id, org_id)
        .await?
        .ok_or_else(|| AppError::NotFound("Bien non trouve".to_string()))?;

    // Check no active lease on property
    if let Some(_active) = lease_repo::find_active_by_property(pool, data.property_id).await? {
        return Err(AppError::Conflict(
            "Ce bien a deja un bail actif".to_string(),
        ));
    }

    // Check tenant belongs to org
    crate::repositories::tenant_repo::find_by_id(pool, data.tenant_id, org_id)
        .await?
        .ok_or_else(|| AppError::NotFound("Locataire non trouve".to_string()))?;

    let lease = lease_repo::create(pool, &data).await?;

    // Update property status to rented
    let update_data = crate::models::property::UpdatePropertyRequest {
        reference: None,
        property_type: None,
        address: None,
        city: None,
        neighborhood: None,
        surface_m2: None,
        rooms: None,
        rent_amount: None,
        charges_amount: None,
        status: Some("rented".to_string()),
        description: None,
    };
    let _ = crate::repositories::property_repo::update(pool, data.property_id, org_id, &update_data).await;

    Ok(LeaseResponse::from(lease))
}

pub async fn update_lease(
    pool: &PgPool,
    id: Uuid,
    org_id: Uuid,
    data: UpdateLeaseRequest,
) -> Result<LeaseResponse, AppError> {
    lease_repo::find_by_id(pool, id, org_id)
        .await?
        .ok_or_else(|| AppError::NotFound("Bail non trouve".to_string()))?;

    if let Some(ref status) = data.status {
        let valid_statuses = ["active", "terminated", "pending"];
        if !valid_statuses.contains(&status.as_str()) {
            return Err(AppError::Validation(format!(
                "Statut invalide. Statuts valides: {}",
                valid_statuses.join(", ")
            )));
        }
    }

    let lease = lease_repo::update(pool, id, org_id, &data).await?;
    Ok(LeaseResponse::from(lease))
}

pub async fn terminate_lease(
    pool: &PgPool,
    id: Uuid,
    org_id: Uuid,
) -> Result<LeaseResponse, AppError> {
    let existing = lease_repo::find_by_id(pool, id, org_id)
        .await?
        .ok_or_else(|| AppError::NotFound("Bail non trouve".to_string()))?;

    if existing.status != "active" {
        return Err(AppError::BadRequest("Seul un bail actif peut etre resilie".to_string()));
    }

    let lease = lease_repo::terminate(pool, id, org_id).await?;

    // Update property status back to available
    let update_data = crate::models::property::UpdatePropertyRequest {
        reference: None,
        property_type: None,
        address: None,
        city: None,
        neighborhood: None,
        surface_m2: None,
        rooms: None,
        rent_amount: None,
        charges_amount: None,
        status: Some("available".to_string()),
        description: None,
    };
    let _ = crate::repositories::property_repo::update(pool, existing.property_id, org_id, &update_data).await;

    Ok(LeaseResponse::from(lease))
}
