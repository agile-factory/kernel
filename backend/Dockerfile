# --- Builder Stage ---
FROM rust:1.82-slim AS builder

RUN apt-get update && apt-get install -y pkg-config libssl-dev && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Cache dependencies
COPY Cargo.toml Cargo.lock* ./
RUN mkdir src && echo "fn main() {}" > src/main.rs
ENV SQLX_OFFLINE=true
RUN cargo build --release 2>/dev/null || true
RUN rm -rf src

# Build actual application
COPY . .
RUN touch src/main.rs
RUN cargo build --release

# --- Runner Stage ---
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/target/release/lokatik-backend /app/lokatik-backend
COPY --from=builder /app/migrations /app/migrations

EXPOSE 3000

CMD ["/app/lokatik-backend"]
