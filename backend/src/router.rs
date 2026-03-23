use std::sync::Arc;

use axum::{
    routing::{get, post},
    Router,
};

use crate::{
    handlers::{auth, dashboard, leases, payments, properties, tenants},
    middleware::{cors::cors_layer, logger::trace_layer},
    AppState,
};

pub fn create_router(state: Arc<AppState>) -> Router {
    let protected = Router::new()
        // Properties
        .route("/properties", get(properties::list).post(properties::create))
        .route(
            "/properties/{id}",
            get(properties::get_by_id)
                .put(properties::update)
                .delete(properties::delete),
        )
        // Tenants
        .route("/tenants", get(tenants::list).post(tenants::create))
        .route(
            "/tenants/{id}",
            get(tenants::get_by_id)
                .put(tenants::update)
                .delete(tenants::delete),
        )
        // Leases
        .route("/leases", get(leases::list).post(leases::create))
        .route(
            "/leases/{id}",
            get(leases::get_by_id).put(leases::update),
        )
        .route("/leases/{id}/terminate", post(leases::terminate))
        // Payments
        .route("/payments", get(payments::list).post(payments::create))
        .route("/payments/{id}", get(payments::get_by_id))
        .route("/payments/overdue", get(payments::overdue))
        .route("/payments/remind", post(payments::remind))
        // Dashboard
        .route("/dashboard/stats", get(dashboard::stats))
        .route("/dashboard/revenue", get(dashboard::revenue))
        .route("/dashboard/occupancy", get(dashboard::occupancy));

    let auth_routes = Router::new()
        .route("/register", post(auth::register))
        .route("/login", post(auth::login))
        .route("/refresh", post(auth::refresh))
        .route("/me", get(auth::me));

    Router::new()
        .route("/api/health", get(health_check))
        .nest("/api/v1/auth", auth_routes)
        .nest("/api/v1", protected)
        .layer(trace_layer())
        .layer(cors_layer())
        .with_state(state)
}

async fn health_check(
    axum::extract::State(state): axum::extract::State<Arc<AppState>>,
) -> axum::Json<serde_json::Value> {
    let db_ok = crate::db::healthcheck(&state.db).await;
    axum::Json(serde_json::json!({
        "status": if db_ok { "ok" } else { "degraded" },
        "database": db_ok,
        "service": "lokatik-backend",
    }))
}
