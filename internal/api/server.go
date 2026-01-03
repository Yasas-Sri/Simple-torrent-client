package api

import (
	"encoding/json"
	"net/http"
	"torrent-client/internal/p2p"
)

type Server struct {
	Manager *p2p.Manager
}

func NewServer(m *p2p.Manager) *Server {
	return &Server{Manager: m}
}

func (s *Server) Start() {

	http.HandleFunc("/stats", s.handleStats)

	http.HandleFunc("/add", s.handleAdd)

	go http.ListenAndServe(":8080", nil)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	stats := s.Manager.GetStats()
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) handleAdd(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Only POST is allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TorrentData []byte `json:"torrentData"`
		URL         string `json:"url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.TorrentData) > 0 {
		err := s.Manager.AddTorrent(req.TorrentData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status":"added"}`))
}
