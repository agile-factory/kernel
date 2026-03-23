use sqlx::PgPool;
use uuid::Uuid;

use crate::models::property::{CreatePropertyRequest, Property, PropertyFilter, UpdatePropertyRequest};
use crate::utils::pagination::PaginationParams;

pub async fn find_all_by_org(
    pool: &PgPool,
    org_id: Uuid,
    pagination: &PaginationParams,
    filters: &PropertyFilter,
) -> Result<(Vec<Property>, i64), sqlx::Error> {
    let mut conditions = vec!["organization_id = $1".to_string()];
    let mut param_idx = 2u32;

    // Build dynamic filter conditions (we still need to bind them in order)
    if filters.status.is_some() {
        conditions.push(format!("status = ${}", param_idx));
        param_idx += 1;
    }
    if filters.city.is_some() {
        conditions.push(format!("city ILIKE ${}", param_idx));
        param_idx += 1;
    }
    if filters.property_type.is_some() {
        conditions.push(format!("property_type = ${}", param_idx));
        param_idx += 1;
    }
    if filters.min_rent.is_some() {
        conditions.push(format!("rent_amount >= ${}", param_idx));
        param_idx += 1;
    }
    if filters.max_rent.is_some() {
        conditions.push(format!("rent_amount <= ${}", param_idx));
        param_idx += 1;
    }

    let where_clause = conditions.join(" AND ");

    // Count query
    let count_sql = format!("SELECT COUNT(*) as count FROM properties WHERE {}", where_clause);
    let mut count_query = sqlx::query_scalar::<_, i64>(&count_sql).bind(org_id);
    if let Some(ref status) = filters.status {
        count_query = count_query.bind(status);
    }
    if let Some(ref city) = filters.city {
        count_query = count_query.bind(format!("%{}%", city));
    }
    if let Some(ref pt) = filters.property_type {
        count_query = count_query.bind(pt);
    }
    if let Some(min_rent) = filters.min_rent {
        count_query = count_query.bind(min_rent);
    }
    if let Some(max_rent) = filters.max_rent {
        count_query = count_query.bind(max_rent);
    }
    let total = count_query.fetch_one(pool).await?;

    // Data query
    let data_sql = format!(
        "SELECT id, organization_id, reference, property_type, address, city, neighborhood, surface_m2, rooms, rent_amount, charges_amount, status, description, created_at, updated_at
         FROM properties WHERE {} ORDER BY created_at DESC LIMIT ${} OFFSET ${}",
        where_clause, param_idx, param_idx + 1
    );
    let mut data_query = sqlx::query_as::<_, Property>(&data_sql).bind(org_id);
    if let Some(ref status) = filters.status {
        data_query = data_query.bind(status);
    }
    if let Some(ref city) = filters.city {
        data_query = data_query.bind(format!("%{}%", city));
    }
    if let Some(ref pt) = filters.property_type {
        data_query = data_query.bind(pt);
    }
    if let Some(min_rent) = filters.min_rent {
        data_query = data_query.bind(min_rent);
    }
    if let Some(max_rent) = filters.max_rent {
        data_query = data_query.bind(max_rent);
    }
    data_query = data_query.bind(pagination.per_page()).bind(pagination.offset());

    let properties = data_query.fetch_all(pool).await?;
    Ok((properties, total))
}

pub async fn find_by_id(pool: &PgPool, id: Uuid, org_id: Uuid) -> Result<Option<Property>, sqlx::Error> {
    let property = sqlx::query_as::<_, Property>(
        "SELECT id, organization_id, reference, property_type, address, city, neighborhood, surface_m2, rooms, rent_amount, charges_amount, status, description, created_at, updated_at
         FROM properties WHERE id = $1 AND organization_id = $2"
    )
    .bind(id)
    .bind(org_id)
    .fetch_optional(pool)
    .await?;
    Ok(property)
}

pub async fn create(
    pool: &PgPool,
    org_id: Uuid,
    data: &CreatePropertyRequest,
) -> Result<Property, sqlx::Error> {
    let property = sqlx::query_as::<_, Property>(
        "INSERT INTO properties (organization_id, reference, property_type, address, city, neighborhood, surface_m2, rooms, rent_amount, charges_amount, description)
         VALUES ($1, $2, $3, $4, COALESCE($5, 'Dakar'), $6, $7, $8, $9, COALESCE($10, 0), $11)
         RETURNING id, organization_id, reference, property_type, address, city, neighborhood, surface_m2, rooms, rent_amount, charges_amount, status, description, created_at, updated_at"
    )
    .bind(org_id)
    .bind(&data.reference)
    .bind(&data.property_type)
    .bind(&data.address)
    .bind(&data.city)
    .bind(&data.neighborhood)
    .bind(data.surface_m2.map(|v| v))
    .bind(data.rooms)
    .bind(data.rent_amount)
    .bind(data.charges_amount)
    .bind(&data.description)
    .fetch_one(pool)
    .await?;
    Ok(property)
}

pub async fn update(
    pool: &PgPool,
    id: Uuid,
    org_id: Uuid,
    data: &UpdatePropertyRequest,
) -> Result<Property, sqlx::Error> {
    let property = sqlx::query_as::<_, Property>(
        "UPDATE properties SET
            reference = COALESCE($3, reference),
            property_type = COALESCE($4, property_type),
            address = COALESCE($5, address),
            city = COALESCE($6, city),
            neighborhood = COALESCE($7, neighborhood),
            surface_m2 = COALESCE($8, surface_m2),
            rooms = COALESCE($9, rooms),
            rent_amount = COALESCE($10, rent_amount),
            charges_amount = COALESCE($11, charges_amount),
            status = COALESCE($12, status),
            description = COALESCE($13, description)
         WHERE id = $1 AND organization_id = $2
         RETURNING id, organization_id, reference, property_type, address, city, neighborhood, surface_m2, rooms, rent_amount, charges_amount, status, description, created_at, updated_at"
    )
    .bind(id)
    .bind(org_id)
    .bind(&data.reference)
    .bind(&data.property_type)
    .bind(&data.address)
    .bind(&data.city)
    .bind(&data.neighborhood)
    .bind(data.surface_m2.map(|v| v))
    .bind(data.rooms)
    .bind(data.rent_amount)
    .bind(data.charges_amount)
    .bind(&data.status)
    .bind(&data.description)
    .fetch_one(pool)
    .await?;
    Ok(property)
}

pub async fn delete(pool: &PgPool, id: Uuid, org_id: Uuid) -> Result<(), sqlx::Error> {
    sqlx::query("DELETE FROM properties WHERE id = $1 AND organization_id = $2")
        .bind(id)
        .bind(org_id)
        .execute(pool)
        .await?;
    Ok(())
}

#[derive(Debug, sqlx::FromRow, serde::Serialize)]
pub struct StatusCount {
    pub status: String,
    pub count: i64,
}

pub async fn count_by_status(pool: &PgPool, org_id: Uuid) -> Result<Vec<StatusCount>, sqlx::Error> {
    let counts = sqlx::query_as::<_, StatusCount>(
        "SELECT status, COUNT(*) as count FROM properties WHERE organization_id = $1 GROUP BY status"
    )
    .bind(org_id)
    .fetch_all(pool)
    .await?;
    Ok(counts)
}
