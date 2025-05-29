package vendoradapters

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	speech "cloud.google.com/go/speech/apiv1"
	"cloud.google.com/go/speech/apiv1/speechpb"
	"google.golang.org/api/option"

	"unified-ai-eval-platform/backend/internal/datastore"
	"unified-ai-eval-platform/backend/internal/objectstore"
)

// GoogleASRAdapter implements the ASRAdapter interface for Google Cloud Speech-to-Text.
type GoogleASRAdapter struct {
	MinioClient *objectstore.MinioClient // Minio client to fetch audio files
}

// NewGoogleASRAdapter creates a new instance of GoogleASRAdapter.
// It requires a MinioClient to fetch audio data from object storage.
func NewGoogleASRAdapter(minioClient *objectstore.MinioClient) *GoogleASRAdapter {
	if minioClient == nil {
		log.Println("Warning: NewGoogleASRAdapter created with a nil MinioClient. File fetching will fail.")
	}
	return &GoogleASRAdapter{MinioClient: minioClient}
}

// Recognize transcribes audio using Google Cloud Speech-to-Text.
func (a *GoogleASRAdapter) Recognize(audioFilePath string, languageCode string, params map[string]interface{}, vendorConfig *datastore.VendorConfig) (recognizedText string, rawResponse string, err error) {
	ctx := context.Background()

	if a.MinioClient == nil {
		return "", "", fmt.Errorf("GoogleASRAdapter: MinioClient is not initialized")
	}

	// 1. Authentication
	var opts []option.ClientOption
	credsPath, pathOk := vendorConfig.OtherConfigs["google_credentials_path"].(string)
	if pathOk && credsPath != "" {
		log.Printf("Using Google credentials from path specified in VendorConfig: %s", credsPath)
		opts = append(opts, option.WithCredentialsFile(credsPath))
	} else {
		// Try to use GOOGLE_APPLICATION_CREDENTIALS environment variable (implicitly handled by the library if set)
		log.Println("Attempting to use GOOGLE_APPLICATION_CREDENTIALS for authentication.")
	}

	speechClient, err := speech.NewClient(ctx, opts...)
	if err != nil {
		return "", "", fmt.Errorf("failed to create Google Speech client: %w", err)
	}
	defer speechClient.Close()

	// 2. Audio Fetching
	audioContent, err := a.MinioClient.GetFileBytes(ctx, audioFilePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch audio file '%s' from MinIO: %w", audioFilePath, err)
	}

	// 3. Construct RecognitionConfig
	// Default encoding and sample rate. These could be overridden by params or derived from file metadata.
	encoding := speechpb.RecognitionConfig_LINEAR16 // Default, common for WAV
	sampleRateHertz := int32(16000)                 // Default, common for WAV

	// Allow overrides from params
	if model, ok := params["model"].(string); ok && model != "" {
		// Google specific model can be set if provided
		// For simplicity, this example doesn't directly map to a specific field in RecognitionConfig
		// but you might use it to choose between standard/enhanced models or set specific model identifiers.
		log.Printf("Using model parameter (if applicable by Google): %s", model)
	}
	if useEnhanced, ok := params["useEnhanced"].(bool); ok {
		// Example: config.UseEnhanced = useEnhanced (if such a field exists or is mapped)
		log.Printf("Using useEnhanced parameter: %v", useEnhanced)
		// This might map to a specific model or a flag in RecognitionConfig.
		// For example, some models are inherently "enhanced".
	}
	if enc, ok := params["encoding"].(string); ok {
		// Map string encoding to speechpb.RecognitionConfig_AudioEncoding
		// This is a simplified example; a more robust mapping would be needed.
		if strings.ToUpper(enc) == "FLAC" {
			encoding = speechpb.RecognitionConfig_FLAC
		} else if strings.ToUpper(enc) == "MP3" {
			encoding = speechpb.RecognitionConfig_MP3
		}
		// Add more mappings as needed
	}
	if rate, ok := params["sampleRateHertz"].(float64); ok { // JSON numbers often parse as float64
		sampleRateHertz = int32(rate)
	}


	config := &speechpb.RecognitionConfig{
		Encoding:                   encoding,
		SampleRateHertz:            sampleRateHertz,
		LanguageCode:               languageCode,
		EnableAutomaticPunctuation: true, // Example: enable by default
		// Add more configuration options as needed, potentially from params or vendorConfig.OtherConfigs
	}

	// Apply model from vendorConfig.OtherConfigs if present
	if otherCfgMap, ok := vendorConfig.OtherConfigs["config"].(map[string]interface{}); ok {
		if model, ok := otherCfgMap["model"].(string); ok && model != "" {
			config.Model = model
			log.Printf("Using model '%s' from vendorConfig.OtherConfigs", model)
		}
		if useEnhanced, ok := otherCfgMap["useEnhanced"].(bool); ok {
			config.UseEnhanced = useEnhanced
			log.Printf("Using useEnhanced=%v from vendorConfig.OtherConfigs", useEnhanced)
		}
	}


	audio := &speechpb.RecognitionAudio{
		AudioSource: &speechpb.RecognitionAudio_Content{Content: audioContent},
	}

	req := &speechpb.RecognizeRequest{
		Config: config,
		Audio:  audio,
	}

	// 4. API Call
	log.Printf("Sending recognition request to Google Speech-to-Text API for %s", audioFilePath)
	startTime := time.Now()
	resp, err := speechClient.Recognize(ctx, req)
	latency := time.Since(startTime)
	log.Printf("Google Speech-to-Text API call for %s completed in %v", audioFilePath, latency)


	if err != nil {
		// Attempt to marshal the error to JSON if it's a Google API error
		// This is a simplification; gRPC errors have more structure.
		rawResponse = fmt.Sprintf(`{"error": "%s"}`, err.Error())
		return "", rawResponse, fmt.Errorf("Google Speech API recognition failed: %w", err)
	}

	// 5. Response Handling
	var transcriptBuilder strings.Builder
	for _, result := range resp.Results {
		if len(result.Alternatives) > 0 {
			transcriptBuilder.WriteString(result.Alternatives[0].Transcript)
			transcriptBuilder.WriteString(" ") // Add space between results if multiple
		}
	}
	recognizedText = strings.TrimSpace(transcriptBuilder.String())

	// Marshal the full response to JSON for rawResponse
	// Using protojson for better handling of protobuf messages
	// Note: This requires "google.golang.org/protobuf/encoding/protojson"
	// and "google.golang.org/protobuf/proto".
	// If these are not available, a simpler JSON marshal might be attempted,
	// but it might not be as accurate for protobuf structures.
	// For MVP, a simple json.Marshal might suffice if the structure is simple enough
	// or if we only care about a subset. However, proper protojson is better.
	// Let's assume for now we try a standard json.Marshal and log if it fails.
	rawResponseBytes, marshalErr := json.Marshal(resp)
	if marshalErr != nil {
		log.Printf("Error marshalling Google Speech API response to JSON: %v. Storing error message as rawResponse.", marshalErr)
		rawResponse = fmt.Sprintf(`{"error_marshalling_response": "%s"}`, marshalErr.Error())
	} else {
		rawResponse = string(rawResponseBytes)
	}
	
	log.Printf("MockASRAdapter: Successfully recognized text for '%s': %s", audioFilePath, recognizedText)
	return recognizedText, rawResponse, nil
}
