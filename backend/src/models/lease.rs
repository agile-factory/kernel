use chrono::{DateTime, NaiveDate, Utc};
use serde::{Deserialize, Serialize};
use sqlx::FromRow;
use uuid::Uuid;
use validator::Validate;

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct Lease {
    pub id: Uuid,
    pub property_id: Uuid,
    pub tenant_id: Uuid,
    pub start_date: NaiveDate,
    pub end_date: Option<NaiveDate>,
    pub rent_amount: i64,
    pub deposit_amount: i64,
    pub payment_day: i16,
    pub status: String,
    pub contract_pdf_url: Option<String>,
    pub signed_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Deserialize, Validate)]
pub struct CreateLeaseRequest {
    pub property_id: Uuid,
    pub tenant_id: Uuid,
    pub start_date: NaiveDate,
    pub end_date: Option<NaiveDate>,

    #[validate(range(min = 0, message = "Le loyer doit être positif"))]
    pub rent_amount: i64,

    pub deposit_amount: Option<i64>,
    pub payment_day: Option<i16>,
}

#[derive(Debug, Deserialize)]
pub struct UpdateLeaseRequest {
    pub end_date: Option<NaiveDate>,
    pub rent_amount: Option<i64>,
    pub deposit_amount: Option<i64>,
    pub payment_day: Option<i16>,
    pub status: Option<String>,
    pub contract_pdf_url: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct LeaseResponse {
    pub id: Uuid,
    pub property_id: Uuid,
    pub tenant_id: Uuid,
    pub start_date: NaiveDate,
    pub end_date: Option<NaiveDate>,
    pub rent_amount: i64,
    pub deposit_amount: i64,
    pub payment_day: i16,
    pub status: String,
    pub contract_pdf_url: Option<String>,
    pub signed_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

impl From<Lease> for LeaseResponse {
    fn from(l: Lease) -> Self {
        Self {
            id: l.id,
            property_id: l.property_id,
            tenant_id: l.tenant_id,
            start_date: l.start_date,
            end_date: l.end_date,
            rent_amount: l.rent_amount,
            deposit_amount: l.deposit_amount,
            payment_day: l.payment_day,
            status: l.status,
            contract_pdf_url: l.contract_pdf_url,
            signed_at: l.signed_at,
            created_at: l.created_at,
            updated_at: l.updated_at,
        }
    }
}

#[derive(Debug, FromRow, Serialize)]
pub struct LeaseWithDetails {
    pub id: Uuid,
    pub property_id: Uuid,
    pub tenant_id: Uuid,
    pub start_date: NaiveDate,
    pub end_date: Option<NaiveDate>,
    pub rent_amount: i64,
    pub deposit_amount: i64,
    pub payment_day: i16,
    pub status: String,
    pub contract_pdf_url: Option<String>,
    pub signed_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    // Property info
    pub property_reference: Option<String>,
    pub property_address: String,
    // Tenant info
    pub tenant_first_name: String,
    pub tenant_last_name: String,
}
