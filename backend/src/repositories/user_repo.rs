use sqlx::PgPool;
use uuid::Uuid;

use crate::models::user::User;

pub async fn find_by_email(pool: &PgPool, email: &str) -> Result<Option<User>, sqlx::Error> {
    let user = sqlx::query_as::<_, User>(
        "SELECT id, organization_id, email, password_hash, role, first_name, last_name, phone, is_active, created_at, updated_at
         FROM users WHERE email = $1"
    )
    .bind(email)
    .fetch_optional(pool)
    .await?;
    Ok(user)
}

pub async fn find_by_id(pool: &PgPool, id: Uuid) -> Result<Option<User>, sqlx::Error> {
    let user = sqlx::query_as::<_, User>(
        "SELECT id, organization_id, email, password_hash, role, first_name, last_name, phone, is_active, created_at, updated_at
         FROM users WHERE id = $1"
    )
    .bind(id)
    .fetch_optional(pool)
    .await?;
    Ok(user)
}

pub async fn create(
    pool: &PgPool,
    organization_id: Uuid,
    email: &str,
    password_hash: &str,
    role: &str,
    first_name: &str,
    last_name: &str,
    phone: Option<&str>,
) -> Result<User, sqlx::Error> {
    let user = sqlx::query_as::<_, User>(
        "INSERT INTO users (organization_id, email, password_hash, role, first_name, last_name, phone)
         VALUES ($1, $2, $3, $4, $5, $6, $7)
         RETURNING id, organization_id, email, password_hash, role, first_name, last_name, phone, is_active, created_at, updated_at"
    )
    .bind(organization_id)
    .bind(email)
    .bind(password_hash)
    .bind(role)
    .bind(first_name)
    .bind(last_name)
    .bind(phone)
    .fetch_one(pool)
    .await?;
    Ok(user)
}

pub async fn update_password(pool: &PgPool, id: Uuid, password_hash: &str) -> Result<(), sqlx::Error> {
    sqlx::query("UPDATE users SET password_hash = $1 WHERE id = $2")
        .bind(password_hash)
        .bind(id)
        .execute(pool)
        .await?;
    Ok(())
}
