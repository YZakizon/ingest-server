package runs

import (
    "net/http"
    "ingest_server/internal/app"
)

func RegisterRoutes(mux *http.ServeMux, state *app.State) {
    mux.HandleFunc("/runs", func(w http.ResponseWriter, r *http.Request) {
        if r.Method == http.MethodPost {
            CreateRunsHandler(w, r, state)
            return
        }
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
    })

    mux.HandleFunc("/runs/", func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
            http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
            return
        }

        id := r.URL.Path[len("/runs/"):]
        if id == "" {
            http.Error(w, "missing run id", http.StatusBadRequest)
            return
        }

        GetRunHandler(w, r, state, id)
    })
}
