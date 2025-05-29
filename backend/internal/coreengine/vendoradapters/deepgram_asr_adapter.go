package vendoradapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"unified-ai-eval-platform/backend/internal/datastore"
	"unified-ai-eval-platform/backend/internal/objectstore"
)

const deepgramBaseURL = "https://api.deepgram.com/v1/listen"

// DeepgramASRAdapter implements the ASRAdapter interface for Deepgram.
type DeepgramASRAdapter struct {
	MinioClient *objectstore.MinioClient
	HTTPClient  *http.Client
}

// NewDeepgramASRAdapter creates a new instance of DeepgramASRAdapter.
func NewDeepgramASRAdapter(minioClient *objectstore.MinioClient) *DeepgramASRAdapter {
	if minioClient == nil {
		log.Println("Warning: NewDeepgramASRAdapter created with a nil MinioClient. File fetching will fail.")
	}
	return &DeepgramASRAdapter{
		MinioClient: minioClient,
		HTTPClient:  &http.Client{Timeout: time.Second * 60}, // Increased timeout for potentially larger files/network latency
	}
}

// DeepgramResponse represents the structure of the JSON response from Deepgram.
// This is a simplified version; the actual response can be more complex.
type DeepgramResponse struct {
	RequestID string `json:"request_id"`
	Metadata  struct {
		TransactionKey string    `json:"transaction_key"`
		RequestID      string    `json:"request_id"`
		SHA256         string    `json:"sha256"`
		CreatedAt      time.Time `json:"created_at"`
		Duration       float64   `json:"duration"`
		Channels       int       `json:"channels"`
		Models         []string  `json:"models"`
		ModelInfo      map[string]struct {
			Name    string `json:"name"`
			Version string `json:"version"`
			Arch    string `json:"arch"`
		} `json:"model_info"`
	} `json:"metadata"`
	Results struct {
		Channels []struct {
			Alternatives []struct {
				Transcript string  `json:"transcript"`
				Confidence float64 `json:"confidence"`
				Words      []struct {
					Word      string  `json:"word"`
					Start     float64 `json:"start"`
					End       float64 `json:"end"`
					Confidence float64 `json:"confidence"`
					PunctuatedWord string `json:"punctuated_word"`
				} `json:"words"`
			} `json:"alternatives"`
		} `json:"channels"`
	} `json:"results"`
}

// Recognize transcribes audio using the Deepgram API.
func (a *DeepgramASRAdapter) Recognize(audioFilePath string, languageCode string, params map[string]interface{}, vendorConfig *datastore.VendorConfig) (recognizedText string, rawResponse string, err error) {
	ctx := context.Background()

	if a.MinioClient == nil {
		return "", "", fmt.Errorf("DeepgramASRAdapter: MinioClient is not initialized")
	}
	if a.HTTPClient == nil {
		return "", "", fmt.Errorf("DeepgramASRAdapter: HTTPClient is not initialized")
	}

	if !vendorConfig.APIKey.Valid || vendorConfig.APIKey.String == "" {
		return "", "", fmt.Errorf("Deepgram API key is missing in vendor configuration")
	}
	apiKey := vendorConfig.APIKey.String

	log.Printf("DeepgramASRAdapter: Recognize called for audio file '%s', language '%s', vendor '%s'", audioFilePath, languageCode, vendorConfig.Name)

	// 1. Fetch audio content from MinIO
	audioBytes, err := a.MinioClient.GetFileBytes(ctx, audioFilePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch audio file '%s' from MinIO: %w", audioFilePath, err)
	}

	// 2. Determine Content-Type (MIME type)
	// Simple inference from file extension. A more robust solution might involve magic bytes.
	contentType := mime.TypeByExtension(filepath.Ext(audioFilePath))
	if contentType == "" {
		contentType = "application/octet-stream" // Default if type cannot be determined
		log.Printf("Warning: Could not determine Content-Type for %s, using default %s", audioFilePath, contentType)
	}

	// 3. Construct URL with query parameters
	reqURL, err := url.Parse(deepgramBaseURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse Deepgram base URL: %w", err)
	}
	query := reqURL.Query()
	if languageCode != "" {
		query.Set("language", languageCode)
	}

	// Apply parameters from vendorConfig.OtherConfigs.config first
	if vendorConfig.OtherConfigs != nil && len(vendorConfig.OtherConfigs) > 0 {
		var otherConfMap map[string]interface{}
		if err := json.Unmarshal(vendorConfig.OtherConfigs, &otherConfMap); err == nil {
			if cfg, ok := otherConfMap["config"].(map[string]interface{}); ok {
				for k, v := range cfg {
					query.Set(k, fmt.Sprintf("%v", v))
				}
			}
		}
	}
	// Apply parameters from job-specific params, potentially overriding vendorConfig defaults
	for key, value := range params {
		query.Set(key, fmt.Sprintf("%v", value))
	}
	reqURL.RawQuery = query.Encode()

	// 4. Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL.String(), bytes.NewReader(audioBytes))
	if err != nil {
		return "", "", fmt.Errorf("failed to create Deepgram request: %w", err)
	}

	req.Header.Set("Authorization", "Token "+apiKey)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")

	// 5. Execute request
	log.Printf("Sending recognition request to Deepgram API: %s", reqURL.String())
	startTime := time.Now()
	httpResp, err := a.HTTPClient.Do(req)
	latency := time.Since(startTime)
	log.Printf("Deepgram API call for %s completed in %v", audioFilePath, latency)

	if err != nil {
		return "", "", fmt.Errorf("failed to send request to Deepgram: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read Deepgram response body: %w", err)
	}
	rawResponse = string(respBody)

	if httpResp.StatusCode != http.StatusOK {
		log.Printf("Deepgram API Error: Status %s, Body: %s", httpResp.Status, rawResponse)
		return "", rawResponse, fmt.Errorf("Deepgram API request failed with status %s: %s", httpResp.Status, rawResponse)
	}

	// 6. Parse response
	var dgResponse DeepgramResponse
	if err := json.Unmarshal(respBody, &dgResponse); err != nil {
		return "", rawResponse, fmt.Errorf("failed to parse Deepgram JSON response: %w. Response: %s", err, rawResponse)
	}

	if len(dgResponse.Results.Channels) > 0 && len(dgResponse.Results.Channels[0].Alternatives) > 0 {
		recognizedText = dgResponse.Results.Channels[0].Alternatives[0].Transcript
	} else {
		log.Printf("Deepgram response did not contain expected transcript structure for %s. Raw response: %s", audioFilePath, rawResponse)
		// It might be a valid response but without transcription (e.g. empty audio)
		// For now, we don't treat this as an error but return empty recognizedText.
		// Depending on requirements, this could be an error.
	}

	log.Printf("DeepgramASRAdapter: Successfully recognized text for '%s': %s", audioFilePath, recognizedText)
	return recognizedText, rawResponse, nil
}
