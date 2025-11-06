package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"canary/internal/config"
	"canary/internal/database"
	"canary/internal/matcher"
	"canary/internal/models"
)

// Hook processes incoming Certspotter webhook events
func Hook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	var event models.CertspotterEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	allDomains := make([]string, 0, len(event.Issuance.DNSNames)+len(event.Endpoints))
	allDomains = append(allDomains, event.Issuance.DNSNames...)
	for _, ep := range event.Endpoints {
		if ep.DNSName != "" {
			allDomains = append(allDomains, ep.DNSName)
		}
	}

	matchedKeywords := matcher.Find(allDomains)
	now := time.Now()

	if len(matchedKeywords) > 0 {
		config.TotalCerts.Add(1)
		log.Printf("Match found: cert_id=%s keywords=%v domains=%v", event.ID, matchedKeywords, allDomains)
	}

	for _, kw := range matchedKeywords {
		config.TotalMatches.Add(1)
		m := models.Match{
			CertID:     event.ID,
			Domains:    allDomains,
			Keyword:    kw,
			Timestamp:  now,
			TbsSha256:  event.Issuance.TbsSha256,
			CertSha256: event.Issuance.CertSha256,
		}

		select {
		case config.MatchChan <- m:
		default:
			log.Printf("match channel full, dropping match cert_id=%s keyword=%s", m.CertID, m.Keyword)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"matches": len(matchedKeywords),
	})
}

// GetMatches returns recent matches from the in-memory cache
func GetMatches(w http.ResponseWriter, r *http.Request) {
	config.CacheMutex.RLock()
	defer config.CacheMutex.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"count":   len(config.RecentMatches),
		"matches": config.RecentMatches,
	})
}

// ClearMatches clears the in-memory matches cache
func ClearMatches(w http.ResponseWriter, r *http.Request) {
	config.CacheMutex.Lock()
	config.RecentMatches = nil
	config.CacheMutex.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "cleared"})
}

// AddKeywords adds keywords via API and reloads the matcher
func AddKeywords(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	var req struct {
		Keywords []string `json:"keywords"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if len(req.Keywords) == 0 {
		http.Error(w, "no keywords", http.StatusBadRequest)
		return
	}

	// Append keywords to file
	if err := matcher.AppendKeywords(config.KeywordsFile, req.Keywords); err != nil {
		http.Error(w, "failed to append keywords", http.StatusInternalServerError)
		return
	}

	// Reload keywords
	if err := matcher.Load(config.KeywordsFile); err != nil {
		http.Error(w, "failed to reload keywords", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":   "keywords added and reloaded",
		"countAdd": len(req.Keywords),
	})
}

// ReloadKeywords reloads keywords from the file
func ReloadKeywords(w http.ResponseWriter, r *http.Request) {
	err := matcher.Load(config.KeywordsFile)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		log.Printf("Failed to reload keywords: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	st := matcher.GetCurrent()
	cnt := 0
	if st != nil {
		cnt = len(st.Keywords)
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "keywords reloaded", "count": fmt.Sprintf("%d", cnt)})
}

// GetRecentFromDB retrieves matches from the last X minutes: /matches/recent?minutes=5
func GetRecentFromDB(w http.ResponseWriter, r *http.Request) {
	minStr := r.URL.Query().Get("minutes")
	if minStr == "" {
		minStr = "5"
	}
	minutes, err := time.ParseDuration(minStr + "m")
	if err != nil {
		http.Error(w, "bad minutes value", http.StatusBadRequest)
		return
	}
	since := time.Now().Add(-minutes)

	all, err := database.GetRecent(since)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"count":   len(all),
		"matches": all,
	})
}

// Metrics returns system metrics
func Metrics(w http.ResponseWriter, r *http.Request) {
	queueLen := 0
	if config.MatchChan != nil {
		queueLen = len(config.MatchChan)
	}

	st := matcher.GetCurrent()
	keywordCount := 0
	if st != nil {
		keywordCount = len(st.Keywords)
	}

	uptime := time.Since(config.StartTime)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"queue_len":      queueLen,
		"total_matches":  config.TotalMatches.Load(),
		"total_certs":    config.TotalCerts.Load(),
		"keyword_count":  keywordCount,
		"uptime_seconds": int(uptime.Seconds()),
		"recent_matches": len(config.RecentMatches),
	})
}

// Health checks system health
func Health(w http.ResponseWriter, r *http.Request) {
	// Check if database is accessible
	if err := config.DB.Ping(); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "unhealthy",
			"error":  "database unreachable",
		})
		return
	}

	// Check if matcher is loaded
	st := matcher.GetCurrent()
	if st == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "unhealthy",
			"error":  "matcher not loaded",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":   "healthy",
		"keywords": len(st.Keywords),
		"uptime":   int(time.Since(config.StartTime).Seconds()),
	})
}
