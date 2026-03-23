use sqlx::PgPool;
use uuid::Uuid;

use crate::errors::AppError;
use crate::models::payment::{CreatePaymentRequest, PaymentFilter, PaymentResponse};
use crate::repositories::payment_repo;
use crate::utils::pagination::{PaginationMeta, PaginationParams};
use crate::utils::validator::validate_request;

pub async fn list_payments(
    pool: &PgPool,
    org_id: Uuid,
    pagination: &PaginationParams,
    filters: &PaymentFilter,
) -> Result<(Vec<PaymentResponse>, PaginationMeta), AppError> {
    let (payments, total) = payment_repo::find_all_by_org(pool, org_id, pagination, filters).await?;
    let meta = PaginationMeta::new(pagination.page(), pagination.per_page(), total);
    let responses: Vec<PaymentResponse> = payments.into_iter().map(PaymentResponse::from).collect();
    Ok((responses, meta))
}

pub async fn get_payment(
    pool: &PgPool,
    id: Uuid,
    org_id: Uuid,
) -> Result<PaymentResponse, AppError> {
    let payment = payment_repo::find_by_id(pool, id, org_id)
        .await?
        .ok_or_else(|| AppError::NotFound("Paiement non trouve".to_string()))?;
    Ok(PaymentResponse::from(payment))
}

pub async fn create_payment(
    pool: &PgPool,
    org_id: Uuid,
    data: CreatePaymentRequest,
) -> Result<PaymentResponse, AppError> {
    validate_request(&data)?;

    // Verify the lease belongs to this org
    crate::repositories::lease_repo::find_by_id(pool, data.lease_id, org_id)
        .await?
        .ok_or_else(|| AppError::NotFound("Bail non trouve".to_string()))?;

    if let Some(ref method) = data.payment_method {
        let valid_methods = ["cash", "orange_money", "wave", "bank_transfer", "check"];
        if !valid_methods.contains(&method.as_str()) {
            return Err(AppError::Validation(format!(
                "Methode de paiement invalide. Methodes valides: {}",
                valid_methods.join(", ")
            )));
        }
    }

    let payment = payment_repo::create(pool, &data).await?;
    Ok(PaymentResponse::from(payment))
}

pub async fn get_overdue_payments(
    pool: &PgPool,
    org_id: Uuid,
) -> Result<Vec<PaymentResponse>, AppError> {
    let payments = payment_repo::find_overdue(pool, org_id).await?;
    let responses: Vec<PaymentResponse> = payments.into_iter().map(PaymentResponse::from).collect();
    Ok(responses)
}
