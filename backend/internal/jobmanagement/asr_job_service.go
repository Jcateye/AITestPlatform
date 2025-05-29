package jobmanagement

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"unified-ai-eval-platform/backend/internal/coreengine/evaluationengine"
	"unified-ai-eval-platform/backend/internal/datastore"
)

// JobService provides methods for managing evaluation jobs.
// It might hold a DB connection or other dependencies in a more complex setup.
type JobService struct {
	// DB *sql.DB // Example if DB needed to be passed around
}

// NewJobService creates a new JobService.
func NewJobService() *JobService {
	return &JobService{}
}

const (
	JobStatusPending   = "PENDING"
	JobStatusRunning   = "RUNNING"
	JobStatusCompleted = "COMPLETED"
	JobStatusFailed    = "FAILED"
	JobTypeASR         = "ASR"
)

// CreateAndRunASRJob creates a new ASR evaluation job and runs it synchronously.
func (s *JobService) CreateAndRunASRJob(jobName sql.NullString, testCaseIDs []int, vendorConfigIDs []int, params json.RawMessage) (*datastore.EvaluationJob, error) {
	log.Printf("CreateAndRunASRJob called: Name: %s, TC_IDs: %v, Vendor_IDs: %v", jobName.String, testCaseIDs, vendorConfigIDs)

	// 1. Construct and store the initial job with "PENDING" status.
	vendorConfigIDsJSON, err := datastore.MarshalIntSliceToJSON(vendorConfigIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal vendor_config_ids: %w", err)
	}
	testCaseIDsJSON, err := datastore.MarshalIntSliceToJSON(testCaseIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal test_case_ids: %w", err)
	}

	job := &datastore.EvaluationJob{
		JobName:         jobName,
		JobType:         JobTypeASR,
		Status:          JobStatusPending,
		VendorConfigIDs: vendorConfigIDsJSON,
		TestCaseIDs:     testCaseIDsJSON,
		Parameters:      params, // Assumed to be valid JSON or null
	}

	jobID, err := datastore.CreateEvaluationJob(job)
	if err != nil {
		return nil, fmt.Errorf("failed to create evaluation job in datastore: %w", err)
	}
	job.ID = jobID // Set the ID on the job object
	log.Printf("Job ID %d created with PENDING status.", jobID)

	// 2. Update job status to "RUNNING" and set started_at.
	err = datastore.UpdateEvaluationJobStatus(jobID, JobStatusRunning)
	if err != nil {
		log.Printf("Failed to update job ID %d status to RUNNING: %v. Attempting to mark as FAILED.", jobID, err)
		// Try to mark as FAILED if this update fails
		_ = datastore.UpdateEvaluationJobStatus(jobID, JobStatusFailed)
		_ = datastore.UpdateEvaluationJobTimestamps(jobID, sql.NullTime{}, sql.NullTime{Time: time.Now(), Valid: true}) // Set completed_at
		job.Status = JobStatusFailed // Update local object
		return job, fmt.Errorf("failed to update job status to RUNNING: %w", err)
	}
	job.Status = JobStatusRunning // Update local object

	startTime := time.Now()
	err = datastore.UpdateEvaluationJobTimestamps(jobID, sql.NullTime{Time: startTime, Valid: true}, sql.NullTime{})
	if err != nil {
		log.Printf("Failed to update job ID %d started_at timestamp: %v. Attempting to mark as FAILED.", jobID, err)
		_ = datastore.UpdateEvaluationJobStatus(jobID, JobStatusFailed)
		_ = datastore.UpdateEvaluationJobTimestamps(jobID, sql.NullTime{}, sql.NullTime{Time: time.Now(), Valid: true}) // Set completed_at
		job.Status = JobStatusFailed
		return job, fmt.Errorf("failed to update job started_at: %w", err)
	}
	job.StartedAt = sql.NullTime{Time: startTime, Valid: true} // Update local object
	log.Printf("Job ID %d status updated to RUNNING, started_at set.", jobID)

	// 3. Call the core evaluation engine.
	// This is a synchronous call for MVP.
	evalErr := evaluationengine.RunASREvaluation(jobID, testCaseIDs, vendorConfigIDs)
	completedTime := time.Now()

	// 4. Update job status based on evaluation outcome.
	if evalErr != nil {
		log.Printf("ASR evaluation for Job ID %d failed: %v", jobID, evalErr)
		job.Status = JobStatusFailed
		err = datastore.UpdateEvaluationJobStatus(jobID, JobStatusFailed)
		if err != nil {
			log.Printf("CRITICAL: Failed to update job ID %d status to FAILED after evaluation error: %v", jobID, err)
			// The job finished (with error) but its status in DB might be stuck at RUNNING.
		}
	} else {
		log.Printf("ASR evaluation for Job ID %d completed successfully.", jobID)
		job.Status = JobStatusCompleted
		err = datastore.UpdateEvaluationJobStatus(jobID, JobStatusCompleted)
		if err != nil {
			log.Printf("CRITICAL: Failed to update job ID %d status to COMPLETED: %v", jobID, err)
			// The job finished successfully but its status in DB might be stuck at RUNNING.
		}
	}
	job.CompletedAt = sql.NullTime{Time: completedTime, Valid: true} // Update local object

	// Update completed_at timestamp regardless of success or failure
	tsErr := datastore.UpdateEvaluationJobTimestamps(jobID, sql.NullTime{}, sql.NullTime{Time: completedTime, Valid: true})
	if tsErr != nil {
		log.Printf("CRITICAL: Failed to update job ID %d completed_at timestamp: %v", jobID, tsErr)
	}
	
	// Fetch the final state of the job to return complete information
	finalJob, fetchErr := datastore.GetEvaluationJob(jobID)
	if fetchErr != nil {
		log.Printf("Failed to fetch final job state for ID %d: %v. Returning local job object.", jobID, fetchErr)
		// Fallback to returning the job object we've been updating locally if fetch fails
		// This might not have the latest updated_at from the DB if status/timestamp updates triggered them.
		return job, evalErr // Return the original evaluation error if one occurred
	}

	return finalJob, evalErr // Return the original evaluation error if one occurred
}
