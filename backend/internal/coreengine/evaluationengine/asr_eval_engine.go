package evaluationengine

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"unified-ai-eval-platform/backend/internal/coreengine/metricscalculator"
	"unified-ai-eval-platform/backend/internal/coreengine/vendoradapters"
	"unified-ai-eval-platform/backend/internal/datastore"
)

// RunASREvaluation executes ASR evaluations for given test cases against specified vendors.
// This is a synchronous MVP implementation.
func RunASREvaluation(jobID int, testCaseIDs []int, vendorConfigIDs []int) error {
	log.Printf("Starting ASR Evaluation for Job ID: %d", jobID)
	log.Printf("Test Case IDs: %v, Vendor Config IDs: %v", testCaseIDs, vendorConfigIDs)

	if datastore.DB == nil {
		return fmt.Errorf("database connection is not initialized")
	}
	// Note: Minio client for adapters is handled by vendoradapters.InitAdapterRegistry,
	// which should be called at application startup if adapters need it.

	for _, testCaseID := range testCaseIDs {
		testCase, err := datastore.GetASRTestCase(testCaseID)
		if err != nil {
			log.Printf("Error fetching ASR Test Case ID %d: %v. Skipping this test case for job %d.", testCaseID, err, jobID)
			// In a more robust system, we might record this error against the job or test case instance.
			continue
		}
		log.Printf("Processing Test Case: %s (ID: %d)", testCase.Name, testCase.ID)

		for _, vendorConfigID := range vendorConfigIDs {
			vendorConfig, err := datastore.GetVendorConfig(vendorConfigID)
			if err != nil {
				log.Printf("Error fetching Vendor Config ID %d: %v. Skipping this vendor for test case %d, job %d.", vendorConfigID, err, testCaseID, jobID)
				continue
			}
			log.Printf("Using Vendor: %s (ID: %d) for Test Case %s (ID: %d)", vendorConfig.Name, vendorConfig.ID, testCase.Name, testCase.ID)

			adapter, err := vendoradapters.GetASRAdapter(vendorConfig)
			if err != nil {
				log.Printf("Error getting ASR adapter for vendor %s (ID: %d): %v. Skipping this vendor for test case %d, job %d.", vendorConfig.Name, vendorConfig.ID, err, testCaseID, jobID)
				continue
			}

			// Parameters for the Recognize method (can be extended in future)
			// For MVP, we don't have specific per-job or per-vendor-test-case parameters.
			// These could come from the `evaluation_jobs.parameters` field if designed so.
			recognitionParams := make(map[string]interface{})
			// Example: recognitionParams["model"] = "enhanced-model" if vendorConfig.SupportedModels or job params specify it

			startTime := time.Now()
			recognizedText, rawResponse, err := adapter.Recognize(testCase.AudioFilePath, testCase.LanguageCode.String, recognitionParams, vendorConfig)
			latencyMs := time.Since(startTime).Milliseconds()

			result := datastore.ASREvaluationResult{
				JobID:          jobID,
				ASRTestCaseID:  testCase.ID,
				VendorConfigID: vendorConfig.ID,
				LatencyMs:      sql.NullInt64{Int64: latencyMs, Valid: true},
			}
			
			if rawResponse != "" {
				result.RawVendorResponse = json.RawMessage(rawResponse)
			} else {
				result.RawVendorResponse = json.RawMessage("null")
			}


			if err != nil {
				log.Printf("Error during ASR recognition for Test Case ID %d, Vendor ID %d: %v", testCaseID, vendorConfigID, err)
				// Store error in recognized_text or a dedicated error field if schema supported it.
				// For now, recognized_text will be empty, metrics will be high or error.
				result.RecognizedText = sql.NullString{String: fmt.Sprintf("Recognition Error: %v", err), Valid: true}
				// Metrics might not be calculable or will be worst-case.
			} else {
				result.RecognizedText = sql.NullString{String: recognizedText, Valid: true}
			}

			// Calculate metrics if ground truth is available
			if testCase.GroundTruthText.Valid && testCase.GroundTruthText.String != "" {
				gt := testCase.GroundTruthText.String
				rec := recognizedText // Use `recognizedText` which is empty if error occurred before this point

				if err == nil { // Only calculate if recognition was successful
					cer, cerErr := metricscalculator.CalculateCER(gt, rec)
					if cerErr != nil {
						log.Printf("Error calculating CER for TC ID %d, Vendor ID %d: %v", testCaseID, vendorConfigID, cerErr)
						result.CER = sql.NullFloat64{Valid: false} // Or some error indicator if schema allows
					} else {
						result.CER = sql.NullFloat64{Float64: cer, Valid: true}
					}

					wer, werErr := metricscalculator.CalculateWER(gt, rec)
					if werErr != nil {
						log.Printf("Error calculating WER for TC ID %d, Vendor ID %d: %v", testCaseID, vendorConfigID, werErr)
						result.WER = sql.NullFloat64{Valid: false}
					} else {
						result.WER = sql.NullFloat64{Float64: wer, Valid: true}
					}
				}
				// SER is optional for MVP, not calculated here.
				result.SER = sql.NullFloat64{Valid: false}

			} else {
				log.Printf("No ground truth for Test Case ID %d. Metrics (CER, WER) will not be calculated.", testCaseID)
				result.CER = sql.NullFloat64{Valid: false}
				result.WER = sql.NullFloat64{Valid: false}
				result.SER = sql.NullFloat64{Valid: false}
			}

			_, dbErr := datastore.CreateASREvaluationResult(&result)
			if dbErr != nil {
				log.Printf("Error saving ASR evaluation result for TC ID %d, Vendor ID %d, Job ID %d: %v", testCaseID, vendorConfigID, jobID, dbErr)
				// This is a critical error; the result wasn't saved.
				// Consider how to handle this - retry? Mark job as partially failed?
				// For MVP, we just log and continue.
			} else {
				log.Printf("Successfully processed and saved result for TC ID %d, Vendor ID %d, Job ID %d.", testCaseID, vendorConfigID, jobID)
			}
		} // End loop vendorConfigIDs
	} // End loop testCaseIDs

	log.Printf("Completed ASR Evaluation for Job ID: %d", jobID)
	return nil
}
