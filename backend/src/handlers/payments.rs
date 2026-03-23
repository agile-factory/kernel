use std::sync::Arc;

use axum::{
    extract::{Path, Query, State},
    Json,
};
use uuid::Uuid;

use crate::errors::AppError;
use crate::middleware::auth::AuthUser;
use crate::models::payment::{CreatePaymentRequest, PaymentFilter, PaymentResponse};
use crate::services::payments as payment_service;
use crate::utils::pagination::PaginationParams;
use crate::utils::response::ApiResponse;
use crate::AppState;

pub async fn list(
    State(state): State<Arc<AppState>>,
    auth_user: AuthUser,
    Query(pagination): Query<PaginationParams>,
    Query(filters): Query<PaymentFilter>,
) -> Result<Json<ApiResponse<Vec<PaymentResponse>>>, AppError> {
    let (payments, meta) = payment_service::list_payments(
        &state.db,
        auth_user.organization_id,
        &pagination,
        &filters,
    )
    .await?;

    Ok(Json(ApiResponse::with_pagination(payments, meta)))
}

pub async fn get_by_id(
    State(state): State<Arc<AppState>>,
    auth_user: AuthUser,
    Path(id): Path<Uuid>,
) -> Result<Json<ApiResponse<PaymentResponse>>, AppError> {
    let payment = payment_service::get_payment(&state.db, id, auth_user.organization_id).await?;
    Ok(Json(ApiResponse::success(payment)))
}

pub async fn create(
    State(state): State<Arc<AppState>>,
    auth_user: AuthUser,
    Json(body): Json<CreatePaymentRequest>,
) -> Result<Json<ApiResponse<PaymentResponse>>, AppError> {
    let payment = payment_service::create_payment(&state.db, auth_user.organization_id, body).await?;
    Ok(Json(ApiResponse::success(payment)))
}

pub async fn overdue(
    State(state): State<Arc<AppState>>,
    auth_user: AuthUser,
) -> Result<Json<ApiResponse<Vec<PaymentResponse>>>, AppError> {
    let payments = payment_service::get_overdue_payments(&state.db, auth_user.organization_id).await?;
    Ok(Json(ApiResponse::success(payments)))
}

pub async fn remind(
    State(state): State<Arc<AppState>>,
    auth_user: AuthUser,
) -> Result<Json<ApiResponse<serde_json::Value>>, AppError> {
    let overdue_payments = crate::repositories::payment_repo::find_overdue(&state.db, auth_user.organization_id).await?;

    let mut sent_count = 0;
    for payment in &overdue_payments {
        // Look up lease -> tenant for phone number
        if let Some(lease) = crate::repositories::lease_repo::find_by_id(&state.db, payment.lease_id, auth_user.organization_id).await? {
            if let Some(tenant) = crate::repositories::tenant_repo::find_by_id(&state.db, lease.tenant_id, auth_user.organization_id).await? {
                let _ = crate::services::notifications::send_payment_reminder(
                    &tenant.phone,
                    payment.amount,
                    payment.due_date,
                )
                .await;
                sent_count += 1;
            }
        }
    }

    Ok(Json(ApiResponse::success(serde_json::json!({
        "reminders_sent": sent_count,
        "total_overdue": overdue_payments.len(),
    }))))
}
