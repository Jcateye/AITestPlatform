package datastore

import (
	"database/sql"
	"encoding/json"
	"time"
)

// ASREvaluationResult maps to the asr_evaluation_results table.
type ASREvaluationResult struct {
	ID                int             `json:"id"`
	JobID             int             `json:"job_id"` // Foreign key to evaluation_jobs
	ASRTestCaseID     int             `json:"asr_test_case_id"` // Foreign key to asr_test_cases
	VendorConfigID    int             `json:"vendor_config_id"` // Foreign key to vendor_configs
	RecognizedText    sql.NullString  `json:"recognized_text,omitempty"`
	CER               sql.NullFloat64 `json:"cer,omitempty"`
	WER               sql.NullFloat64 `json:"wer,omitempty"`
	SER               sql.NullFloat64 `json:"ser,omitempty"` // Optional for MVP
	LatencyMs         sql.NullInt64   `json:"latency_ms,omitempty"`
	RawVendorResponse json.RawMessage `json:"raw_vendor_response,omitempty"` // Store the full response
	CreatedAt         time.Time       `json:"created_at"`
}
