use std::sync::Arc;

use axum::{extract::State, Json};
use chrono::Utc;
use serde::Serialize;

use crate::errors::AppError;
use crate::middleware::auth::AuthUser;
use crate::repositories::{lease_repo, payment_repo, property_repo, tenant_repo};
use crate::utils::response::ApiResponse;
use crate::AppState;

#[derive(Debug, Serialize)]
pub struct DashboardStats {
    pub total_properties: i64,
    pub occupied_properties: i64,
    pub total_tenants: i64,
    pub monthly_revenue: i64,
    pub overdue_count: i64,
    pub occupancy_rate: f64,
}

#[derive(Debug, Serialize)]
pub struct MonthlyRevenueData {
    pub month: u32,
    pub year: i32,
    pub revenue: i64,
}

#[derive(Debug, Serialize)]
pub struct OccupancyData {
    pub total_properties: i64,
    pub occupied_properties: i64,
    pub available_properties: i64,
    pub maintenance_properties: i64,
    pub occupancy_rate: f64,
}

pub async fn stats(
    State(state): State<Arc<AppState>>,
    auth_user: AuthUser,
) -> Result<Json<ApiResponse<DashboardStats>>, AppError> {
    let org_id = auth_user.organization_id;

    let status_counts = property_repo::count_by_status(&state.db, org_id).await?;
    let total_properties: i64 = status_counts.iter().map(|s| s.count).sum();
    let occupied_properties = status_counts
        .iter()
        .find(|s| s.status == "rented")
        .map(|s| s.count)
        .unwrap_or(0);

    let total_tenants = tenant_repo::count_by_org(&state.db, org_id).await?;

    let now = Utc::now();
    let monthly_revenue = payment_repo::monthly_revenue(
        &state.db,
        org_id,
        now.format("%Y").to_string().parse().unwrap_or(2026),
        now.format("%m").to_string().parse().unwrap_or(1),
    )
    .await?;

    let overdue_count = payment_repo::overdue_count(&state.db, org_id).await?;

    let occupancy_rate = if total_properties > 0 {
        (occupied_properties as f64 / total_properties as f64) * 100.0
    } else {
        0.0
    };

    let stats = DashboardStats {
        total_properties,
        occupied_properties,
        total_tenants,
        monthly_revenue,
        overdue_count,
        occupancy_rate,
    };

    Ok(Json(ApiResponse::success(stats)))
}

pub async fn revenue(
    State(state): State<Arc<AppState>>,
    auth_user: AuthUser,
) -> Result<Json<ApiResponse<Vec<MonthlyRevenueData>>>, AppError> {
    let org_id = auth_user.organization_id;
    let now = Utc::now();
    let current_year = now.format("%Y").to_string().parse::<i32>().unwrap_or(2026);

    let mut revenue_data = Vec::new();
    for month in 1..=12u32 {
        let rev = payment_repo::monthly_revenue(&state.db, org_id, current_year, month as i32).await?;
        revenue_data.push(MonthlyRevenueData {
            month,
            year: current_year,
            revenue: rev,
        });
    }

    Ok(Json(ApiResponse::success(revenue_data)))
}

pub async fn occupancy(
    State(state): State<Arc<AppState>>,
    auth_user: AuthUser,
) -> Result<Json<ApiResponse<OccupancyData>>, AppError> {
    let org_id = auth_user.organization_id;

    let status_counts = property_repo::count_by_status(&state.db, org_id).await?;
    let total_properties: i64 = status_counts.iter().map(|s| s.count).sum();
    let occupied = status_counts
        .iter()
        .find(|s| s.status == "rented")
        .map(|s| s.count)
        .unwrap_or(0);
    let available = status_counts
        .iter()
        .find(|s| s.status == "available")
        .map(|s| s.count)
        .unwrap_or(0);
    let maintenance = status_counts
        .iter()
        .find(|s| s.status == "maintenance")
        .map(|s| s.count)
        .unwrap_or(0);

    let occupancy_rate = if total_properties > 0 {
        (occupied as f64 / total_properties as f64) * 100.0
    } else {
        0.0
    };

    let data = OccupancyData {
        total_properties,
        occupied_properties: occupied,
        available_properties: available,
        maintenance_properties: maintenance,
        occupancy_rate,
    };

    Ok(Json(ApiResponse::success(data)))
}
