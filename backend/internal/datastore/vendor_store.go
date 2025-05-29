package datastore

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	// pq is the PostgreSQL driver
	_ "github.com/lib/pq"
)

// DB is a global database connection pool (for simplicity in this context)
// In a real application, this would be managed more carefully, e.g., passed through context or via dependency injection.
var DB *sql.DB

// InitDB initializes the database connection.
// This is a placeholder; actual connection details would come from config.
func InitDB(dataSourceName string) error {
	var err error
	DB, err = sql.Open("postgres", dataSourceName)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	if err = DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	return nil
}

// CreateVendorConfig inserts a new vendor config into the database and returns its ID.
func CreateVendorConfig(vc *VendorConfig) (int, error) {
	if DB == nil {
		return 0, errors.New("database connection not initialized")
	}

	query := `
		INSERT INTO vendor_configs (name, api_type, api_key, api_secret, api_endpoint, supported_models, other_configs, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`
	vc.CreatedAt = time.Now()
	vc.UpdatedAt = time.Now()

	// Handle potentially nil JSON RawMessage fields
	var supportedModels, otherConfigs []byte
	if vc.SupportedModels != nil {
		supportedModels = vc.SupportedModels
	} else {
		supportedModels = json.RawMessage("null")
	}
	if vc.OtherConfigs != nil {
		otherConfigs = vc.OtherConfigs
	} else {
		otherConfigs = json.RawMessage("null")
	}


	var id int
	err := DB.QueryRow(
		query,
		vc.Name,
		vc.APIType,
		vc.APIKey,
		vc.APISecret,
		vc.APIEndpoint,
		supportedModels,
		otherConfigs,
		vc.CreatedAt,
		vc.UpdatedAt,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("failed to create vendor config: %w", err)
	}
	return id, nil
}

// GetVendorConfig retrieves a vendor config by ID.
func GetVendorConfig(id int) (*VendorConfig, error) {
	if DB == nil {
		return nil, errors.New("database connection not initialized")
	}

	query := `
		SELECT id, name, api_type, api_key, api_secret, api_endpoint, supported_models, other_configs, created_at, updated_at
		FROM vendor_configs
		WHERE id = $1
	`
	vc := &VendorConfig{}
	var supportedModels, otherConfigs []byte // Use []byte for json.RawMessage

	err := DB.QueryRow(query, id).Scan(
		&vc.ID,
		&vc.Name,
		&vc.APIType,
		&vc.APIKey,
		&vc.APISecret,
		&vc.APIEndpoint,
		&supportedModels, // Scan into []byte
		&otherConfigs,   // Scan into []byte
		&vc.CreatedAt,
		&vc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("vendor config with ID %d not found: %w", id, err)
		}
		return nil, fmt.Errorf("failed to get vendor config: %w", err)
	}
	vc.SupportedModels = json.RawMessage(supportedModels)
	vc.OtherConfigs = json.RawMessage(otherConfigs)

	return vc, nil
}

// UpdateVendorConfig updates an existing vendor config.
func UpdateVendorConfig(vc *VendorConfig) error {
	if DB == nil {
		return errors.New("database connection not initialized")
	}

	query := `
		UPDATE vendor_configs
		SET name = $1, api_type = $2, api_key = $3, api_secret = $4, api_endpoint = $5, supported_models = $6, other_configs = $7, updated_at = $8
		WHERE id = $9
	`
	vc.UpdatedAt = time.Now()

	var supportedModels, otherConfigs []byte
	if vc.SupportedModels != nil {
		supportedModels = vc.SupportedModels
	} else {
		supportedModels = json.RawMessage("null")
	}
	if vc.OtherConfigs != nil {
		otherConfigs = vc.OtherConfigs
	} else {
		otherConfigs = json.RawMessage("null")
	}

	result, err := DB.Exec(
		query,
		vc.Name,
		vc.APIType,
		vc.APIKey,
		vc.APISecret,
		vc.APIEndpoint,
		supportedModels,
		otherConfigs,
		vc.UpdatedAt,
		vc.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update vendor config: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("vendor config with ID %d not found for update", vc.ID)
	}

	return nil
}

// DeleteVendorConfig deletes a vendor config by ID.
func DeleteVendorConfig(id int) error {
	if DB == nil {
		return errors.New("database connection not initialized")
	}
	query := "DELETE FROM vendor_configs WHERE id = $1"
	result, err := DB.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete vendor config: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("vendor config with ID %d not found for deletion", id)
	}

	return nil
}

// ListVendorConfigs lists vendor configs, optionally filtered by api_type.
// If apiType is an empty string, all configs are listed.
func ListVendorConfigs(apiType string) ([]*VendorConfig, error) {
	if DB == nil {
		return nil, errors.New("database connection not initialized")
	}

	var rows *sql.Rows
	var err error

	if apiType == "" {
		query := "SELECT id, name, api_type, api_key, api_secret, api_endpoint, supported_models, other_configs, created_at, updated_at FROM vendor_configs ORDER BY created_at DESC"
		rows, err = DB.Query(query)
	} else {
		query := "SELECT id, name, api_type, api_key, api_secret, api_endpoint, supported_models, other_configs, created_at, updated_at FROM vendor_configs WHERE api_type = $1 ORDER BY created_at DESC"
		rows, err = DB.Query(query, apiType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list vendor configs: %w", err)
	}
	defer rows.Close()

	configs := []*VendorConfig{}
	for rows.Next() {
		vc := &VendorConfig{}
		var supportedModels, otherConfigs []byte // Use []byte for json.RawMessage

		if err := rows.Scan(
			&vc.ID,
			&vc.Name,
			&vc.APIType,
			&vc.APIKey,
			&vc.APISecret,
			&vc.APIEndpoint,
			&supportedModels,
			&otherConfigs,
			&vc.CreatedAt,
			&vc.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan vendor config row: %w", err)
		}
		vc.SupportedModels = json.RawMessage(supportedModels)
		vc.OtherConfigs = json.RawMessage(otherConfigs)
		configs = append(configs, vc)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration for vendor configs: %w", err)
	}

	return configs, nil
}
