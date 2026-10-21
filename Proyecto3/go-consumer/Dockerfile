FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY consumer ./consumer
RUN go mod tidy && CGO_ENABLED=0 go build -o /out/consumer ./consumer

FROM alpine:3.20
COPY --from=build /out/consumer /app/consumer
CMD ["/app/consumer"]
