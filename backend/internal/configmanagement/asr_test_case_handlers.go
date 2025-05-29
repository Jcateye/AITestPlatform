package configmanagement

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"unified-ai-eval-platform/backend/internal/datastore"
	"unified-ai-eval-platform/backend/internal/objectstore"

	"github.com/gin-gonic/gin"
)

const maxUploadSize = 50 << 20 // 50 MB

// CreateASRTestCaseHandler handles the creation of a new ASR test case.
// It expects a multipart/form-data request with an audio file and metadata.
func CreateASRTestCaseHandler(c *gin.Context) {
	// Parse multipart form, 50 MB limit for the entire form
	if err := c.Request.ParseMultipartForm(maxUploadSize); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to parse multipart form: %v. Max size: %d MB", err, maxUploadSize>>20)})
		return
	}

	fileHeader, err := c.FormFile("audio_file")
	if err != nil {
		if err == http.ErrMissingFile {
			c.JSON(http.StatusBadRequest, gin.H{"error": "audio_file is required"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to get audio_file: %v", err)})
		}
		return
	}

	// Validate file size (redundant if ParseMultipartForm is well-behaved, but good for explicit check)
	if fileHeader.Size > maxUploadSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Audio file size exceeds limit of %d MB", maxUploadSize>>20)})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to open uploaded file: %v", err)})
		return
	}
	defer file.Close()

	// Upload to MinIO
	minioClient, err := objectstore.GetGlobalMinioClient()
	if err != nil {
		log.Printf("Error getting Minio client: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Object storage service not available"})
		return
	}

	objectName, err := minioClient.UploadFile(context.Background(), fileHeader.Filename, file, fileHeader.Size, fileHeader.Header.Get("Content-Type"))
	if err != nil {
		log.Printf("Error uploading file to Minio: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to upload audio file: %v", err)})
		return
	}

	// Populate ASRTestCase struct from form data
	var tc datastore.ASRTestCase
	tc.Name = c.PostForm("name")
	tc.AudioFilePath = objectName // Set by MinIO upload

	if tc.Name == "" {
		// If name is empty, MinIO file needs to be deleted to avoid orphaned files
		go func() {
			if err := minioClient.DeleteFile(context.Background(), objectName); err != nil {
				log.Printf("Failed to delete orphaned MinIO object '%s' after validation error: %v", objectName, err)
			}
		}()
		c.JSON(http.StatusBadRequest, gin.H{"error": "name field is required"})
		return
	}

	// Optional fields
	if langCode := c.PostForm("language_code"); langCode != "" {
		tc.LanguageCode = sql.NullString{String: langCode, Valid: true}
	}
	if gtText := c.PostForm("ground_truth_text"); gtText != "" {
		tc.GroundTruthText = sql.NullString{String: gtText, Valid: true}
	}
	if desc := c.PostForm("description"); desc != "" {
		tc.Description = sql.NullString{String: desc, Valid: true}
	}

	tagsStr := c.PostForm("tags") // Expecting a JSON array string e.g., ["short", "noisy"]
	if tagsStr != "" {
		if json.Valid([]byte(tagsStr)) {
			tc.Tags = json.RawMessage(tagsStr)
		} else {
			// Handle invalid JSON for tags - perhaps return error or ignore
			// As above, if we error out, we should delete the uploaded MinIO file.
			go func() {
				if err := minioClient.DeleteFile(context.Background(), objectName); err != nil {
					log.Printf("Failed to delete orphaned MinIO object '%s' after tags validation error: %v", objectName, err)
				}
			}()
			c.JSON(http.StatusBadRequest, gin.H{"error": "tags field contains invalid JSON"})
			return
		}
	} else {
		tc.Tags = json.RawMessage("null") // Default to SQL NULL if not provided
	}

	// Create test case metadata in DB
	id, err := datastore.CreateASRTestCase(&tc)
	if err != nil {
		// Attempt to delete the uploaded file from MinIO if DB operation fails
		go func() {
			if errDel := minioClient.DeleteFile(context.Background(), objectName); errDel != nil {
				log.Printf("CRITICAL: Failed to delete MinIO object '%s' after DB error: %v. DB error was: %v", objectName, errDel, err)
			} else {
				log.Printf("Successfully deleted MinIO object '%s' after DB error.", objectName)
			}
		}()
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create ASR test case metadata: %v", err)})
		return
	}

	tc.ID = id
	// Refetch to get DB-generated timestamps
	createdTC, err := datastore.GetASRTestCase(id)
	if err != nil {
		log.Printf("Failed to refetch ASR Test Case %d after creation: %v", id, err)
		// Fallback to returning tc without timestamps if refetch fails
		c.JSON(http.StatusCreated, tc)
		return
	}

	c.JSON(http.StatusCreated, createdTC)
}

