# Run the Go app locally using RDS
run:
	go run cmd/api/main.go

# Build the app Docker image
build:
	$(MAKE) swagger
	docker build -t gold-digger -f Dockerfile .

# Run the app in Docker (connects to cloud DB via env vars)
up:
	docker-compose up --build

# Start only the TimescaleDB container
tsdb:
	docker-compose up -d timescaledb

# Stop the app container
down:
	docker-compose down --remove-orphans

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
