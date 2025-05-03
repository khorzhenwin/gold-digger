FROM golang:1.24-bullseye AS builder

WORKDIR /app
COPY . .

RUN go build -o gochujang ./cmd/api

FROM debian:bullseye-slim

# Install CA certificates
RUN apt-get update && apt-get install -y ca-certificates

WORKDIR /app
COPY --from=builder /app/gochujang ./gochujang
COPY migrations ./migrations
COPY global-bundle.pem /app/certs/global-bundle.pem

CMD ["./gochujang"]
