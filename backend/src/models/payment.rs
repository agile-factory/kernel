use chrono::{DateTime, NaiveDate, Utc};
use serde::{Deserialize, Serialize};
use sqlx::FromRow;
use uuid::Uuid;
use validator::Validate;

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct Payment {
    pub id: Uuid,
    pub lease_id: Uuid,
    pub amount: i64,
    pub due_date: NaiveDate,
    pub paid_date: Option<NaiveDate>,
    pub payment_method: Option<String>,
    pub reference: Option<String>,
    pub status: String,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Deserialize, Validate)]
pub struct CreatePaymentRequest {
    pub lease_id: Uuid,

    #[validate(range(min = 0, message = "Le montant doit être positif"))]
    pub amount: i64,

    pub due_date: NaiveDate,
    pub paid_date: Option<NaiveDate>,
    pub payment_method: Option<String>,
    pub reference: Option<String>,
    pub status: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct PaymentResponse {
    pub id: Uuid,
    pub lease_id: Uuid,
    pub amount: i64,
    pub due_date: NaiveDate,
    pub paid_date: Option<NaiveDate>,
    pub payment_method: Option<String>,
    pub reference: Option<String>,
    pub status: String,
    pub created_at: DateTime<Utc>,
}

impl From<Payment> for PaymentResponse {
    fn from(p: Payment) -> Self {
        Self {
            id: p.id,
            lease_id: p.lease_id,
            amount: p.amount,
            due_date: p.due_date,
            paid_date: p.paid_date,
            payment_method: p.payment_method,
            reference: p.reference,
            status: p.status,
            created_at: p.created_at,
        }
    }
}

#[derive(Debug, Deserialize, Default)]
pub struct PaymentFilter {
    pub status: Option<String>,
    pub lease_id: Option<Uuid>,
    pub from_date: Option<NaiveDate>,
    pub to_date: Option<NaiveDate>,
}
