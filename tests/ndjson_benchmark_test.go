package tests

import (
    "testing"

    "ingest_server/internal/runs"

    "github.com/google/uuid"
)

func BenchmarkGenerateNDJSON(b *testing.B) {
    runsList := generateBatchRuns(500, 100)
    runIDs := make([]uuid.UUID, len(runsList))
    for i := range runIDs {
        runIDs[i] = uuid.New()
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _, _ = runs.GenerateNDJSONAndOffsets(runIDs, runsList)
    }

    msPerOp := float64(b.Elapsed().Nanoseconds()) / float64(b.N) / 1e6
    b.ReportMetric(msPerOp, "ms/op")

}
