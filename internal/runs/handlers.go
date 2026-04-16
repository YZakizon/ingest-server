package runs

import (
    "net/http"
	// "fmt"
	// "bytes"
	// "io"

    // "github.com/google/uuid"
    "ingest_server/internal/app"
)

// IncomingRun represents one item in the POST body
type IncomingRun struct {
    TraceID  string                 `json:"trace_id"`
    Name     string                 `json:"name"`
    Inputs   map[string]interface{} `json:"inputs"`
    Outputs  map[string]interface{} `json:"outputs"`
    Metadata map[string]interface{} `json:"metadata"`
}

// CreateRunsHandler handles POST /runs
func CreateRunsHandler(w http.ResponseWriter, r *http.Request, state *app.State) {

	// body, _ := io.ReadAll(r.Body)
	// fmt.Println("RAW BODY:", string(body))
	// r.Body = io.NopCloser(bytes.NewReader(body))

    // if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
    //     http.Error(w, "invalid JSON", http.StatusBadRequest)
    //     return
    // }


	createRunsHandler(w, r, state)
}

// GetRunHandler handles GET /runs/{id}
func GetRunHandler(w http.ResponseWriter, r *http.Request, state *app.State, id string) {
	getRunHandler(w, r, state, id)

}
