package datastore

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// CreateEvaluationJob inserts a new evaluation job into the database.
func CreateEvaluationJob(job *EvaluationJob) (int, error) {
	if DB == nil {
		return 0, errors.New("database connection not initialized")
	}

	query := `
		INSERT INTO evaluation_jobs (job_name, job_type, status, vendor_config_ids, test_case_ids, parameters, created_at, updated_at, started_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`
	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()

	var vendorIDsJSON, testCaseIDsJSON, paramsJSON []byte
	var err error

	if job.VendorConfigIDs != nil {
		vendorIDsJSON = job.VendorConfigIDs
	} else {
		vendorIDsJSON = json.RawMessage("[]") // Default to empty JSON array
	}

	if job.TestCaseIDs != nil {
		testCaseIDsJSON = job.TestCaseIDs
	} else {
		testCaseIDsJSON = json.RawMessage("[]") // Default to empty JSON array
	}
	
	if job.Parameters != nil && len(job.Parameters) > 0 {
		paramsJSON = job.Parameters
	} else {
		paramsJSON = json.RawMessage("null") // Default to SQL NULL
	}


	var id int
	err = DB.QueryRow(
		query,
		job.JobName,
		job.JobType,
		job.Status,
		vendorIDsJSON,
		testCaseIDsJSON,
		paramsJSON,
		job.CreatedAt,
		job.UpdatedAt,
		job.StartedAt,
		job.CompletedAt,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("failed to create evaluation job: %w", err)
	}
	job.ID = id
	return id, nil
}

// GetEvaluationJob retrieves an evaluation job by ID.
func GetEvaluationJob(id int) (*EvaluationJob, error) {
	if DB == nil {
		return nil, errors.New("database connection not initialized")
	}

	query := `
		SELECT id, job_name, job_type, status, vendor_config_ids, test_case_ids, parameters, created_at, updated_at, started_at, completed_at
		FROM evaluation_jobs
		WHERE id = $1
	`
	job := &EvaluationJob{}
	var vendorIDsJSON, testCaseIDsJSON, paramsJSON []byte


	err := DB.QueryRow(query, id).Scan(
		&job.ID,
		&job.JobName,
		&job.JobType,
		&job.Status,
		&vendorIDsJSON,
		&testCaseIDsJSON,
		&paramsJSON,
		&job.CreatedAt,
		&job.UpdatedAt,
		&job.StartedAt,
		&job.CompletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("evaluation job with ID %d not found: %w", id, err)
		}
		return nil, fmt.Errorf("failed to get evaluation job: %w", err)
	}
	job.VendorConfigIDs = json.RawMessage(vendorIDsJSON)
	job.TestCaseIDs = json.RawMessage(testCaseIDsJSON)
	if paramsJSON != nil && string(paramsJSON) != "null" {
		job.Parameters = json.RawMessage(paramsJSON)
	}


	return job, nil
}

// UpdateEvaluationJobStatus updates the status of an evaluation job.
func UpdateEvaluationJobStatus(id int, status string) error {
	if DB == nil {
		return errors.New("database connection not initialized")
	}

	query := `UPDATE evaluation_jobs SET status = $1, updated_at = $2 WHERE id = $3`
	result, err := DB.Exec(query, status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update status for job ID %d: %w", id, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected when updating status for job ID %d: %w", id, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("job ID %d not found for status update", id)
	}
	return nil
}

// UpdateEvaluationJobTimestamps updates the started_at and completed_at timestamps of an evaluation job.
// Use sql.NullTime for parameters to allow setting one or both.
func UpdateEvaluationJobTimestamps(id int, startTime, endTime sql.NullTime) error {
	if DB == nil {
		return errors.New("database connection not initialized")
	}

	// Build query dynamically based on which timestamps are valid
	var querySetClauses []string
	var args []interface{}
	argCount := 1

	if startTime.Valid {
		querySetClauses = append(querySetClauses, fmt.Sprintf("started_at = $%d", argCount))
		args = append(args, startTime)
		argCount++
	}
	if endTime.Valid {
		querySetClauses = append(querySetClauses, fmt.Sprintf("completed_at = $%d", argCount))
		args = append(args, endTime)
		argCount++
	}

	if len(querySetClauses) == 0 {
		return errors.New("no timestamps provided for update")
	}

	querySetClauses = append(querySetClauses, fmt.Sprintf("updated_at = $%d", argCount))
	args = append(args, time.Now())
	argCount++

	args = append(args, id) // For WHERE id = $N

	query := fmt.Sprintf("UPDATE evaluation_jobs SET %s WHERE id = $%d", strings.Join(querySetClauses, ", "), argCount)

	result, err := DB.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update timestamps for job ID %d: %w", id, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected for job ID %d timestamp update: %w", id, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("job ID %d not found for timestamp update", id)
	}
	return nil
}


// ListEvaluationJobs lists evaluation jobs, optionally filtered by job_type.
func ListEvaluationJobs(jobType string) ([]*EvaluationJob, error) {
	if DB == nil {
		return nil, errors.New("database connection not initialized")
	}

	var rows *sql.Rows
	var err error
	baseQuery := "SELECT id, job_name, job_type, status, vendor_config_ids, test_case_ids, parameters, created_at, updated_at, started_at, completed_at FROM evaluation_jobs"
	
	if jobType != "" {
		rows, err = DB.Query(baseQuery+" WHERE job_type = $1 ORDER BY created_at DESC", jobType)
	} else {
		rows, err = DB.Query(baseQuery + " ORDER BY created_at DESC")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list evaluation jobs: %w", err)
	}
	defer rows.Close()

	jobs := []*EvaluationJob{}
	for rows.Next() {
		job := &EvaluationJob{}
		var vendorIDsJSON, testCaseIDsJSON, paramsJSON []byte
		
		if err := rows.Scan(
			&job.ID,
			&job.JobName,
			&job.JobType,
			&job.Status,
			&vendorIDsJSON,
			&testCaseIDsJSON,
			&paramsJSON,
			&job.CreatedAt,
			&job.UpdatedAt,
			&job.StartedAt,
			&job.CompletedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan evaluation job row: %w", err)
		}
		job.VendorConfigIDs = json.RawMessage(vendorIDsJSON)
		job.TestCaseIDs = json.RawMessage(testCaseIDsJSON)
		if paramsJSON != nil && string(paramsJSON) != "null" {
			job.Parameters = json.RawMessage(paramsJSON)
		}
		jobs = append(jobs, job)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration for evaluation jobs: %w", err)
	}

	return jobs, nil
}
