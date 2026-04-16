package mocks

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
)

// MOCK DB
type MockDB struct{}

func NewMockDB() *MockDB { return &MockDB{} }

func (m *MockDB) Begin(ctx context.Context) (pgx.Tx, error) {
	return &MockTx{}, nil
}

func (m *MockDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return pgx.Row(nil)
}

func (m *MockDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag(""), nil
}

// MOCK S3
type MockS3 struct{}

func NewMockS3() *MockS3 { return &MockS3{} }

func (m *MockS3) PutObject(ctx context.Context, input *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return &s3.PutObjectOutput{}, nil
}

func (m *MockS3) GetObject(ctx context.Context, input *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return &s3.GetObjectOutput{}, nil
}

func (m *MockS3) CreateMultipartUpload(ctx context.Context, input *s3.CreateMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error) {
	return &s3.CreateMultipartUploadOutput{
		UploadId: aws.String("mock-upload-id"),
	}, nil
}

func (m *MockS3) UploadPart(ctx context.Context, input *s3.UploadPartInput, optFns ...func(*s3.Options)) (*s3.UploadPartOutput, error) {
	return &s3.UploadPartOutput{
		ETag: aws.String("mock-etag"),
	}, nil
}

func (m *MockS3) CompleteMultipartUpload(ctx context.Context, input *s3.CompleteMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error) {
	return &s3.CompleteMultipartUploadOutput{}, nil
}

func (m *MockS3) AbortMultipartUpload(ctx context.Context, input *s3.AbortMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, error) {
	return &s3.AbortMultipartUploadOutput{}, nil
}

// MOCK REDIS
type MockRedis struct{}

func NewMockRedis() *MockRedis { return &MockRedis{} }

func (m *MockRedis) Get(ctx context.Context, key string) *redis.StringCmd {
	return redis.NewStringResult("", nil)
}

func (m *MockRedis) Set(ctx context.Context, key string, value any, expiration time.Duration) *redis.StatusCmd {
	return redis.NewStatusResult("OK", nil)
}