// GetASRTestCaseHandler retrieves a specific ASR test case by its ID.
func GetASRTestCaseHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ASR test case ID format"})
		return
	}

	tc, err := datastore.GetASRTestCase(id)
	if err != nil {
		if err.Error().Contains("not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to retrieve ASR test case: %v", err)})
		}
		return
	}

	c.JSON(http.StatusOK, tc)
}

// ListASRTestCasesHandler lists ASR test cases, with optional filters.
func ListASRTestCasesHandler(c *gin.Context) {
	languageCode := c.Query("language_code")
	tagsQuery := c.Query("tags") // e.g., /asr-test-cases?tags=short,noisy

	tcs, err := datastore.ListASRTestCases(languageCode, tagsQuery)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to list ASR test cases: %v", err)})
		return
	}

	if tcs == nil {
		tcs = []*datastore.ASRTestCase{} // Return empty array instead of null
	}

	c.JSON(http.StatusOK, tcs)
}

// UpdateASRTestCaseHandler updates metadata for an existing ASR test case.
// Does not handle audio file replacement in this version.
func UpdateASRTestCaseHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ASR test case ID format"})
		return
	}

	// Check if test case exists before attempting update
	_, err = datastore.GetASRTestCase(id)
	if err != nil {
		if err.Error().Contains("not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("ASR test case with ID %d not found", id)})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to verify ASR test case: %v", err)})
		}
		return
	}

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request payload: %v", err)})
		return
	}

	// Prevent audio_file_path from being updated via this handler
	if _, ok := updateData["audio_file_path"]; ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "audio_file_path cannot be updated via this endpoint"})
		return
	}
	// also id, created_at, updated_at should not be updatable from payload
	delete(updateData, "id")
	delete(updateData, "created_at")
	delete(updateData, "updated_at")


	updatedTC, err := datastore.UpdateASRTestCase(id, updateData)
	if err != nil {
		if err.Error().Contains("no valid fields provided for update") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else if err.Error().Contains("not found") { // Should be caught by pre-check, but good failsafe
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to update ASR test case: %v", err)})
		}
		return
	}

	c.JSON(http.StatusOK, updatedTC)
}

// DeleteASRTestCaseHandler deletes an ASR test case.
// For MVP, this only deletes the database record. File deletion from MinIO is a future enhancement.
func DeleteASRTestCaseHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ASR test case ID format"})
		return
	}

	// Retrieve test case to get audio_file_path for deletion from MinIO
	tc, err := datastore.GetASRTestCase(id)
	if err != nil {
		if err.Error().Contains("not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("ASR test case with ID %d not found", id)})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to retrieve ASR test case before deletion: %v", err)})
		}
		return
	}


	// Delete metadata from DB
	err = datastore.DeleteASRTestCase(id)
	if err != nil {
		// DB deletion failed, so we don't proceed to MinIO deletion.
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to delete ASR test case metadata: %v", err)})
		return
	}

	// If DB deletion was successful, proceed to delete from MinIO
	if tc.AudioFilePath != "" {
		minioClient, clientErr := objectstore.GetGlobalMinioClient()
		if clientErr != nil {
			log.Printf("Error getting Minio client for file deletion: %v. DB record for ID %d deleted, but MinIO file %s may be orphaned.", clientErr, id, tc.AudioFilePath)
			// Still return success for DB deletion, but log the MinIO issue.
			c.JSON(http.StatusOK, gin.H{"message": "ASR test case metadata deleted successfully, but failed to connect to object storage to remove audio file."})
			return
		}

		err = minioClient.DeleteFile(context.Background(), tc.AudioFilePath)
		if err != nil {
			log.Printf("Failed to delete audio file '%s' from MinIO for ASR test case ID %d: %v. DB record was deleted.", tc.AudioFilePath, id, err)
			// Still return success for DB deletion, but log the MinIO issue.
			c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("ASR test case metadata deleted successfully, but failed to remove audio file '%s' from object storage: %v", tc.AudioFilePath, err)})
			return
		}
		log.Printf("Successfully deleted audio file '%s' from MinIO for ASR test case ID %d.", tc.AudioFilePath, id)
	}


	c.JSON(http.StatusOK, gin.H{"message": "ASR test case and associated audio file deleted successfully"})
}
