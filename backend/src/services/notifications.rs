use chrono::NaiveDate;

use crate::errors::AppError;

pub async fn send_sms(phone: &str, message: &str) -> Result<(), AppError> {
    tracing::info!(
        phone = phone,
        message = message,
        "SMS notification (stub): would send SMS"
    );
    Ok(())
}

pub async fn send_whatsapp(phone: &str, message: &str) -> Result<(), AppError> {
    tracing::info!(
        phone = phone,
        message = message,
        "WhatsApp notification (stub): would send WhatsApp message"
    );
    Ok(())
}

pub async fn send_payment_reminder(
    tenant_phone: &str,
    amount: i64,
    due_date: NaiveDate,
) -> Result<(), AppError> {
    let message = format!(
        "Rappel: Votre loyer de {} FCFA est du le {}. Merci de regulariser votre situation.",
        amount, due_date
    );
    tracing::info!(
        phone = tenant_phone,
        amount = amount,
        due_date = %due_date,
        "Payment reminder (stub): {}",
        message
    );
    Ok(())
}
