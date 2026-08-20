FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod ./
COPY main.go ./

RUN go build -o api .

FROM debian:bookworm-slim

WORKDIR /app

COPY --from=builder /app/api .

EXPOSE 8082

CMD ["./api"]