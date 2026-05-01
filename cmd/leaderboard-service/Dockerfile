FROM golang:1.26-alpine AS builder
WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/leaderboard-service ./cmd/leaderboard-service

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

COPY --from=builder /bin/leaderboard-service /app/leaderboard-service

EXPOSE 8081
ENTRYPOINT ["/app/leaderboard-service"]
