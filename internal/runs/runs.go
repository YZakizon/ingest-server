package runs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"

	// "strconv"
	"strings"
	"time"

	"ingest_server/internal/app"
	"ingest_server/internal/config"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	// "github.com/redis/go-redis/v9"
	"github.com/segmentio/ksuid"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3sdk "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const (
	singleUploadThreshold = 32 * 1024 * 1024 // 32 MB
	partSize              = 8 * 1024 * 1024  // 8 MB
	maxRetries            = 5
	cacheTTL              = 30 * time.Minute
)

type Run struct {
	TraceID  uuid.UUID      `json:"trace_id"`
	Name     string         `json:"name"`
	Inputs   map[string]any `json:"inputs,omitempty"`
	Outputs  map[string]any `json:"outputs,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type RunResponse struct {
	ID       uuid.UUID      `json:"id"`
	TraceID  uuid.UUID      `json:"trace_id"`
	Name     string         `json:"name"`
	Inputs   map[string]any `json:"inputs"`
	Outputs  map[string]any `json:"outputs"`
	Metadata map[string]any `json:"metadata"`
}

func RegisterRunRoutes(mux *http.ServeMux, app *app.State) {
	mux.HandleFunc("/runs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			createRunsHandler(w, r, app)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/runs/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/runs/")
		if id == "" {
			http.Error(w, "missing run id", http.StatusBadRequest)
			return
		}
		getRunHandler(w, r, app, id)
	})
}

//
// Helpers
//

func makeBatchObjectKey(batchID string) string {
	return "batches/" + batchID + ".ndjson"
}

func runToNDJSONLine(runID uuid.UUID, run Run) []byte {
	payload := map[string]any{
		"id":       runID.String(),
		"trace_id": run.TraceID.String(),
		"name":     run.Name,
		"inputs":   run.Inputs,
		"outputs":  run.Outputs,
		"metadata": run.Metadata,
	}
	b, _ := json.Marshal(payload)
	return append(b, '\n')
}

func GenerateNDJSONAndOffsets(runIDs []uuid.UUID, runs []Run) ([]byte, []int64, []int32) {
	var buf bytes.Buffer
	offsets := make([]int64, len(runs))
	lengths := make([]int32, len(runs))

	var offset int64
	for i, run := range runs {
		line := runToNDJSONLine(runIDs[i], run)
		l := len(line)
		offsets[i] = offset
		lengths[i] = int32(l)
		buf.Write(line)
		offset += int64(l)
	}
	return buf.Bytes(), offsets, lengths
}

func estimateNDJSONSize(runs []Run, sampleSize int) int64 {
	if len(runs) == 0 {
		return 0
	}
	if sampleSize <= 0 {
		sampleSize = 16
	}
	sampleCount := int(math.Min(float64(sampleSize), float64(len(runs))))
	dummyID := uuid.New()

	var total int64
	for i := 0; i < sampleCount; i++ {
		line := runToNDJSONLine(dummyID, runs[i])
		total += int64(len(line))
	}
	avg := float64(total) / float64(sampleCount)
	return int64(avg * float64(len(runs)))
}

//
// S3 Upload Helpers
//

func uploadSinglePut(ctx context.Context, s3 app.S3, bucket, key string, ndjson []byte) error {
	_, err := s3.PutObject(ctx, &s3sdk.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(ndjson),
		ContentType: aws.String("application/x-ndjson"),
	})
	return err
}

func uploadPartWithRetry(
	ctx context.Context,
	s3 app.S3,
	bucket, key, uploadID string,
	partNumber int32,
	body []byte,
) (string, error) {
	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		out, err := s3.UploadPart(ctx, &s3sdk.UploadPartInput{
			Bucket:     aws.String(bucket),
			Key:        aws.String(key),
			UploadId:   aws.String(uploadID),
			PartNumber: aws.Int32(partNumber),
			Body:       bytes.NewReader(body),
		})
		if err == nil {
			return aws.ToString(out.ETag), nil
		}
		lastErr = err
		sleep := time.Duration(math.Pow(2, float64(attempt))) * time.Second
		sleep += time.Duration(rand.Intn(1000)) * time.Millisecond
		time.Sleep(sleep)
	}
	return "", lastErr
}

func streamRunsToS3WithOffsets(
	ctx context.Context,
	s3 app.S3,
	bucket, key string,
	runIDs []uuid.UUID,
	runs []Run,
) ([]int64, []int32, error) {

	createOut, err := s3.CreateMultipartUpload(ctx, &s3sdk.CreateMultipartUploadInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		ContentType: aws.String("application/x-ndjson"),
	})
	if err != nil {
		return nil, nil, err
	}
	uploadID := aws.ToString(createOut.UploadId)

	var (
		buffer  bytes.Buffer
		offsets = make([]int64, len(runs))
		lengths = make([]int32, len(runs))
		parts   []types.CompletedPart
		offset  int64
		partNum int32 = 1
	)

	defer func() {
		if err != nil {
			_, _ = s3.AbortMultipartUpload(ctx, &s3sdk.AbortMultipartUploadInput{
				Bucket:   aws.String(bucket),
				Key:      aws.String(key),
				UploadId: aws.String(uploadID),
			})
		}
	}()

	for i, run := range runs {
		line := runToNDJSONLine(runIDs[i], run)
		l := len(line)

		offsets[i] = offset
		lengths[i] = int32(l)
		offset += int64(l)

		buffer.Write(line)

		if buffer.Len() >= partSize {
			etag, upErr := uploadPartWithRetry(ctx, s3, bucket, key, uploadID, partNum, buffer.Bytes())
			if upErr != nil {
				err = upErr
				return nil, nil, err
			}
			parts = append(parts, types.CompletedPart{
				ETag:       aws.String(etag),
				PartNumber: aws.Int32(partNum),
			})
			partNum++
			buffer.Reset()
		}
	}

	if buffer.Len() > 0 {
		etag, upErr := uploadPartWithRetry(ctx, s3, bucket, key, uploadID, partNum, buffer.Bytes())
		if upErr != nil {
			err = upErr
			return nil, nil, err
		}
		parts = append(parts, types.CompletedPart{
			ETag:       aws.String(etag),
			PartNumber: aws.Int32(partNum),
		})
	}

	_, err = s3.CompleteMultipartUpload(ctx, &s3sdk.CompleteMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: parts,
		},
	})
	if err != nil {
		return nil, nil, err
	}

	return offsets, lengths, nil
}

//
// S3 GET by byte range
//

func s3GetLineByOffset(
	ctx context.Context,
	s3 app.S3,
	bucket, key string,
	startOffset int64,
	byteLength int32,
) (map[string]any, error) {

	endOffset := startOffset + int64(byteLength) - 1
	rangeHeader := fmt.Sprintf("bytes=%d-%d", startOffset, endOffset)

	out, err := s3.GetObject(ctx, &s3sdk.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Range:  aws.String(rangeHeader),
	})
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()

	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

//
// Redis helpers
//

func redisRunKey(id uuid.UUID) string {
	return "run:" + id.String()
}

//
// POST /runs
//

func createRunsHandler(w http.ResponseWriter, r *http.Request, app *app.State) {
	ctx := r.Context()
	settings := config.SettingsInstance

	var runs []Run
	if err := json.NewDecoder(r.Body).Decode(&runs); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if len(runs) == 0 {
		http.Error(w, "No runs provided", http.StatusBadRequest)
		return
	}

	batchID := ksuid.New().String()
	objectKey := makeBatchObjectKey(batchID)

	runIDs := make([]uuid.UUID, len(runs))
	for i := range runs {
		runIDs[i] = uuid.New()
	}

	estimatedSize := estimateNDJSONSize(runs, 16)

	var (
		offsets []int64
		lengths []int32
		err     error
	)

	if estimatedSize <= singleUploadThreshold {
		ndjsonBytes, offs, lens := GenerateNDJSONAndOffsets(runIDs, runs)
		offsets, lengths = offs, lens
		err = uploadSinglePut(ctx, app.S3, settings.S3BucketName, objectKey, ndjsonBytes)
	} else {
		offsets, lengths, err = streamRunsToS3WithOffsets(ctx, app.S3, settings.S3BucketName, objectKey, runIDs, runs)
	}

	if err != nil {
		http.Error(w, "failed to upload to S3: "+err.Error(), http.StatusInternalServerError)
		return
	}

	rows := make([][]any, len(runs))
	for i := range runs {
		rows[i] = []any{
			runIDs[i],
			runs[i].TraceID,
			runs[i].Name,
			objectKey,
			offsets[i],
			lengths[i],
		}
	}

	tx, err := app.DB.Begin(ctx)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx)

	_, err = tx.CopyFrom(
		ctx,
		pgx.Identifier{"runs"},
		[]string{"id", "trace_id", "name", "batch_key", "byte_offset", "byte_length"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		http.Error(w, "copy error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		http.Error(w, "commit error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]any{
		"status":  "created",
		"run_ids": runIDsToStrings(runIDs),
	}
	writeJSON(w, http.StatusCreated, resp)
}

func runIDsToStrings(ids []uuid.UUID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}

//
// GET /runs/{id}
//

func getRunHandler(w http.ResponseWriter, r *http.Request, app *app.State, idStr string) {
	ctx := r.Context()
	settings := config.SettingsInstance

	runID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid run id", http.StatusBadRequest)
		return
	}

	key := redisRunKey(runID)

	if app.Redis != nil {
		if cached, err := app.Redis.Get(ctx, key).Bytes(); err == nil && len(cached) > 0 {
			var runData map[string]any
			if err := json.Unmarshal(cached, &runData); err == nil {
				writeJSON(w, http.StatusOK, runData)
				return
			}
		}
	}

	var (
		traceID  uuid.UUID
		name     string
		batchKey string
		byteOff  int64
		byteLen  int32
	)

	row := app.DB.QueryRow(ctx, `
        SELECT trace_id, name, batch_key, byte_offset, byte_length
        FROM runs
        WHERE id = $1
    `, runID)

	if err := row.Scan(&traceID, &name, &batchKey, &byteOff, &byteLen); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "Run not found", http.StatusNotFound)
			return
		}
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	runData, err := s3GetLineByOffset(ctx, app.S3, settings.S3BucketName, batchKey, byteOff, byteLen)
	if err != nil {
		http.Error(w, "s3 error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if app.Redis != nil {
		if b, err := json.Marshal(runData); err == nil {
			_ = app.Redis.Set(ctx, key, b, cacheTTL).Err()
		}
	}

	writeJSON(w, http.StatusOK, runData)
}

//
// JSON helper
//

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
