package vendoradapters

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"unified-ai-eval-platform/backend/internal/datastore"
	"unified-ai-eval-platform/backend/internal/objectstore"

	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	"github.com/volcengine/volcengine-go-sdk/service/speech_ai" // Correct package for ASR
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"github.com/volcengine/volcengine-go-sdk/volcengine/credentials"
	"github.com/volcengine/volcengine-go-sdk/volcengine/session"
)

// VolcengineASRAdapter implements the ASRAdapter interface for Volcengine Speech Recognition.
type VolcengineASRAdapter struct {
	MinioClient *objectstore.MinioClient
}

// NewVolcengineASRAdapter creates a new instance of VolcengineASRAdapter.
func NewVolcengineASRAdapter(minioClient *objectstore.MinioClient) *VolcengineASRAdapter {
	if minioClient == nil {
		log.Println("Warning: NewVolcengineASRAdapter created with a nil MinioClient. File fetching will fail.")
	}
	return &VolcengineASRAdapter{MinioClient: minioClient}
}

// Volcengine specific request/response structures might be needed if using direct HTTP
// For SDK usage, the SDK's own types are used.

// Recognize transcribes audio using Volcengine Cloud Speech Recognition API.
func (a *VolcengineASRAdapter) Recognize(audioFilePath string, languageCode string, params map[string]interface{}, vendorConfig *datastore.VendorConfig) (recognizedText string, rawResponse string, err error) {
	ctx := context.Background()

	if a.MinioClient == nil {
		return "", "", fmt.Errorf("VolcengineASRAdapter: MinioClient is not initialized")
	}

	// 1. Authentication and Configuration
	accessKeyId := vendorConfig.APIKey.String
	secretKey := vendorConfig.APISecret.String

	if accessKeyId == "" {
		return "", "", fmt.Errorf("Volcengine AccessKeyID (APIKey) is missing")
	}
	if secretKey == "" {
		return "", "", fmt.Errorf("Volcengine SecretAccessKey (APISecret) is missing")
	}

	var region string
	var appId string // AppId is typically a string for Volcengine
	var cluster string // Some APIs might require a cluster identifier

	// Default values
	audioFormat := "wav" // Default format
	sampleRate := int64(16000) // Default sample rate

	if vendorConfig.OtherConfigs != nil {
		var otherConfMap map[string]interface{}
		if err := json.Unmarshal(vendorConfig.OtherConfigs, &otherConfMap); err == nil {
			if r, ok := otherConfMap["volcengine_region"].(string); ok && r != "" {
				region = r
			}
			if id, ok := otherConfMap["volcengine_app_id"].(string); ok && id != "" {
				appId = id
			}
			if c, ok := otherConfMap["volcengine_cluster"].(string); ok && c != "" {
				cluster = c
			}
			if cfg, cfgOk := otherConfMap["config"].(map[string]interface{}); cfgOk {
				if f, ok := cfg["format"].(string); ok && f != "" {
					audioFormat = f
				}
				if sr, ok := cfg["sample_rate"].(float64); ok { // JSON numbers often float64
					sampleRate = int64(sr)
				}
				// Language is usually part of the engine type or a direct parameter
				if lang, ok := cfg["language"].(string); ok && lang != "" {
					languageCode = lang // Override if specified in config
				}
			}
		}
	}

	if region == "" {
		return "", "", fmt.Errorf("Volcengine region (volcengine_region) is missing in OtherConfigs")
	}
	if appId == "" {
		return "", "", fmt.Errorf("Volcengine AppID (volcengine_app_id) is missing in OtherConfigs")
	}
	// Cluster might be optional depending on the specific API version/endpoint

	log.Printf("VolcengineASRAdapter: Recognize called. File: %s, Lang: %s, Region: %s, AppID: %s, Format: %s, SampleRate: %d",
		audioFilePath, languageCode, region, appId, audioFormat, sampleRate)

	// Initialize Volcengine session and ASR client
	cfg := volcengine.NewConfig()
	cfg.Credentials = credentials.NewStaticCredentials(accessKeyId, secretKey, "")
	cfg.Region = region
	// cfg.WithScheme("https") // Default is https, can be http for local testing if needed

	sess, err := session.NewSession(cfg)
	if err != nil {
		return "", "", fmt.Errorf("failed to create Volcengine session: %w", err)
	}

	asrClient := speech_ai.New(sess)
	asrClient.Client.SetTimeout(60 * time.Second) // Set a timeout for the API call

	// 2. Audio Fetching and Encoding
	audioBytes, err := a.MinioClient.GetFileBytes(ctx, audioFilePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch audio file '%s' from MinIO: %w", audioFilePath, err)
	}
	base64Audio := base64.StdEncoding.EncodeToString(audioBytes)

	// 3. Construct Request for Speech Recognition
	// Volcengine typically uses a "workflow" approach or a direct "recognize" API.
	// For short audio, "RecognizeSpeech" or similar might be suitable.
	// The `speech_ai.SubmitAudioTaskRequest` is for asynchronous tasks.
	// Let's look for a synchronous or short audio API endpoint if available.
	// The `speech_ai.SpeechRecognitionRequest` looks like a good candidate for synchronous recognition.

	req := &speech_ai.SpeechRecognizeRequest{
		App: &speech_ai.App{
			AppId:      volcengine.String(appId),
			Cluster:    volcengine.String(cluster), // Cluster might be optional or specific to certain services/regions
			Token:      volcengine.String(""),      // Token is usually for client-side SDKs, not typically needed for server-to-server with AK/SK
			WorkflowId: volcengine.String(""),      // Not using a workflow for direct recognition
		},
		Audio: &speech_ai.Audio{
			Format: volcengine.String(strings.ToLower(filepath.Ext(audioFilePath))[1:]), // e.g., "wav", "pcm", "mp3"
			// SampleRate: volcengine.Int(int(sampleRate)), // SampleRate seems to be part of format or engine_type
			Data: volcengine.String(base64Audio),
		},
		Config: &speech_ai.RecognitionConfig{
			Language: volcengine.String(languageCode), // e.g., "zh-CN", "en-US"
			// EngineType: volcengine.String("16k_auto"), // Example, make configurable
			// AddPunc: volcengine.Bool(true),
			// ResultType: volcengine.String("text"),
		},
	}

	// Apply parameters from `params` or `vendorConfig.OtherConfigs.config`
	if configMap, ok := vendorConfig.OtherConfigs["config"].(map[string]interface{}); ok {
		if engineType, ok := configMap["engine_type"].(string); ok {
			req.Config.EngineType = volcengine.String(engineType)
		}
		if addPunc, ok := configMap["add_punc"].(bool); ok {
			req.Config.AddPunc = volcengine.Bool(addPunc)
		}
		// Add more parameter mappings here
	}
	if jobParamsEngineType, ok := params["engine_type"].(string); ok && jobParamsEngineType != "" {
		req.Config.EngineType = volcengine.String(jobParamsEngineType)
	}
	if jobParamsAddPunc, ok := params["add_punc"].(bool); ok {
		req.Config.AddPunc = volcengine.Bool(jobParamsAddPunc)
	}
	if req.Config.EngineType == nil || *req.Config.EngineType == "" {
		// Default based on language or a general default
		if strings.HasPrefix(languageCode, "zh") {
			req.Config.EngineType = volcengine.String("16k_zh")
		} else if strings.HasPrefix(languageCode, "en") {
			req.Config.EngineType = volcengine.String("16k_en")
		} else {
			req.Config.EngineType = volcengine.String("16k_auto") // A generic default
		}
		log.Printf("VolcengineASRAdapter: Using default/derived EngineType: %s", *req.Config.EngineType)
	}
	if req.Audio.Format == nil || *req.Audio.Format == "" {
		req.Audio.Format = volcengine.String("wav") // Default if not determined by extension
	}


	// 4. API Call
	log.Printf("Sending recognition request to Volcengine ASR API for %s. AppID: %s, EngineType: %s, Format: %s",
		audioFilePath, *req.App.AppId, *req.Config.EngineType, *req.Audio.Format)
	
	startTime := time.Now()
	resp, err := asrClient.SpeechRecognize(req)
	latency := time.Since(startTime)
	log.Printf("Volcengine ASR API call for %s completed in %v", audioFilePath, latency)

	// 5. Response Handling
	rawResponseBytes, _ := json.Marshal(resp)
	rawResponse = string(rawResponseBytes)

	if err != nil {
		// The Volcengine SDK might return errors in a specific format.
		// Example: check for `volcengine.SdkError`
		if sdkErr, ok := err.(volcengine.SdkError); ok {
			log.Printf("Volcengine ASR API Error: Code=%s, Message=%s, RequestId=%s", sdkErr.Code(), sdkErr.Message(), sdkErr.RequestId())
			return "", rawResponse, fmt.Errorf("Volcengine ASR API error: %s (Code: %s)", sdkErr.Message(), sdkErr.Code())
		}
		log.Printf("Volcengine ASR API Error (non-SDK): %v. Raw Response: %s", err, rawResponse)
		return "", rawResponse, fmt.Errorf("Volcengine ASR API request failed: %w", err)
	}

	if resp == nil || resp.Result == nil || resp.Result.Result == nil || len(resp.Result.Result) == 0 {
		log.Printf("Volcengine ASR API Error: Response or Result is nil/empty. RawResponse: %s", rawResponse)
		return "", rawResponse, fmt.Errorf("Volcengine ASR API returned empty or invalid result. Raw: %s", rawResponse)
	}

	// Assuming the first result is the most relevant one.
	// The structure of `resp.Result.Result` might be a list of segments or alternatives.
	// For this MVP, we concatenate them if it's a list of strings.
	// Example: resp.Result.Result might be a string or a struct containing the transcript.
	// Based on `speech_ai.SpeechRecognizeResult`, `resp.Result.Result` is `*string`.
	if resp.Result.Result != nil {
		recognizedText = *resp.Result.Result
	} else {
		recognizedText = "" // No text recognized or field is nil
	}
	
	// The `ResultDetail` field might contain more structured information if available
	// For example, if `ResultType` was set to "all", ResultDetail would be populated.
	// For now, we are using the top-level Result string.

	log.Printf("VolcengineASRAdapter: Successfully recognized text for '%s': %s", audioFilePath, recognizedText)
	return recognizedText, rawResponse, nil
}
