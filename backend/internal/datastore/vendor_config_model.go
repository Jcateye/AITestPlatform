package datastore

import (
	"database/sql"
	"encoding/json"
	"time"
)

// VendorConfig maps to the vendor_configs table in the database.
type VendorConfig struct {
	ID              int             `json:"id"`
	Name            string          `json:"name"`
	APIType         string          `json:"api_type"` // "ASR", "TTS", "LLM"
	APIKey          sql.NullString  `json:"api_key,omitempty"`
	APISecret       sql.NullString  `json:"api_secret,omitempty"` // Consider encrypting if storing real secrets
	APIEndpoint     sql.NullString  `json:"api_endpoint,omitempty"`
	SupportedModels json.RawMessage `json:"supported_models,omitempty"` // e.g., [{"model_id": "model1", "name": "Model One"}]
	OtherConfigs    json.RawMessage `json:"other_configs,omitempty"`    // Vendor-specific JSON
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}
