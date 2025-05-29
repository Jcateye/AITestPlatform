package datastore

import (
	"database/sql"
	"encoding/json"
	"time"
)

// EvaluationJob maps to the evaluation_jobs table in the database.
type EvaluationJob struct {
	ID              int             `json:"id"`
	JobName         sql.NullString  `json:"job_name,omitempty"` // Nullable string
	JobType         string          `json:"job_type"`           // e.g., ASR, TTS, LLM
	Status          string          `json:"status"`             // e.g., PENDING, RUNNING, COMPLETED, FAILED
	VendorConfigIDs json.RawMessage `json:"vendor_config_ids"`  // JSONB array of vendor_config_id
	TestCaseIDs     json.RawMessage `json:"test_case_ids"`      // JSONB array of test_case_id (or prompt_ids for LLM)
	Parameters      json.RawMessage `json:"parameters,omitempty"` // Specific parameters for this job run
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	StartedAt       sql.NullTime    `json:"started_at,omitempty"`
	CompletedAt     sql.NullTime    `json:"completed_at,omitempty"`
}

// Helper to marshal []int to json.RawMessage
func MarshalIntSliceToJSON(ids []int) (json.RawMessage, error) {
	if ids == nil {
		// Return JSON 'null' if the slice is nil, or an empty array '[]' if preferred
		return json.RawMessage("[]"), nil // Or "null"
	}
	bytes, err := json.Marshal(ids)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(bytes), nil
}

// Helper to unmarshal json.RawMessage to []int
func UnmarshalJSONToIntSlice(data json.RawMessage) ([]int, error) {
	if data == nil || string(data) == "null" || string(data) == "" {
		return []int{}, nil // Return empty slice for null or empty JSON
	}
	var ids []int
	if err := json.Unmarshal(data, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}
