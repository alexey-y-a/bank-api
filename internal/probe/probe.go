package probe

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
)

type ReadyProbe struct {
	ready atomic.Bool
}

func NewReadyProbe() *ReadyProbe {
	return &ReadyProbe{}
}

func (p *ReadyProbe) MarkReady() {
	p.ready.Store(true)
}

func (p *ReadyProbe) UnmarkReady() {
	p.ready.Store(false)
}

func (p *ReadyProbe) IsReady() bool {
	return p.ready.Load()
}

func (p *ReadyProbe) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if p.IsReady() {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
			return
		}

		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "not ready"})
	}
}
