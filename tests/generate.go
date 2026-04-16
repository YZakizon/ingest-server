package tests

import (
    "strings"

    "github.com/google/uuid"
    "ingest_server/internal/runs"
)

func generateBatchRuns(count int, sizeKB int) []runs.Run {
    runsList := make([]runs.Run, count)

    payload := strings.Repeat("x", sizeKB*1024)

    for i := 0; i < count; i++ {
        runsList[i] = runs.Run{
            TraceID: uuid.New(),
            Name:    "benchmark_run",
            Inputs: map[string]any{
                "payload": payload,
            },
            Outputs: map[string]any{
                "result": "ok",
            },
            Metadata: map[string]any{
                "index": i,
            },
        }
    }

    return runsList
}
