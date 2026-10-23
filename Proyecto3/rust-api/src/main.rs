use axum::{
    extract::State,
    http::StatusCode,
    routing::{get, post},
    Json, Router,
};
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use std::{sync::Arc, time::Duration};

const COUNTRIES: [&str; 5] = ["USA", "RUS", "CHN", "ESP", "GTM"];

#[derive(Deserialize, Serialize)]
struct Report {
    country: String,
    warplanes_in_air: i32,
    warships_in_water: i32,
    timestamp: String,
}

struct AppState {
    client: reqwest::Client,
    go_url: String,
    dapr_url: String,
}

#[tokio::main]
async fn main() {
    let go_url = std::env::var("GO_URL")
        .unwrap_or_else(|_| "http://go-deployment1:8081/report".to_string());
    let dapr_url = std::env::var("DAPR_URL")
        .unwrap_or_else(|_| "http://dapr-publisher:8090/publish".to_string());
    let client = reqwest::Client::builder()
        .timeout(Duration::from_secs(5))
        .pool_max_idle_per_host(100)
        .build()
        .unwrap();
    let state = Arc::new(AppState { client, go_url, dapr_url });

    let app = Router::new()
        .route("/health", get(|| async { "ok" }))
        // Gateway: /grpc-201800632 -> "/"   |   /dapr-201800632 -> "/dapr"
        .route("/", post(handle_grpc))
        .route("/dapr", post(handle_dapr))
        .with_state(state);

    let listener = tokio::net::TcpListener::bind("0.0.0.0:8080").await.unwrap();
    println!("rust-api escuchando en :8080");
    axum::serve(listener, app).await.unwrap();
}

fn validate(r: &Report) -> Result<(), String> {
    if !COUNTRIES.contains(&r.country.as_str()) {
        return Err(format!("country invalido: {}", r.country));
    }
    if !(0..=50).contains(&r.warplanes_in_air) {
        return Err("warplanes_in_air fuera de rango (0-50)".into());
    }
    if !(0..=30).contains(&r.warships_in_water) {
        return Err("warships_in_water fuera de rango (0-30)".into());
    }
    chrono::DateTime::parse_from_rfc3339(&r.timestamp)
        .map(|_| ())
        .map_err(|_| "timestamp no es RFC3339".to_string())
}

async fn forward(st: &AppState, url: &str, report: &Report) -> (StatusCode, Json<Value>) {
    match st.client.post(url).json(report).send().await {
        Ok(resp) => {
            let code = StatusCode::from_u16(resp.status().as_u16())
                .unwrap_or(StatusCode::BAD_GATEWAY);
            (code, Json(json!({ "forwarded": true, "upstream_status": code.as_u16() })))
        }
        Err(e) => (
            StatusCode::BAD_GATEWAY,
            Json(json!({ "error": format!("upstream no responde: {e}") })),
        ),
    }
}

async fn handle_grpc(
    State(st): State<Arc<AppState>>,
    Json(report): Json<Report>,
) -> (StatusCode, Json<Value>) {
    if let Err(e) = validate(&report) {
        return (StatusCode::BAD_REQUEST, Json(json!({ "error": e })));
    }
    forward(&st, &st.go_url, &report).await
}

async fn handle_dapr(
    State(st): State<Arc<AppState>>,
    Json(report): Json<Report>,
) -> (StatusCode, Json<Value>) {
    if let Err(e) = validate(&report) {
        return (StatusCode::BAD_REQUEST, Json(json!({ "error": e })));
    }
    forward(&st, &st.dapr_url, &report).await
}
