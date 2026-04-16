package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"ingest_server/internal/app"
	appconfig "ingest_server/internal/config"
	"ingest_server/internal/runs"
)

func setupRealBenchmarkServer(b *testing.B) *httptest.Server {
	// Load .env.test overrides into SettingsInstance
	appconfig.LoadTestEnv()

	ctx := context.Background()
	s := appconfig.SettingsInstance

	// --- PostgreSQL ---
	dbURL := "postgres://" + s.DBUser + ":" + s.DBPassword +
		"@" + s.DBHost + ":" + strconv.Itoa(s.DBPort) +
		"/" + s.DBName

	dbpool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		b.Fatalf("failed to connect to postgres: %v", err)
	}

	// --- Redis ---
	redisClient := redis.NewClient(&redis.Options{
		Addr: s.RedisCacheHost + ":" + strconv.Itoa(s.RedisCachePort),
	})

	// --- MinIO (S3-compatible) ---
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(s.S3.Region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				s.S3.AccessKey,
				s.S3.SecretKey,
				"",
			),
		),
		config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{
						URL:               s.S3.EndpointURL,
						HostnameImmutable: true,
						PartitionID:       "aws",
						SigningRegion:     s.S3.Region,
					}, nil
				},
			),
		),
	)
	if err != nil {
		b.Fatalf("failed to load AWS config: %v", err)
	}

	s3client := s3.NewFromConfig(awsCfg)

	// --- Build real app state ---
	state := &app.State{
		DB:    dbpool,
		Redis: redisClient,
		S3:    s3client,
	}

	mux := http.NewServeMux()
	runs.RegisterRoutes(mux, state)

	return httptest.NewServer(mux)
}

func benchmarkCreateRunsReal(b *testing.B, batchSize, sizeKB int) {
	ts := setupRealBenchmarkServer(b)
	defer ts.Close()

	runsList := generateBatchRuns(batchSize, sizeKB)
	body, _ := json.Marshal(runsList)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := http.Post(ts.URL+"/runs", "application/json", bytes.NewReader(body))
		if err != nil {
			b.Fatalf("request error: %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			bodyBytes, _ := io.ReadAll(resp.Body)
			b.Fatalf("unexpected status: %d, body: %s", resp.StatusCode, string(bodyBytes))
		}

		resp.Body.Close()
	}

	// ms/op metric
	msPerOp := float64(b.Elapsed().Nanoseconds()) / float64(b.N) / 1e6
	b.ReportMetric(msPerOp, "ms/op")
}

func BenchmarkCreateRunsReal_500_10KB(b *testing.B) {
	benchmarkCreateRunsReal(b, 500, 10)
}

func BenchmarkCreateRunsReal_50_100KB(b *testing.B) {
	benchmarkCreateRunsReal(b, 50, 100)
}

func BenchmarkCreateRunsReal_500_100KB(b *testing.B) {
	benchmarkCreateRunsReal(b, 500, 100)
}
