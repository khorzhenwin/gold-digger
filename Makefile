# Run the Go app locally using RDS
.PHONY: run proto ab-test build up tsdb down reset migrate swagger

run:
	go run cmd/api/main.go

# Generate protobuf, gRPC, gateway, and gRPC OpenAPI outputs
proto:
	@echo "🧬 Generating protobuf artifacts..."
	mkdir -p gen docs/openapi/grpc
	GOPATH="$(PWD)/.gopath" GOCACHE="$(PWD)/.gocache" go run github.com/bufbuild/buf/cmd/buf@latest dep update
	GOPATH="$(PWD)/.gopath" GOCACHE="$(PWD)/.gocache" go run github.com/bufbuild/buf/cmd/buf@latest generate

# Compare REST vs gRPC behavior and timings
ab-test:
	./scripts/ab_rest_vs_grpc.sh

# Build the app Docker image
build:
	$(MAKE) swagger
	$(MAKE) proto
	docker build -t gold-digger -f Dockerfile .

# Run the app in Docker (connects to cloud DB via env vars)
up:
	docker-compose up --build

# Start only the TimescaleDB container
tsdb:
	docker-compose up -d timescaledb

# Stop the app container
down:
	docker-compose down -v --remove-orphans

# Reset and rebuild (use only if needed)
reset:
	docker-compose down -v
	docker-compose up --build

# Run migrations (assumes they are inside main.go or AutoMigrate)
migrate:
	go run cmd/api/main.go

# Generate Swagger docs
swagger:
	@echo "📝 Generating Swagger docs..."
	swag init -g cmd/api/api.go --output docs
