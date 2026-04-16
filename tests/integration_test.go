package tests

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "ingest_server/internal/app"
    "ingest_server/internal/runs"
)

func TestCreateAndGetRun(t *testing.T) {
    state := app.NewRealStateForTesting() // uses real DB + MinIO + Redis
    mux := http.NewServeMux()
    runs.RegisterRoutes(mux, state)

    ts := httptest.NewServer(mux)
    defer ts.Close()

    // Create run
    payload := generateBatchRuns(1, 10)
    body, _ := json.Marshal(payload)

    resp, err := http.Post(ts.URL+"/runs", "application/json", bytes.NewReader(body))
    if err != nil {
        t.Fatal(err)
    }
    if resp.StatusCode != http.StatusCreated {
        t.Fatalf("unexpected status: %d", resp.StatusCode)
    }

    var out struct {
        RunIDs []string `json:"run_ids"`
    }
    json.NewDecoder(resp.Body).Decode(&out)
    resp.Body.Close()

    // Get run
    getResp, err := http.Get(ts.URL + "/runs/" + out.RunIDs[0])
    if err != nil {
        t.Fatal(err)
    }
    if getResp.StatusCode != http.StatusOK {
        t.Fatalf("unexpected status: %d", getResp.StatusCode)
    }
}
