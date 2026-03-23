use axum::extract::FromRequestParts;
use axum::http::request::Parts;
use jsonwebtoken::{decode, DecodingKey, Validation};
use serde::{Deserialize, Serialize};
use std::pin::Pin;
use std::sync::Arc;
use uuid::Uuid;

use crate::errors::AppError;

#[derive(Debug, Clone)]
pub struct AuthUser {
    pub user_id: Uuid,
    pub organization_id: Uuid,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct Claims {
    pub sub: String,
    pub org_id: String,
    pub exp: usize,
}

impl FromRequestParts<Arc<crate::AppState>> for AuthUser {
    type Rejection = AppError;

    fn from_request_parts<'a, 'b, 'c>(
        parts: &'a mut Parts,
        state: &'b Arc<crate::AppState>,
    ) -> Pin<Box<(dyn std::future::Future<Output = Result<AuthUser, AppError>> + Send + 'c)>>
    where
        'a: 'c,
        'b: 'c,
    {
        let jwt_secret = state.config.jwt_secret.clone();
        Box::pin(async move {
            let auth_header = parts
                .headers
                .get("Authorization")
                .and_then(|value| value.to_str().ok())
                .ok_or_else(|| AppError::Unauthorized("Token d'authentification manquant".into()))?;

            let token = auth_header
                .strip_prefix("Bearer ")
                .ok_or_else(|| AppError::Unauthorized("Format de token invalide".into()))?;

            let token_data = decode::<Claims>(
                token,
                &DecodingKey::from_secret(jwt_secret.as_bytes()),
                &Validation::default(),
            )
            .map_err(|e| {
                tracing::warn!("JWT decode error: {}", e);
                AppError::Unauthorized("Token invalide ou expiré".into())
            })?;

            let user_id = Uuid::parse_str(&token_data.claims.sub)
                .map_err(|_| AppError::Unauthorized("Token invalide".into()))?;
            let organization_id = Uuid::parse_str(&token_data.claims.org_id)
                .map_err(|_| AppError::Unauthorized("Token invalide".into()))?;

            Ok(AuthUser {
                user_id,
                organization_id,
            })
        })
    }
}
