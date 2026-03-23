use sqlx::PgPool;
use uuid::Uuid;

use crate::models::invoice::{CreateInvoiceRequest, Invoice};

pub async fn find_by_id(pool: &PgPool, id: Uuid) -> Result<Option<Invoice>, sqlx::Error> {
    let invoice = sqlx::query_as::<_, Invoice>(
        "SELECT id, payment_id, invoice_number, pdf_url, sent_at, created_at
         FROM invoices WHERE id = $1"
    )
    .bind(id)
    .fetch_optional(pool)
    .await?;
    Ok(invoice)
}

pub async fn find_by_payment(pool: &PgPool, payment_id: Uuid) -> Result<Option<Invoice>, sqlx::Error> {
    let invoice = sqlx::query_as::<_, Invoice>(
        "SELECT id, payment_id, invoice_number, pdf_url, sent_at, created_at
         FROM invoices WHERE payment_id = $1"
    )
    .bind(payment_id)
    .fetch_optional(pool)
    .await?;
    Ok(invoice)
}

pub async fn create(pool: &PgPool, data: &CreateInvoiceRequest) -> Result<Invoice, sqlx::Error> {
    let invoice = sqlx::query_as::<_, Invoice>(
        "INSERT INTO invoices (payment_id, invoice_number, pdf_url)
         VALUES ($1, $2, $3)
         RETURNING id, payment_id, invoice_number, pdf_url, sent_at, created_at"
    )
    .bind(data.payment_id)
    .bind(&data.invoice_number)
    .bind(&data.pdf_url)
    .fetch_one(pool)
    .await?;
    Ok(invoice)
}

pub async fn find_all_by_org(
    pool: &PgPool,
    org_id: Uuid,
) -> Result<Vec<Invoice>, sqlx::Error> {
    let invoices = sqlx::query_as::<_, Invoice>(
        "SELECT inv.id, inv.payment_id, inv.invoice_number, inv.pdf_url, inv.sent_at, inv.created_at
         FROM invoices inv
         JOIN payments pay ON inv.payment_id = pay.id
         JOIN leases l ON pay.lease_id = l.id
         JOIN properties p ON l.property_id = p.id
         WHERE p.organization_id = $1
         ORDER BY inv.created_at DESC"
    )
    .bind(org_id)
    .fetch_all(pool)
    .await?;
    Ok(invoices)
}
