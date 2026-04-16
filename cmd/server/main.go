package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "time"

    "ingest_server/internal/app"
    "ingest_server/internal/config"
    "ingest_server/internal/runs"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/redis/go-redis/v9"

    "github.com/aws/aws-sdk-go-v2/aws"
    awsconfig "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/credentials"
    s3sdk "github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
    log.Println("Starting Go ingestion server...")

    ctx := context.Background()
    settings := config.SettingsInstance

    // ------------------------------------------------------------
    // Postgres pool
    // ------------------------------------------------------------
    dbURL := fmt.Sprintf(
        "postgres://%s:%s@%s:%d/%s",
        settings.DBUser,
        settings.DBPassword,
        settings.DBHost,
        settings.DBPort,
        settings.DBName,
    )

    dbpool, err := pgxpool.New(ctx, dbURL)
    if err != nil {
        log.Fatalf("failed to create db pool: %v", err)
    }
    defer dbpool.Close()

    // ------------------------------------------------------------
    // MinIO S3 client (correct configuration)
    // ------------------------------------------------------------
    resolver := aws.EndpointResolverWithOptionsFunc(
        func(service, region string, options ...interface{}) (aws.Endpoint, error) {
            return aws.Endpoint{
                URL:               settings.S3EndpointURL, // e.g. http://localhost:9000
                HostnameImmutable: true,
                SigningRegion:     settings.S3Region,      // usually "us-east-1"
            }, nil
        },
    )

    awsCfg, err := awsconfig.LoadDefaultConfig(
        ctx,
        awsconfig.WithRegion(settings.S3Region),
        awsconfig.WithEndpointResolverWithOptions(resolver),
        awsconfig.WithCredentialsProvider(
            credentials.NewStaticCredentialsProvider(
                settings.S3AccessKey,     // minioadmin
                settings.S3SecretKey,     // minioadmin
                "",
            ),
        ),
    )
    if err != nil {
        log.Fatalf("failed to load AWS config: %v", err)
    }

    s3Client := s3sdk.NewFromConfig(awsCfg)

    // Ensure bucket exists (best-effort)
    _, _ = s3Client.CreateBucket(ctx, &s3sdk.CreateBucketInput{
        Bucket: aws.String(settings.S3BucketName),
    })

    // ------------------------------------------------------------
    // Redis client
    // ------------------------------------------------------------
    redisAddr := fmt.Sprintf("%s:%d", settings.RedisCacheHost, settings.RedisCachePort)

    rdb := redis.NewClient(&redis.Options{
        Addr: redisAddr,
    })
    defer rdb.Close()

    // ------------------------------------------------------------
    // App state
    // ------------------------------------------------------------
    state := &app.State{
        DB:    dbpool,
        S3:    s3Client,
        Redis: rdb,
    }

    // ------------------------------------------------------------
    // HTTP server
    // ------------------------------------------------------------
    mux := http.NewServeMux()
    runs.RegisterRoutes(mux, state)

    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"message":"Go Ingestion API"}`))
    })

    srv := &http.Server{
        Addr:    ":8080",
        Handler: mux,
    }

    go func() {
        log.Printf("server listening on %s", srv.Addr)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("server error: %v", err)
        }
    }()

    // graceful shutdown
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, os.Interrupt)
    <-stop

    ctxShutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    if err := srv.Shutdown(ctxShutdown); err != nil {
        log.Printf("server shutdown error: %v", err)
    }
}
