FROM golang:1.26-alpine AS builder
WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/numbers-server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

COPY --from=builder /bin/numbers-server /app/numbers-server
COPY --from=builder /app/web /app/web

EXPOSE 8080
ENTRYPOINT ["/app/numbers-server"]
