FROM rust:alpine AS build
RUN apk add --no-cache musl-dev build-base
WORKDIR /app
COPY Cargo.toml ./
COPY src ./src
RUN cargo build --release

FROM alpine:3.20
WORKDIR /app
COPY --from=build /app/target/release/rust-api /app/rust-api
EXPOSE 8080
CMD ["/app/rust-api"]
