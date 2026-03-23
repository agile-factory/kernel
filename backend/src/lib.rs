pub mod config;
pub mod db;
pub mod errors;
pub mod handlers;
pub mod middleware;
pub mod models;
pub mod repositories;
pub mod router;
pub mod services;
pub mod utils;

use sqlx::PgPool;

pub struct AppState {
    pub db: PgPool,
    pub config: config::Config,
}
