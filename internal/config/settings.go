package config
import (
    "os"
    "strconv"

    "github.com/joho/godotenv"
)
type S3Settings struct {
    BucketName      string
    EndpointURL     string
    AccessKey       string
    SecretKey       string
    Region          string
    MultipartWorkers int
}

type Settings struct {
    AppTitle       string
    AppDescription string
    AppVersion     string

    DBHost     string
    DBPort     int
    DBUser     string
    DBPassword string
    DBName     string

    S3BucketName  string
    S3EndpointURL string
    S3AccessKey   string
    S3SecretKey   string
    S3Region      string

    S3 S3Settings

    RedisCacheHost string
    RedisCachePort int
}

var SettingsInstance = Settings{
    AppTitle:       "LS Run Handler",
    AppDescription: "A simple Go server with run endpoints",
    AppVersion:     "0.1.0",

    DBHost:     "localhost",
    DBPort:     5432,
    DBUser:     "postgres",
    DBPassword: "postgres",
    DBName:     "postgres",

    S3BucketName:  "runs",
    S3EndpointURL: "http://localhost:9002",
    S3AccessKey:   "minioadmin1",
    S3SecretKey:   "minioadmin1",
    S3Region:      "us-east-1",

    S3: S3Settings{
        BucketName:      "runs",
        EndpointURL:     "http://localhost:9002",
        AccessKey:       "minioadmin1",
        SecretKey:       "minioadmin1",
        Region:          "us-east-1",
        MultipartWorkers: 4,
    },

    RedisCacheHost: "localhost",
    RedisCachePort: 6379,
}

func LoadTestEnv() {
    // Load .env.test if present
    _ = godotenv.Load(".env.test")

    // Helper to override string fields
    setString := func(env string, target *string) {
        if v := os.Getenv(env); v != "" {
            *target = v
        }
    }

    // Helper to override int fields
    setInt := func(env string, target *int) {
        if v := os.Getenv(env); v != "" {
            if i, err := strconv.Atoi(v); err == nil {
                *target = i
            }
        }
    }

    // --- DB ---
    setString("DB_HOST", &SettingsInstance.DBHost)
    setInt("DB_PORT", &SettingsInstance.DBPort)
    setString("DB_USER", &SettingsInstance.DBUser)
    setString("DB_PASSWORD", &SettingsInstance.DBPassword)
    setString("DB_NAME", &SettingsInstance.DBName)

    // --- Redis ---
    setString("REDIS_CACHE_HOST", &SettingsInstance.RedisCacheHost)
    setInt("REDIS_CACHE_PORT", &SettingsInstance.RedisCachePort)

    // --- S3 ---
    setString("S3_BUCKET_NAME", &SettingsInstance.S3BucketName)
    setString("S3_ENDPOINT_URL", &SettingsInstance.S3EndpointURL)
    setString("S3_ACCESS_KEY", &SettingsInstance.S3AccessKey)
    setString("S3_SECRET_KEY", &SettingsInstance.S3SecretKey)
    setString("S3_REGION", &SettingsInstance.S3Region)

    // Nested S3 struct
    setString("S3_BUCKET_NAME", &SettingsInstance.S3.BucketName)
    setString("S3_ENDPOINT_URL", &SettingsInstance.S3.EndpointURL)
    setString("S3_ACCESS_KEY", &SettingsInstance.S3.AccessKey)
    setString("S3_SECRET_KEY", &SettingsInstance.S3.SecretKey)
    setString("S3_REGION", &SettingsInstance.S3.Region)
    setInt("S3_MULTIPART_WORKERS", &SettingsInstance.S3.MultipartWorkers)
}