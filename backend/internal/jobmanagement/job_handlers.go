package jobmanagement

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"unified-ai-eval-platform/backend/internal/datastore"

	"github.com/gin-gonic/gin"
)

// CreateASRJobRequest defines the expected payload for creating an ASR job.
type CreateASRJobRequest struct {
	JobName         string          `json:"job_name"` // Optional, can be empty
	TestCaseIDs     []int           `json:"test_case_ids" binding:"required,min=1"`
	VendorConfigIDs []int           `json:"vendor_config_ids" binding:"required,min=1"`
	Parameters      json.RawMessage `json:"parameters"` // Optional, can be null or valid JSON
}

// CreateASRJobHandler handles requests to create and run a new ASR evaluation job.
func CreateASRJobHandler(c *gin.Context) {
	var req CreateASRJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// Validate Parameters if provided
	if req.Parameters != nil && len(req.Parameters) > 0 {
		if !json.Valid(req.Parameters) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "parameters field contains invalid JSON"})
			return
		}
	} else {
		// If parameters are not provided or empty, explicitly set to null for DB
		req.Parameters = json.RawMessage("null")
	}
	
	jobNameSQL := sql.NullString{String: req.JobName, Valid: req.JobName != ""}


	service := NewJobService() // In a real app, this might be injected
	job, err := service.CreateAndRunASRJob(jobNameSQL, req.TestCaseIDs, req.VendorConfigIDs, req.Parameters)

	if err != nil {
		// CreateAndRunASRJob should ideally return specific error types or codes
		// to allow for more granular HTTP status codes here.
		// For now, using 500 for any error from the service.
		if job != nil && job.Status == JobStatusFailed {
			// If the job was created but failed during execution
			c.JSON(http.StatusAccepted, gin.H{ // 202 Accepted, but processing failed. Or use 500.
				"message": "Job initiated but failed during execution.",
				"job":     job,
				"detail":  err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create or run ASR job: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, job) // 201 Created, and processing finished (synchronously)
}

// GetJobHandler handles requests to retrieve a specific evaluation job by its ID.
func GetJobHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID format"})
		return
	}

	job, err := datastore.GetEvaluationJob(id)
	if err != nil {
		if err.Error().Contains("not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve job: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, job)
}

// ListJobsHandler handles requests to list evaluation jobs, optionally filtered by job_type.
func ListJobsHandler(c *gin.Context) {
	jobType := c.Query("job_type") // e.g., /jobs?job_type=ASR

	jobs, err := datastore.ListEvaluationJobs(jobType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list jobs: " + err.Error()})
		return
	}

	if jobs == nil {
		jobs = []*datastore.EvaluationJob{} // Return empty array instead of null
	}

	c.JSON(http.StatusOK, jobs)
}

// GetJobResultsHandler handles requests to retrieve evaluation results for a specific job ID.
// This handler is specific to ASR results for now, based on GetASREvaluationResultsForJob.
// A more generic approach might be needed for different job types in the future.
func GetJobResultsHandler(c *gin.Context) {
	idStr := c.Param("id")
	jobID, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID format"})
		return
	}

	// First, check if the job itself exists to provide a clear error message
	_, err = datastore.GetEvaluationJob(jobID)
	if err != nil {
		if err.Error().Contains("not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Job with ID %d not found", jobID)})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify job existence: " + err.Error()})
		}
		return
	}

	// Assuming this is for ASR jobs, call the ASR-specific results function.
	// If other job types are introduced, this might need to inspect job.JobType
	// and call a different result retrieval function.
	results, err := datastore.GetASREvaluationResultsForJob(jobID)
	if err != nil {
		// This error means the job exists, but results couldn't be fetched.
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve results for job: " + err.Error()})
		return
	}

	if results == nil {
		results = []*datastore.ASREvaluationResult{} // Return empty array
	}

	c.JSON(http.StatusOK, results)
}
