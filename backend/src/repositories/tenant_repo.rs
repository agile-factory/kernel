use sqlx::PgPool;
use uuid::Uuid;

use crate::models::tenant::{CreateTenantRequest, Tenant, UpdateTenantRequest};
use crate::utils::pagination::PaginationParams;

pub async fn find_all_by_org(
    pool: &PgPool,
    org_id: Uuid,
    pagination: &PaginationParams,
) -> Result<(Vec<Tenant>, i64), sqlx::Error> {
    let total = sqlx::query_scalar::<_, i64>(
        "SELECT COUNT(*) FROM tenants WHERE organization_id = $1"
    )
    .bind(org_id)
    .fetch_one(pool)
    .await?;

    let tenants = sqlx::query_as::<_, Tenant>(
        "SELECT id, organization_id, first_name, last_name, phone, email, cni_number, profession, employer, emergency_contact, emergency_phone, created_at, updated_at
         FROM tenants WHERE organization_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3"
    )
    .bind(org_id)
    .bind(pagination.per_page())
    .bind(pagination.offset())
    .fetch_all(pool)
    .await?;

    Ok((tenants, total))
}

pub async fn find_by_id(pool: &PgPool, id: Uuid, org_id: Uuid) -> Result<Option<Tenant>, sqlx::Error> {
    let tenant = sqlx::query_as::<_, Tenant>(
        "SELECT id, organization_id, first_name, last_name, phone, email, cni_number, profession, employer, emergency_contact, emergency_phone, created_at, updated_at
         FROM tenants WHERE id = $1 AND organization_id = $2"
    )
    .bind(id)
    .bind(org_id)
    .fetch_optional(pool)
    .await?;
    Ok(tenant)
}

pub async fn create(
    pool: &PgPool,
    org_id: Uuid,
    data: &CreateTenantRequest,
) -> Result<Tenant, sqlx::Error> {
    let tenant = sqlx::query_as::<_, Tenant>(
        "INSERT INTO tenants (organization_id, first_name, last_name, phone, email, cni_number, profession, employer, emergency_contact, emergency_phone)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
         RETURNING id, organization_id, first_name, last_name, phone, email, cni_number, profession, employer, emergency_contact, emergency_phone, created_at, updated_at"
    )
    .bind(org_id)
    .bind(&data.first_name)
    .bind(&data.last_name)
    .bind(&data.phone)
    .bind(&data.email)
    .bind(&data.cni_number)
    .bind(&data.profession)
    .bind(&data.employer)
    .bind(&data.emergency_contact)
    .bind(&data.emergency_phone)
    .fetch_one(pool)
    .await?;
    Ok(tenant)
}

pub async fn update(
    pool: &PgPool,
    id: Uuid,
    org_id: Uuid,
    data: &UpdateTenantRequest,
) -> Result<Tenant, sqlx::Error> {
    let tenant = sqlx::query_as::<_, Tenant>(
        "UPDATE tenants SET
            first_name = COALESCE($3, first_name),
            last_name = COALESCE($4, last_name),
            phone = COALESCE($5, phone),
            email = COALESCE($6, email),
            cni_number = COALESCE($7, cni_number),
            profession = COALESCE($8, profession),
            employer = COALESCE($9, employer),
            emergency_contact = COALESCE($10, emergency_contact),
            emergency_phone = COALESCE($11, emergency_phone)
         WHERE id = $1 AND organization_id = $2
         RETURNING id, organization_id, first_name, last_name, phone, email, cni_number, profession, employer, emergency_contact, emergency_phone, created_at, updated_at"
    )
    .bind(id)
    .bind(org_id)
    .bind(&data.first_name)
    .bind(&data.last_name)
    .bind(&data.phone)
    .bind(&data.email)
    .bind(&data.cni_number)
    .bind(&data.profession)
    .bind(&data.employer)
    .bind(&data.emergency_contact)
    .bind(&data.emergency_phone)
    .fetch_one(pool)
    .await?;
    Ok(tenant)
}

pub async fn delete(pool: &PgPool, id: Uuid, org_id: Uuid) -> Result<(), sqlx::Error> {
    sqlx::query("DELETE FROM tenants WHERE id = $1 AND organization_id = $2")
        .bind(id)
        .bind(org_id)
        .execute(pool)
        .await?;
    Ok(())
}

pub async fn count_by_org(pool: &PgPool, org_id: Uuid) -> Result<i64, sqlx::Error> {
    let count = sqlx::query_scalar::<_, i64>(
        "SELECT COUNT(*) FROM tenants WHERE organization_id = $1"
    )
    .bind(org_id)
    .fetch_one(pool)
    .await?;
    Ok(count)
}
