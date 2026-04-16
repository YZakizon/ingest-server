package app

import (
    "context"
    "fmt"

    "ingest_server/internal/config"

    "github.com/aws/aws-sdk-go-v2/aws"
    awsconfig "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/credentials"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/redis/go-redis/v9"
)

// State holds all shared application resources.
// type State struct {
//     DB    *pgxpool.Pool
//     S3    *s3.Client
//     Redis *redis.Client
// }
type State struct {
    DB    DB
    S3    S3
    Redis Redis
}


// NewState is used by cmd/server/main.go.
func NewState(db *pgxpool.Pool, s3Client *s3.Client, redisClient *redis.Client) *State {
    return &State{
        DB:    db,
        S3:    s3Client,
        Redis: redisClient,
    }
}

// NewMockState is used by unit tests and benchmarks.
func NewMockState(db *pgxpool.Pool, s3Client *s3.Client, redisClient *redis.Client) *State {
    return &State{
        DB:    db,
        S3:    s3Client,
        Redis: redisClient,
    }
}

// NewRealStateForTesting creates a full real environment for integration tests.
// It connects to real Postgres, MinIO, and Redis using config.SettingsInstance.
func NewRealStateForTesting() *State {
    ctx := context.Background()
    settings := config.SettingsInstance

    // -----------------------------
    // Postgres
    // -----------------------------
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
        panic(fmt.Errorf("failed to create db pool: %w", err))
    }

    // -----------------------------
    // MinIO S3
    // -----------------------------
    resolver := aws.EndpointResolverWithOptionsFunc(
        func(service, region string, options ...interface{}) (aws.Endpoint, error) {
            return aws.Endpoint{
                URL:               settings.S3EndpointURL,
                HostnameImmutable: true,
                SigningRegion:     settings.S3Region,
            }, nil
        },
    )

    awsCfg, err := awsconfig.LoadDefaultConfig(
        ctx,
        awsconfig.WithRegion(settings.S3Region),
        awsconfig.WithEndpointResolverWithOptions(resolver),
        awsconfig.WithCredentialsProvider(
            credentials.NewStaticCredentialsProvider(
                settings.S3AccessKey,
                settings.S3SecretKey,
                "",
            ),
        ),
    )
    if err != nil {
        panic(fmt.Errorf("failed to load AWS config: %w", err))
    }

    s3Client := s3.NewFromConfig(awsCfg)

    // -----------------------------
    // Redis
    // -----------------------------
    redisAddr := fmt.Sprintf("%s:%d", settings.RedisCacheHost, settings.RedisCachePort)

    rdb := redis.NewClient(&redis.Options{
        Addr: redisAddr,
    })

    return &State{
        DB:    dbpool,
        S3:    s3Client,
        Redis: rdb,
    }
}
