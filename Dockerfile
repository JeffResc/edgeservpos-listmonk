# Stage 1: Build the Go binary
FROM golang:1.25 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY main.go .

RUN CGO_ENABLED=0 GOOS=linux go build -o app main.go

# Stage 2: Create a minimal image with distroless
FROM gcr.io/distroless/static@sha256:4b2a093ef4649bccd586625090a3c668b254cfe180dee54f4c94f3e9bd7e381e

COPY --from=builder /app/app /

ENTRYPOINT ["/app"]
