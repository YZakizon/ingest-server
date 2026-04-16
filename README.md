# LS Run Handler

The **LS Run Handler** is a high‑performance ingestion service designed to accept run metadata, store it in PostgreSQL, and upload run payloads to S3‑compatible storage (e.g., MinIO).  
It includes a full integration test and benchmarking suite that exercises the real stack end‑to‑end.

---

## Features

- 🚀 High‑throughput ingestion endpoint (`POST /runs`)
- 📦 S3/MinIO object storage for run payloads
- 🗄️ PostgreSQL for run metadata
- ⚡ Redis for caching (optional)
- 🧪 Integration benchmarks using real services
- 🛠️ Makefile automation:
  - Create/drop test DB
  - Run migrations
  - Prepare MinIO buckets
  - Run integration benchmarks

---

## Project Structure

```
.
├── cmd/               # Application entrypoints
├── internal/          # Core logic
├── tests/             # Integration benchmarks
├── migration.sql      # Database schema for test environment
├── Makefile           # Automation for test environment + benchmarks
├── .env.test          # Test environment configuration
└── README.md
```

---

## Requirements

- Go 1.21+
- PostgreSQL
- MinIO or any S3‑compatible storage
- Redis
- `mc` (MinIO client)
- `psql` CLI

---

## Environment Configuration

The test environment is configured via `.env.test`:

```
APP_TITLE=LS Run Handler Test
APP_DESCRIPTION=Test instance of the LS Run Handler
APP_VERSION=0.1.0-test

DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=postgres_test

S3_BUCKET_NAME=runs-test
S3_ENDPOINT_URL=http://localhost:9002
S3_ACCESS_KEY=minioadmin1
S3_SECRET_KEY=minioadmin1
S3_REGION=us-east-1

REDIS_CACHE_HOST=localhost
REDIS_CACHE_PORT=6379
```

---



## Makefile Commands

### Drop + Create Test DB

```
make dropdb
make createdb
```

### Run Migration

```
make migrate
```

### Prepare MinIO Bucket

```
make minio-bucket
```

### Full Test Environment Reset

Drops DB → creates DB → runs migration → ensures bucket exists.

```
make testenv
```

### Run Integration Benchmarks

```
make benchmark
```

This runs the real ingestion pipeline against PostgreSQL, MinIO, and Redis.

---

## Integration Benchmarks

Example output:

```
BenchmarkCreateRunsReal_500_10KB-8        130.7 ms/op
BenchmarkCreateRunsReal_50_100KB-8        128.0 ms/op
BenchmarkCreateRunsReal_500_100KB-8      1172.0 ms/op
```

These benchmarks measure:

- JSON decoding  
- NDJSON generation  
- S3 upload  
- PostgreSQL insert  
- Redis caching (if enabled)

---

## Running the Server

Run the database server and minio server:

```
docker compose -f docker-compose.yml up -d
```

```
go run ./cmd/server
```

### POST /runs

Accepts run metadata and payload, writes metadata to Postgres, uploads payload to S3.

---

## Development Workflow

1. Start Postgres, MinIO, Redis  
2. Configure `.env.test`  
3. Run:

```
make testenv
make benchmark
```

4. Iterate on ingestion performance

---

## Troubleshooting

### 500: NoSuchBucket

Your MinIO bucket does not exist.

```
make minio-bucket
```

### 500: relation "runs" does not exist

You forgot to run migrations:

```
make migrate
```

### mc: command not found

Install MinIO client:

```
brew install minio/stable/mc
```

---

## Building the Project

Build the Go binary:

```bash
make build
```

This is similar to run this command below 
and will generate executable `run-handler`:

```bash
go build -o run-handler ./cmd/server
```


Or run directly:

```bash
make run
```

this runs:

```bash
go run ./cmd/server
```

Make sure your environment variables are loaded (for example):

```bash
source .env.test
```

---

## Running the Application

```bash
./run-handler
```

The server starts on the port defined in your environment (commonly `8080`).

---

## POST /runs

Creates one or more runs and uploads associated data to S3/MinIO.

### Example cURL

```bash
curl -X POST http://localhost:8080/runs \
  -H "Content-Type: application/json" \
  -d '{
        "trace_id": "d2f1c8e0-5c3c-4b8d-9b3e-2e4b8f8c9a11",
        "name": "example-run",
        "inputs": { "x": 1 },
        "outputs": { "y": 2 },
        "metadata": { "source": "curl-example" }
      }'
```

### Response (actual server output)

HTTP Status code: 201
```json
{
  "status": "created",
  "run_ids": [
    "uuid-1",
    "uuid-2"
  ]
}
```

- `status` is always `"created"`.
- `run_ids` is an array because the ingestion pipeline may create multiple run records internally.

---

## GET /runs/{id}

Fetches a single run record.  
If Redis is enabled, the result is cached automatically.

### Example cURL

```bash
curl http://localhost:8080/runs/<run_id>
```

Replace `<run_id>` with one of the IDs returned from POST /runs.

### Response (actual server output)

HTTP Status code: 200
```json
{
  "id": "uuid",
  "trace_id": "uuid",
  "name": "example-run",
  "inputs": { "x": 1 },
  "outputs": { "y": 2 },
  "metadata": { "source": "curl-example" }
}
```

This is the full `runData` object returned by your handler.

---
