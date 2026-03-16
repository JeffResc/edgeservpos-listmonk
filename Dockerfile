# Stage 1: Build the Go binary
FROM golang:1.25 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY main.go .

RUN CGO_ENABLED=0 GOOS=linux go build -o app main.go

# Stage 2: Create a minimal image with distroless
FROM gcr.io/distroless/static@sha256:47b2d72ff90843eb8a768b5c2f89b40741843b639d065b9b937b07cd59b479c6

COPY --from=builder /app/app /

ENTRYPOINT ["/app"]
