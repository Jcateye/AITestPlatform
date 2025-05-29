package datastore

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// CreateASREvaluationResult inserts a new ASR evaluation result into the database.
func CreateASREvaluationResult(result *ASREvaluationResult) (int, error) {
	if DB == nil {
		return 0, errors.New("database connection not initialized")
	}

	query := `
		INSERT INTO asr_evaluation_results (
			job_id, asr_test_case_id, vendor_config_id, 
			recognized_text, cer, wer, ser, latency_ms, 
			raw_vendor_response, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`
	result.CreatedAt = time.Now()

	var rawResponseJSON []byte
	if result.RawVendorResponse != nil && len(result.RawVendorResponse) > 0 {
		rawResponseJSON = result.RawVendorResponse
	} else {
		rawResponseJSON = json.RawMessage("null") // Store as SQL NULL if empty or nil
	}

	var id int
	err := DB.QueryRow(
		query,
		result.JobID,
		result.ASRTestCaseID,
		result.VendorConfigID,
		result.RecognizedText,
		result.CER,
		result.WER,
		result.SER, // Optional for MVP, will be sql.NullFloat64
		result.LatencyMs,
		rawResponseJSON,
		result.CreatedAt,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("failed to create ASR evaluation result: %w", err)
	}
	result.ID = id
	return id, nil
}

// GetASREvaluationResultsForJob retrieves all ASR evaluation results for a given job ID.
func GetASREvaluationResultsForJob(jobID int) ([]*ASREvaluationResult, error) {
	if DB == nil {
		return nil, errors.New("database connection not initialized")
	}

	query := `
		SELECT id, job_id, asr_test_case_id, vendor_config_id, 
		       recognized_text, cer, wer, ser, latency_ms, 
		       raw_vendor_response, created_at
		FROM asr_evaluation_results
		WHERE job_id = $1
		ORDER BY created_at ASC
	`

	rows, err := DB.Query(query, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to query ASR evaluation results for job ID %d: %w", jobID, err)
	}
	defer rows.Close()

	results := []*ASREvaluationResult{}
	for rows.Next() {
		res := &ASREvaluationResult{}
		var rawResponseJSON []byte
		if err := rows.Scan(
			&res.ID,
			&res.JobID,
			&res.ASRTestCaseID,
			&res.VendorConfigID,
			&res.RecognizedText,
			&res.CER,
			&res.WER,
			&res.SER,
			&res.LatencyMs,
			&rawResponseJSON,
			&res.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan ASR evaluation result row for job ID %d: %w", jobID, err)
		}
		if rawResponseJSON != nil && string(rawResponseJSON) != "null" {
			res.RawVendorResponse = json.RawMessage(rawResponseJSON)
		}
		results = append(results, res)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration for ASR evaluation results (job ID %d): %w", jobID, err)
	}

	return results, nil
}
