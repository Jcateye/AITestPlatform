package datastore

import (
	"database/sql"
	"encoding/json"
	"time"
)

// ASRTestCase maps to the asr_test_cases table in the database.
type ASRTestCase struct {
	ID              int             `json:"id"`
	Name            string          `json:"name"`
	LanguageCode    sql.NullString  `json:"language_code,omitempty"`
	AudioFilePath   string          `json:"audio_file_path"` // Path/key in object storage
	GroundTruthText sql.NullString  `json:"ground_truth_text,omitempty"`
	Tags            json.RawMessage `json:"tags,omitempty"` // e.g., ["short_audio", "noisy"]
	Description     sql.NullString  `json:"description,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}
