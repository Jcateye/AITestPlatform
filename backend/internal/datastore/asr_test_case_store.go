package datastore

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	// Ensure pq driver is imported for DB operations, typically done in main or a central db init file.
	// _ "github.com/lib/pq" // Already in vendor_store.go, so accessible in package
)

// CreateASRTestCase inserts a new ASR test case metadata into the database.
func CreateASRTestCase(tc *ASRTestCase) (int, error) {
	if DB == nil {
		return 0, errors.New("database connection not initialized")
	}

	query := `
		INSERT INTO asr_test_cases (name, language_code, audio_file_path, ground_truth_text, tags, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	tc.CreatedAt = time.Now()
	tc.UpdatedAt = time.Now()

	var tagsJSON []byte
	if tc.Tags != nil && len(tc.Tags) > 0 {
		tagsJSON = tc.Tags
	} else {
		tagsJSON = json.RawMessage("null") // Store as SQL NULL if empty or nil
	}

	var id int
	err := DB.QueryRow(
		query,
		tc.Name,
		tc.LanguageCode,
		tc.AudioFilePath,
		tc.GroundTruthText,
		tagsJSON,
		tc.Description,
		tc.CreatedAt,
		tc.UpdatedAt,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("failed to create ASR test case: %w", err)
	}
	return id, nil
}

// GetASRTestCase retrieves an ASR test case by ID.
func GetASRTestCase(id int) (*ASRTestCase, error) {
	if DB == nil {
		return nil, errors.New("database connection not initialized")
	}

	query := `
		SELECT id, name, language_code, audio_file_path, ground_truth_text, tags, description, created_at, updated_at
		FROM asr_test_cases
		WHERE id = $1
	`
	tc := &ASRTestCase{}
	var tagsJSON []byte

	err := DB.QueryRow(query, id).Scan(
		&tc.ID,
		&tc.Name,
		&tc.LanguageCode,
		&tc.AudioFilePath,
		&tc.GroundTruthText,
		&tagsJSON,
		&tc.Description,
		&tc.CreatedAt,
		&tc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("ASR test case with ID %d not found: %w", id, err)
		}
		return nil, fmt.Errorf("failed to get ASR test case: %w", err)
	}
	if tagsJSON != nil && string(tagsJSON) != "null" {
		tc.Tags = json.RawMessage(tagsJSON)
	}


	return tc, nil
}

// ListASRTestCases lists ASR test cases, optionally filtered by language_code and tags.
// languageCode: exact match for language_code.
// tagsQuery: comma-separated string of tags; uses JSONB containment `?&` operator.
func ListASRTestCases(languageCode string, tagsQuery string) ([]*ASRTestCase, error) {
	if DB == nil {
		return nil, errors.New("database connection not initialized")
	}

	var conditions []string
	var args []interface{}
	argID := 1

	if languageCode != "" {
		conditions = append(conditions, fmt.Sprintf("language_code = $%d", argID))
		args = append(args, languageCode)
		argID++
	}

	if tagsQuery != "" {
		tags := strings.Split(tagsQuery, ",")
		var validTags []string
		for _, t := range tags {
			trimmedTag := strings.TrimSpace(t)
			if trimmedTag != "" {
				validTags = append(validTags, trimmedTag)
			}
		}
		if len(validTags) > 0 {
			conditions = append(conditions, fmt.Sprintf("tags ?& $%d::text[]", argID))
			args = append(args, validTags) // Corrected: pass []string directly
			argID++
		}
	}

	query := "SELECT id, name, language_code, audio_file_path, ground_truth_text, tags, description, created_at, updated_at FROM asr_test_cases"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY created_at DESC"

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list ASR test cases: %w", err)
	}
	defer rows.Close()

	testCases := []*ASRTestCase{}
	for rows.Next() {
		tc := &ASRTestCase{}
		var tagsJSON []byte
		if err := rows.Scan(
			&tc.ID,
			&tc.Name,
			&tc.LanguageCode,
			&tc.AudioFilePath,
			&tc.GroundTruthText,
			&tagsJSON,
			&tc.Description,
			&tc.CreatedAt,
			&tc.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan ASR test case row: %w", err)
		}
		if tagsJSON != nil && string(tagsJSON) != "null" {
			tc.Tags = json.RawMessage(tagsJSON)
		}
		testCases = append(testCases, tc)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration for ASR test cases: %w", err)
	}

	return testCases, nil
}


// UpdateASRTestCase updates specific fields of an existing ASR test case.
// tcUpdateData is a map of field names to new values.
// Audio file path is not updated here; should be a separate process if needed.
func UpdateASRTestCase(id int, tcUpdateData map[string]interface{}) (*ASRTestCase, error) {
	if DB == nil {
		return nil, errors.New("database connection not initialized")
	}

	var setClauses []string
	var args []interface{}
	argID := 1

	allowedFields := map[string]string{
		"name":              "string",
		"language_code":     "sql.NullString",
		"ground_truth_text": "sql.NullString",
		"tags":              "json.RawMessage",
		"description":       "sql.NullString",
	}

	for key, value := range tcUpdateData {
		fieldType, ok := allowedFields[key]
		if !ok {
			continue 
		}

		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", key, argID))

		switch fieldType {
		case "sql.NullString":
			if strVal, ok := value.(string); ok && strVal != "" {
				args = append(args, sql.NullString{String: strVal, Valid: true})
			} else {
				args = append(args, sql.NullString{Valid: false})
			}
		case "json.RawMessage":
			if rawMsg, ok := value.(json.RawMessage); ok && len(rawMsg) > 0 && json.Valid(rawMsg) {
				args = append(args, rawMsg)
			} else if strVal, ok := value.(string); ok && strVal != "" { 
				if json.Valid([]byte(strVal)) {
					args = append(args, json.RawMessage(strVal))
				} else {
					args = append(args, json.RawMessage("null")) 
				}
			} else {
				args = append(args, json.RawMessage("null")) 
			}
		default: 
			args = append(args, value)
		}
		argID++
	}

	if len(setClauses) == 0 {
		// If only audio_file_path was intended for update (which is not supported by this func)
		// or if no valid metadata fields were provided.
		// It might be better to fetch and return the existing record or a specific error.
		// For now, returning an error indicating no valid fields for update.
		currentTC, err := GetASRTestCase(id)
		if err != nil {
			return nil, fmt.Errorf("no valid fields provided for update and failed to fetch current test case: %w", err)
		}
		// If no updatable fields are provided, maybe return the current state without error?
		// Or an error "no updatable metadata provided". Let's stick to error.
		return currentTC, errors.New("no updatable metadata fields provided")
	}


	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argID))
	args = append(args, time.Now())
	argID++

	query := fmt.Sprintf("UPDATE asr_test_cases SET %s WHERE id = $%d", strings.Join(setClauses, ", "), argID)
	args = append(args, id)

	result, err := DB.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update ASR test case with ID %d: %w", id, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected for ASR test case ID %d: %w", id, err)
	}
	if rowsAffected == 0 {
		// This could also mean the data provided was the same as existing data,
		// resulting in no actual row change. Some DBs might report 0 in such cases.
		// However, it's more common to indicate the record wasn't found.
		return nil, fmt.Errorf("ASR test case with ID %d not found for update or no data changed", id)
	}

	return GetASRTestCase(id) 
}

// DeleteASRTestCase deletes an ASR test case by ID from the database.
func DeleteASRTestCase(id int) error {
	if DB == nil {
		return errors.New("database connection not initialized")
	}
	query := "DELETE FROM asr_test_cases WHERE id = $1"
	result, err := DB.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete ASR test case with ID %d: %w", id, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected for ASR test case ID %d: %w", id, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("ASR test case with ID %d not found for deletion", id)
	}

	return nil
}

// The pqArray helper type is no longer needed as lib/pq handles []string for text array parameters correctly.
// type pqArray []string
// func (a pqArray) Value() (driver.Value, error) {
//    // ... implementation ...
// }
