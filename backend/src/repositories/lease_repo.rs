use sqlx::PgPool;
use uuid::Uuid;

use crate::models::lease::{CreateLeaseRequest, Lease, LeaseWithDetails, UpdateLeaseRequest};
use crate::utils::pagination::PaginationParams;

pub async fn find_all_by_org(
    pool: &PgPool,
    org_id: Uuid,
    pagination: &PaginationParams,
) -> Result<(Vec<LeaseWithDetails>, i64), sqlx::Error> {
    let total = sqlx::query_scalar::<_, i64>(
        "SELECT COUNT(*) FROM leases l
         JOIN properties p ON l.property_id = p.id
         WHERE p.organization_id = $1"
    )
    .bind(org_id)
    .fetch_one(pool)
    .await?;

    let leases = sqlx::query_as::<_, LeaseWithDetails>(
        "SELECT l.id, l.property_id, l.tenant_id, l.start_date, l.end_date, l.rent_amount, l.deposit_amount, l.payment_day, l.status, l.contract_pdf_url, l.signed_at, l.created_at, l.updated_at,
                p.reference AS property_reference, p.address AS property_address,
                t.first_name AS tenant_first_name, t.last_name AS tenant_last_name
         FROM leases l
         JOIN properties p ON l.property_id = p.id
         JOIN tenants t ON l.tenant_id = t.id
         WHERE p.organization_id = $1
         ORDER BY l.created_at DESC
         LIMIT $2 OFFSET $3"
    )
    .bind(org_id)
    .bind(pagination.per_page())
    .bind(pagination.offset())
    .fetch_all(pool)
    .await?;

    Ok((leases, total))
}

pub async fn find_by_id(pool: &PgPool, id: Uuid, org_id: Uuid) -> Result<Option<Lease>, sqlx::Error> {
    let lease = sqlx::query_as::<_, Lease>(
        "SELECT l.id, l.property_id, l.tenant_id, l.start_date, l.end_date, l.rent_amount, l.deposit_amount, l.payment_day, l.status, l.contract_pdf_url, l.signed_at, l.created_at, l.updated_at
         FROM leases l
         JOIN properties p ON l.property_id = p.id
         WHERE l.id = $1 AND p.organization_id = $2"
    )
    .bind(id)
    .bind(org_id)
    .fetch_optional(pool)
    .await?;
    Ok(lease)
}

pub async fn find_with_details(pool: &PgPool, id: Uuid, org_id: Uuid) -> Result<Option<LeaseWithDetails>, sqlx::Error> {
    let lease = sqlx::query_as::<_, LeaseWithDetails>(
        "SELECT l.id, l.property_id, l.tenant_id, l.start_date, l.end_date, l.rent_amount, l.deposit_amount, l.payment_day, l.status, l.contract_pdf_url, l.signed_at, l.created_at, l.updated_at,
                p.reference AS property_reference, p.address AS property_address,
                t.first_name AS tenant_first_name, t.last_name AS tenant_last_name
         FROM leases l
         JOIN properties p ON l.property_id = p.id
         JOIN tenants t ON l.tenant_id = t.id
         WHERE l.id = $1 AND p.organization_id = $2"
    )
    .bind(id)
    .bind(org_id)
    .fetch_optional(pool)
    .await?;
    Ok(lease)
}

pub async fn find_active_by_property(pool: &PgPool, property_id: Uuid) -> Result<Option<Lease>, sqlx::Error> {
    let lease = sqlx::query_as::<_, Lease>(
        "SELECT id, property_id, tenant_id, start_date, end_date, rent_amount, deposit_amount, payment_day, status, contract_pdf_url, signed_at, created_at, updated_at
         FROM leases WHERE property_id = $1 AND status = 'active'"
    )
    .bind(property_id)
    .fetch_optional(pool)
    .await?;
    Ok(lease)
}

pub async fn create(pool: &PgPool, data: &CreateLeaseRequest) -> Result<Lease, sqlx::Error> {
    let lease = sqlx::query_as::<_, Lease>(
        "INSERT INTO leases (property_id, tenant_id, start_date, end_date, rent_amount, deposit_amount, payment_day)
         VALUES ($1, $2, $3, $4, $5, COALESCE($6, 0), COALESCE($7, 1))
         RETURNING id, property_id, tenant_id, start_date, end_date, rent_amount, deposit_amount, payment_day, status, contract_pdf_url, signed_at, created_at, updated_at"
    )
    .bind(data.property_id)
    .bind(data.tenant_id)
    .bind(data.start_date)
    .bind(data.end_date)
    .bind(data.rent_amount)
    .bind(data.deposit_amount)
    .bind(data.payment_day)
    .fetch_one(pool)
    .await?;
    Ok(lease)
}

pub async fn update(
    pool: &PgPool,
    id: Uuid,
    org_id: Uuid,
    data: &UpdateLeaseRequest,
) -> Result<Lease, sqlx::Error> {
    let lease = sqlx::query_as::<_, Lease>(
        "UPDATE leases SET
            end_date = COALESCE($2, end_date),
            rent_amount = COALESCE($3, rent_amount),
            deposit_amount = COALESCE($4, deposit_amount),
            payment_day = COALESCE($5, payment_day),
            status = COALESCE($6, status),
            contract_pdf_url = COALESCE($7, contract_pdf_url)
         WHERE id = $1 AND property_id IN (SELECT id FROM properties WHERE organization_id = $8)
         RETURNING id, property_id, tenant_id, start_date, end_date, rent_amount, deposit_amount, payment_day, status, contract_pdf_url, signed_at, created_at, updated_at"
    )
    .bind(id)
    .bind(data.end_date)
    .bind(data.rent_amount)
    .bind(data.deposit_amount)
    .bind(data.payment_day)
    .bind(&data.status)
    .bind(&data.contract_pdf_url)
    .bind(org_id)
    .fetch_one(pool)
    .await?;
    Ok(lease)
}

pub async fn terminate(pool: &PgPool, id: Uuid, org_id: Uuid) -> Result<Lease, sqlx::Error> {
    let lease = sqlx::query_as::<_, Lease>(
        "UPDATE leases SET status = 'terminated', end_date = CURRENT_DATE
         WHERE id = $1 AND property_id IN (SELECT id FROM properties WHERE organization_id = $2) AND status = 'active'
         RETURNING id, property_id, tenant_id, start_date, end_date, rent_amount, deposit_amount, payment_day, status, contract_pdf_url, signed_at, created_at, updated_at"
    )
    .bind(id)
    .bind(org_id)
    .fetch_one(pool)
    .await?;
    Ok(lease)
}

pub async fn count_active_by_org(pool: &PgPool, org_id: Uuid) -> Result<i64, sqlx::Error> {
    let count = sqlx::query_scalar::<_, i64>(
        "SELECT COUNT(*) FROM leases l
         JOIN properties p ON l.property_id = p.id
         WHERE p.organization_id = $1 AND l.status = 'active'"
    )
    .bind(org_id)
    .fetch_one(pool)
    .await?;
    Ok(count)
}
