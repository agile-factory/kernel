use std::sync::Arc;

use lokatik_backend::{config::Config, db, router, AppState};
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt, EnvFilter};

#[tokio::main]
async fn main() {
    dotenvy::dotenv().ok();

    tracing_subscriber::registry()
        .with(EnvFilter::try_from_default_env().unwrap_or_else(|_| EnvFilter::new("info")))
        .with(tracing_subscriber::fmt::layer())
        .init();

    let config = Config::from_env();
    let pool = db::create_pool(&config.database_url).await;

    db::run_migrations(&pool).await;
    tracing::info!("Database migrations applied successfully");

    let state = Arc::new(AppState {
        db: pool,
        config: config.clone(),
    });

    let app = router::create_router(state);

    let addr = config.server_addr();
    tracing::info!("Lokatik server starting on {}", addr);

    let listener = tokio::net::TcpListener::bind(&addr)
        .await
        .expect("Failed to bind address");

    axum::serve(listener, app)
        .await
        .expect("Server failed to start");
}
