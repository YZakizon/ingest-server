# Load environment variables from .env.test
include .env.test
export $(shell sed 's/=.*//' .env.test)

DB_CONN = PGPASSWORD=$(DB_PASSWORD) psql -U $(DB_USER) -h $(DB_HOST)
MC_ALIAS = local

.PHONY: dropdb createdb migrate minio-bucket testenv bench-int

dropdb:
	@echo "Dropping test database '$(DB_NAME)' if it exists..."
	@$(DB_CONN) -tc "SELECT 1 FROM pg_database WHERE datname='$(DB_NAME)'" | grep -q 1 && \
		$(DB_CONN) -c "DROP DATABASE $(DB_NAME);" || \
		echo "Database $(DB_NAME) does not exist, skipping drop."

createdb:
	@echo "Creating test database '$(DB_NAME)'..."
	@$(DB_CONN) -c "CREATE DATABASE $(DB_NAME);"

migrate:
	@echo "Running migration on $(DB_NAME)..."
	@$(DB_CONN) -d $(DB_NAME) -f migrations/migration.sql

minio-bucket:
	@echo "Configuring MinIO alias..."
	@mc alias set $(MC_ALIAS) $(S3_ENDPOINT_URL) $(S3_ACCESS_KEY) $(S3_SECRET_KEY)
	@echo "Creating bucket '$(S3_BUCKET_NAME)' if missing..."
	@mc ls $(MC_ALIAS)/$(S3_BUCKET_NAME) >/dev/null 2>&1 || mc mb $(MC_ALIAS)/$(S3_BUCKET_NAME)

build:
	@echo "Building run-handler binary..."
	@go build -o run-handler ./cmd/server

run:
	@echo "Starting go run server..."
	@go run ./cmd/server

testenv: dropdb createdb migrate minio-bucket
	@echo "Test environment ready."

test:
	@echo "Running tests..."
	@go test ./...

benchmark: testenv
	@echo "Running integration benchmarks..."
	@go test ./tests -bench=CreateRunsReal -run=^$