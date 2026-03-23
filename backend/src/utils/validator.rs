use validator::Validate;

use crate::errors::AppError;

pub fn validate_request<T: Validate>(req: &T) -> Result<(), AppError> {
    req.validate().map_err(|e| {
        let messages: Vec<String> = e
            .field_errors()
            .into_iter()
            .flat_map(|(field, errors)| {
                errors.iter().map(move |err| {
                    match &err.message {
                        Some(msg) => msg.to_string(),
                        None => format!("Champ '{}' invalide", field),
                    }
                })
            })
            .collect();
        AppError::Validation(messages.join(", "))
    })
}
