package tests

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "ingest_server/internal/app"
    "ingest_server/internal/mocks"
    "ingest_server/internal/runs"
)

func setupBenchmarkServer() *httptest.Server {
    state := &app.State{
        DB:    mocks.NewMockDB(),
        S3:    mocks.NewMockS3(),
        Redis: mocks.NewMockRedis(),
    }

    mux := http.NewServeMux()
    runs.RegisterRoutes(mux, state)

    return httptest.NewServer(mux)
}

func benchmarkCreateRuns(b *testing.B, batchSize, sizeKB int) {
    ts := setupBenchmarkServer()
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
            b.Fatalf("unexpected status: %d", resp.StatusCode)
        }
        resp.Body.Close()
    }
	msPerOp := float64(b.Elapsed().Nanoseconds()) / float64(b.N) / 1e6
b.ReportMetric(msPerOp, "ms/op")


}

func BenchmarkCreateRuns_500_10KB(b *testing.B) {
    benchmarkCreateRuns(b, 500, 10)
}

func BenchmarkCreateRuns_50_100KB(b *testing.B) {
    benchmarkCreateRuns(b, 50, 100)
}

func BenchmarkCreateRuns_500_100KB(b *testing.B) {
    benchmarkCreateRuns(b, 500, 100)
}
