use crate::errors::AppError;
use crate::models::lease::Lease;
use crate::models::payment::Payment;
use crate::models::property::Property;
use crate::models::tenant::Tenant;

pub async fn generate_lease_contract(
    lease: &Lease,
    property: &Property,
    tenant: &Tenant,
) -> Result<Vec<u8>, AppError> {
    tracing::info!(
        lease_id = %lease.id,
        property_id = %property.id,
        tenant_id = %tenant.id,
        "PDF generation (stub): would generate lease contract"
    );

    let placeholder = format!(
        "CONTRAT DE BAIL\n\nBien: {}\nLocataire: {} {}\nLoyer: {} FCFA\nDate debut: {}\n",
        property.address,
        tenant.first_name,
        tenant.last_name,
        lease.rent_amount,
        lease.start_date,
    );

    Ok(placeholder.into_bytes())
}

pub async fn generate_receipt(
    payment: &Payment,
    lease: &Lease,
) -> Result<Vec<u8>, AppError> {
    tracing::info!(
        payment_id = %payment.id,
        lease_id = %lease.id,
        "PDF generation (stub): would generate payment receipt"
    );

    let placeholder = format!(
        "QUITTANCE DE LOYER\n\nMontant: {} FCFA\nDate echeance: {}\nStatut: {}\n",
        payment.amount,
        payment.due_date,
        payment.status,
    );

    Ok(placeholder.into_bytes())
}
