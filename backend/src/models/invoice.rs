use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::FromRow;
use uuid::Uuid;

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct Invoice {
    pub id: Uuid,
    pub payment_id: Uuid,
    pub invoice_number: String,
    pub pdf_url: Option<String>,
    pub sent_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Deserialize)]
pub struct CreateInvoiceRequest {
    pub payment_id: Uuid,
    pub invoice_number: String,
    pub pdf_url: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct InvoiceResponse {
    pub id: Uuid,
    pub payment_id: Uuid,
    pub invoice_number: String,
    pub pdf_url: Option<String>,
    pub sent_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
}

impl From<Invoice> for InvoiceResponse {
    fn from(i: Invoice) -> Self {
        Self {
            id: i.id,
            payment_id: i.payment_id,
            invoice_number: i.invoice_number,
            pdf_url: i.pdf_url,
            sent_at: i.sent_at,
            created_at: i.created_at,
        }
    }
}
