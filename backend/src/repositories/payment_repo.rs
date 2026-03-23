use chrono::NaiveDate;
use sqlx::PgPool;
use uuid::Uuid;

use crate::models::payment::{CreatePaymentRequest, Payment, PaymentFilter};
use crate::utils::pagination::PaginationParams;

pub async fn find_all_by_org(
    pool: &PgPool,
    org_id: Uuid,
    pagination: &PaginationParams,
    filters: &PaymentFilter,
) -> Result<(Vec<Payment>, i64), sqlx::Error> {
    let mut conditions = vec![
        "p.organization_id = $1".to_string(),
    ];
    let mut param_idx = 2u32;

    if filters.status.is_some() {
        conditions.push(format!("pay.status = ${}", param_idx));
        param_idx += 1;
    }
    if filters.lease_id.is_some() {
        conditions.push(format!("pay.lease_id = ${}", param_idx));
        param_idx += 1;
    }
    if filters.from_date.is_some() {
        conditions.push(format!("pay.due_date >= ${}", param_idx));
        param_idx += 1;
    }
    if filters.to_date.is_some() {
        conditions.push(format!("pay.due_date <= ${}", param_idx));
        param_idx += 1;
    }

    let where_clause = conditions.join(" AND ");

    let count_sql = format!(
        "SELECT COUNT(*) FROM payments pay
         JOIN leases l ON pay.lease_id = l.id
         JOIN properties p ON l.property_id = p.id
         WHERE {}", where_clause
    );
    let mut count_query = sqlx::query_scalar::<_, i64>(&count_sql).bind(org_id);
    if let Some(ref status) = filters.status {
        count_query = count_query.bind(status);
    }
    if let Some(lease_id) = filters.lease_id {
        count_query = count_query.bind(lease_id);
    }
    if let Some(from_date) = filters.from_date {
        count_query = count_query.bind(from_date);
    }
    if let Some(to_date) = filters.to_date {
        count_query = count_query.bind(to_date);
    }
    let total = count_query.fetch_one(pool).await?;

    let data_sql = format!(
        "SELECT pay.id, pay.lease_id, pay.amount, pay.due_date, pay.paid_date, pay.payment_method, pay.reference, pay.status, pay.created_at
         FROM payments pay
         JOIN leases l ON pay.lease_id = l.id
         JOIN properties p ON l.property_id = p.id
         WHERE {}
         ORDER BY pay.due_date DESC
         LIMIT ${} OFFSET ${}",
        where_clause, param_idx, param_idx + 1
    );
    let mut data_query = sqlx::query_as::<_, Payment>(&data_sql).bind(org_id);
    if let Some(ref status) = filters.status {
        data_query = data_query.bind(status);
    }
    if let Some(lease_id) = filters.lease_id {
        data_query = data_query.bind(lease_id);
    }
    if let Some(from_date) = filters.from_date {
        data_query = data_query.bind(from_date);
    }
    if let Some(to_date) = filters.to_date {
        data_query = data_query.bind(to_date);
    }
    data_query = data_query.bind(pagination.per_page()).bind(pagination.offset());

    let payments = data_query.fetch_all(pool).await?;
    Ok((payments, total))
}

pub async fn find_by_id(pool: &PgPool, id: Uuid, org_id: Uuid) -> Result<Option<Payment>, sqlx::Error> {
    let payment = sqlx::query_as::<_, Payment>(
        "SELECT pay.id, pay.lease_id, pay.amount, pay.due_date, pay.paid_date, pay.payment_method, pay.reference, pay.status, pay.created_at
         FROM payments pay
         JOIN leases l ON pay.lease_id = l.id
         JOIN properties p ON l.property_id = p.id
         WHERE pay.id = $1 AND p.organization_id = $2"
    )
    .bind(id)
    .bind(org_id)
    .fetch_optional(pool)
    .await?;
    Ok(payment)
}

pub async fn create(pool: &PgPool, data: &CreatePaymentRequest) -> Result<Payment, sqlx::Error> {
    let status = data.status.as_deref().unwrap_or("pending");
    let payment = sqlx::query_as::<_, Payment>(
        "INSERT INTO payments (lease_id, amount, due_date, paid_date, payment_method, reference, status)
         VALUES ($1, $2, $3, $4, $5, $6, $7)
         RETURNING id, lease_id, amount, due_date, paid_date, payment_method, reference, status, created_at"
    )
    .bind(data.lease_id)
    .bind(data.amount)
    .bind(data.due_date)
    .bind(data.paid_date)
    .bind(&data.payment_method)
    .bind(&data.reference)
    .bind(status)
    .fetch_one(pool)
    .await?;
    Ok(payment)
}

pub async fn find_overdue(pool: &PgPool, org_id: Uuid) -> Result<Vec<Payment>, sqlx::Error> {
    let payments = sqlx::query_as::<_, Payment>(
        "SELECT pay.id, pay.lease_id, pay.amount, pay.due_date, pay.paid_date, pay.payment_method, pay.reference, pay.status, pay.created_at
         FROM payments pay
         JOIN leases l ON pay.lease_id = l.id
         JOIN properties p ON l.property_id = p.id
         WHERE p.organization_id = $1 AND pay.status IN ('pending', 'late') AND pay.due_date < CURRENT_DATE
         ORDER BY pay.due_date ASC"
    )
    .bind(org_id)
    .fetch_all(pool)
    .await?;
    Ok(payments)
}

#[derive(Debug, sqlx::FromRow, serde::Serialize)]
pub struct MonthlyRevenue {
    pub total: Option<i64>,
}

pub async fn monthly_revenue(pool: &PgPool, org_id: Uuid, year: i32, month: i32) -> Result<i64, sqlx::Error> {
    let start_date = NaiveDate::from_ymd_opt(year, month as u32, 1)
        .unwrap_or_else(|| NaiveDate::from_ymd_opt(year, 1, 1).unwrap());
    let end_date = if month == 12 {
        NaiveDate::from_ymd_opt(year + 1, 1, 1).unwrap()
    } else {
        NaiveDate::from_ymd_opt(year, (month + 1) as u32, 1).unwrap()
    };

    let result = sqlx::query_scalar::<_, Option<i64>>(
        "SELECT SUM(pay.amount) FROM payments pay
         JOIN leases l ON pay.lease_id = l.id
         JOIN properties p ON l.property_id = p.id
         WHERE p.organization_id = $1 AND pay.status = 'paid' AND pay.paid_date >= $2 AND pay.paid_date < $3"
    )
    .bind(org_id)
    .bind(start_date)
    .bind(end_date)
    .fetch_one(pool)
    .await?;

    Ok(result.unwrap_or(0))
}

pub async fn overdue_count(pool: &PgPool, org_id: Uuid) -> Result<i64, sqlx::Error> {
    let count = sqlx::query_scalar::<_, i64>(
        "SELECT COUNT(*) FROM payments pay
         JOIN leases l ON pay.lease_id = l.id
         JOIN properties p ON l.property_id = p.id
         WHERE p.organization_id = $1 AND pay.status IN ('pending', 'late') AND pay.due_date < CURRENT_DATE"
    )
    .bind(org_id)
    .fetch_one(pool)
    .await?;
    Ok(count)
}
