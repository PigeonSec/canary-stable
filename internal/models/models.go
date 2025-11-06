package models

import (
	"time"

	ac "github.com/anknown/ahocorasick"
)

// Match represents a keyword match found in a certificate's domains
type Match struct {
	CertID     string    `json:"cert_id"`
	Domains    []string  `json:"domains"`
	Keyword    string    `json:"keyword"`
	Timestamp  time.Time `json:"timestamp"`
	TbsSha256  string    `json:"tbs_sha256"`
	CertSha256 string    `json:"cert_sha256"`
}

// CertspotterEvent represents the webhook payload from Certspotter
type CertspotterEvent struct {
	ID       string `json:"id"`
	Issuance struct {
		DNSNames   []string `json:"dns_names"`
		TbsSha256  string   `json:"tbs_sha256"`
		CertSha256 string   `json:"cert_sha256"`
	} `json:"issuance"`
	Endpoints []struct {
		DNSName string `json:"dns_name"`
	} `json:"endpoints"`
}

// MatcherState holds the Aho-Corasick automaton and keywords list
type MatcherState struct {
	Machine  ac.Machine
	Keywords []string
}
