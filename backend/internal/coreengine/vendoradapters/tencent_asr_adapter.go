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

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	asr "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/asr/v20190614"
)

// TencentASRAdapter implements the ASRAdapter interface for Tencent Cloud Speech Recognition.
type TencentASRAdapter struct {
	MinioClient *objectstore.MinioClient
}

// NewTencentASRAdapter creates a new instance of TencentASRAdapter.
func NewTencentASRAdapter(minioClient *objectstore.MinioClient) *TencentASRAdapter {
	if minioClient == nil {
		log.Println("Warning: NewTencentASRAdapter created with a nil MinioClient. File fetching will fail.")
	}
	return &TencentASRAdapter{MinioClient: minioClient}
}

// Recognize transcribes audio using Tencent Cloud Speech Recognition API.
func (a *TencentASRAdapter) Recognize(audioFilePath string, languageCode string, params map[string]interface{}, vendorConfig *datastore.VendorConfig) (recognizedText string, rawResponse string, err error) {
	ctx := context.Background()

	if a.MinioClient == nil {
		return "", "", fmt.Errorf("TencentASRAdapter: MinioClient is not initialized")
	}

	// 1. Authentication and Configuration
	if !vendorConfig.APIKey.Valid || vendorConfig.APIKey.String == "" {
		return "", "", fmt.Errorf("Tencent Cloud SecretId (APIKey) is missing in vendor configuration")
	}
	secretId := vendorConfig.APIKey.String

	if !vendorConfig.APISecret.Valid || vendorConfig.APISecret.String == "" {
		return "", "", fmt.Errorf("Tencent Cloud SecretKey (APISecret) is missing in vendor configuration")
	}
	secretKey := vendorConfig.APISecret.String

	var region string
	var appID uint64 // AppId is often numeric for Tencent Cloud services
	var engineModelType string = "16k_zh" // Default engine model type

	if vendorConfig.OtherConfigs != nil {
		var otherConfMap map[string]interface{}
		if err := json.Unmarshal(vendorConfig.OtherConfigs, &otherConfMap); err == nil {
			if r, ok := otherConfMap["tencent_region"].(string); ok && r != "" {
				region = r
			}
			if id, ok := otherConfMap["tencent_app_id"].(float64); ok { // JSON numbers are float64
				appID = uint64(id)
			}
			if cfg, cfgOk := otherConfMap["config"].(map[string]interface{}); cfgOk {
				if emt, ok := cfg["engine_model_type"].(string); ok && emt != "" {
					engineModelType = emt
				}
			}
		}
	}

	if region == "" {
		return "", "", fmt.Errorf("Tencent Cloud region is missing in vendor configuration (OtherConfigs.tencent_region)")
	}

	log.Printf("TencentASRAdapter: Recognize called for audio file '%s', language '%s', region '%s', AppID %d, EngineModelType '%s', vendor '%s'",
		audioFilePath, languageCode, region, appID, engineModelType, vendorConfig.Name)

	credential := common.NewCredential(secretId, secretKey)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "asr.tencentcloudapi.com" // Default ASR endpoint
	// Potentially override endpoint from vendorConfig.APIEndpoint if provided
	if vendorConfig.APIEndpoint.Valid && vendorConfig.APIEndpoint.String != "" {
		cpf.HttpProfile.Endpoint = vendorConfig.APIEndpoint.String
		log.Printf("Using custom API endpoint: %s", cpf.HttpProfile.Endpoint)
	}

	client, err := asr.NewClient(credential, region, cpf)
	if err != nil {
		return "", "", fmt.Errorf("failed to create Tencent ASR client: %w", err)
	}

	// 2. Audio Fetching and Encoding
	audioBytes, err := a.MinioClient.GetFileBytes(ctx, audioFilePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch audio file '%s' from MinIO: %w", audioFilePath, err)
	}

	base64Audio := base64.StdEncoding.EncodeToString(audioBytes)

	// 3. Construct Request for SentenceRecognition API
	request := asr.NewSentenceRecognitionRequest()
	if appID != 0 { // AppId is often optional or part of older APIs, for newer SDKs it might be part of ProjectId or implicit.
		request.ProjectId = common.Uint64Ptr(appID) // ProjectId for some Tencent services maps to AppId
	}
	request.SubServiceType = common.Uint64Ptr(2) // 2 for far-field, common default
	request.EngSerViceType = common.StringPtr(engineModelType) // Example: "16k_zh", "16k_en"
	request.SourceType = common.Uint64Ptr(1) // 1 for audio data passed directly
	request.Data = common.StringPtr(base64Audio)
	request.DataLen = common.Int64Ptr(int64(len(audioBytes)))

	voiceFormat := strings.TrimPrefix(filepath.Ext(audioFilePath), ".")
	if voiceFormat == "" {
		voiceFormat = "wav" // Default if extension is missing
	}
	request.VoiceFormat = common.StringPtr(voiceFormat)

	// Apply parameters from `params` (job-specific)
	// These could override or add to the `EngSerViceType` or other request fields if needed.
	// Example: if params["engine_model_type"] exists, use it.
	if jobEngineModel, ok := params["engine_model_type"].(string); ok && jobEngineModel != "" {
		request.EngSerViceType = common.StringPtr(jobEngineModel)
		log.Printf("Overriding engine_model_type with job param: %s", jobEngineModel)
	}
	if jobLangCode, ok := params["language_code"].(string); ok && jobLangCode != "" {
		// Tencent's EngSerViceType often includes language and sample rate.
		// This is a simplified mapping. A more robust solution would be needed.
		// For example, "en-US" -> "16k_en", "zh-CN" -> "16k_zh"
		// This example assumes languageCode passed to Recognize() is already in Tencent's format if not overridden by params.
		if request.EngSerViceType == nil || *request.EngSerViceType == "" { // Only if not already set by vendor config
			if strings.HasPrefix(languageCode, "en") {
				request.EngSerViceType = common.StringPtr("16k_en")
			} else if strings.HasPrefix(languageCode, "zh") {
				request.EngSerViceType = common.StringPtr("16k_zh")
			}
			log.Printf("Setting EngSerViceType based on languageCode: %s", *request.EngSerViceType)
		}
	}
	if request.EngSerViceType == nil || *request.EngSerViceType == "" {
		log.Printf("Warning: EngSerViceType is not set. Defaulting to '16k_zh'. Language code provided was: '%s'", languageCode)
		request.EngSerViceType = common.StringPtr("16k_zh") // Fallback if not derived
	}


	// 4. API Call
	log.Printf("Sending SentenceRecognition request to Tencent ASR API for %s. EngSerViceType: %s, VoiceFormat: %s",
		audioFilePath, *request.EngSerViceType, *request.VoiceFormat)
	startTime := time.Now()
	response, err := client.SentenceRecognition(request)
	latency := time.Since(startTime)
	log.Printf("Tencent ASR API call for %s completed in %v", audioFilePath, latency)

	// 5. Response Handling
	// The raw response is the JSON string representation of the response object
	rawResponseBytes, _ := json.Marshal(response) // Ignoring marshal error for raw response for now
	rawResponse = string(rawResponseBytes)

	if err != nil {
		// Check if it's a TencentCloudSDKError
		if terr, ok := err.(*errors.TencentCloudSDKError); ok {
			log.Printf("Tencent ASR API Error: Code=%s, Message=%s, RequestId=%s", terr.GetCode(), terr.GetMessage(), terr.GetRequestId())
			return "", rawResponse, fmt.Errorf("Tencent ASR API error: %s (Code: %s)", terr.GetMessage(), terr.GetCode())
		}
		log.Printf("Tencent ASR API Error (non-SDK): %v", err)
		return "", rawResponse, fmt.Errorf("Tencent ASR API request failed: %w", err)
	}

	if response.Response == nil || response.Response.Result == nil {
		log.Printf("Tencent ASR API Error: Response or Result is nil. RawResponse: %s", rawResponse)
		return "", rawResponse, fmt.Errorf("Tencent ASR API returned nil response or result. Raw: %s", rawResponse)
	}
	
	recognizedText = *response.Response.Result
	log.Printf("TencentASRAdapter: Successfully recognized text for '%s': %s", audioFilePath, recognizedText)

	return recognizedText, rawResponse, nil
}
