package vendoradapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"time"
	"unified-ai-eval-platform/backend/internal/datastore"
	"unified-ai-eval-platform/backend/internal/objectstore"

	"github.com/Microsoft/cognitive-services-speech-sdk-go/audio"
	"github.com/Microsoft/cognitive-services-speech-sdk-go/speech"
)

// MicrosoftASRAdapter implements the ASRAdapter interface for Azure Cognitive Speech Services.
type MicrosoftASRAdapter struct {
	MinioClient *objectstore.MinioClient
}

// NewMicrosoftASRAdapter creates a new instance of MicrosoftASRAdapter.
func NewMicrosoftASRAdapter(minioClient *objectstore.MinioClient) *MicrosoftASRAdapter {
	if minioClient == nil {
		log.Println("Warning: NewMicrosoftASRAdapter created with a nil MinioClient. File fetching will fail.")
	}
	return &MicrosoftASRAdapter{MinioClient: minioClient}
}

// Recognize transcribes audio using Azure Cognitive Speech Services.
func (a *MicrosoftASRAdapter) Recognize(audioFilePath string, languageCode string, params map[string]interface{}, vendorConfig *datastore.VendorConfig) (recognizedText string, rawResponse string, err error) {
	ctx := context.Background()

	if a.MinioClient == nil {
		return "", "", fmt.Errorf("MicrosoftASRAdapter: MinioClient is not initialized")
	}

	if !vendorConfig.APIKey.Valid || vendorConfig.APIKey.String == "" {
		return "", "", fmt.Errorf("Azure Speech API key is missing in vendor configuration")
	}
	subscriptionKey := vendorConfig.APIKey.String

	var region string
	if vendorConfig.OtherConfigs != nil {
		var otherConfMap map[string]interface{}
		if err := json.Unmarshal(vendorConfig.OtherConfigs, &otherConfMap); err == nil {
			if r, ok := otherConfMap["azure_region"].(string); ok && r != "" {
				region = r
			}
		}
	}
	if region == "" {
		return "", "", fmt.Errorf("Azure Speech region is missing in vendor configuration (OtherConfigs.azure_region)")
	}

	log.Printf("MicrosoftASRAdapter: Recognize called for audio file '%s', language '%s', region '%s', vendor '%s'", audioFilePath, languageCode, region, vendorConfig.Name)

	// 1. Create SpeechConfig
	speechConfig, err := speech.NewSpeechConfigFromSubscription(subscriptionKey, region)
	if err != nil {
		return "", "", fmt.Errorf("failed to create Azure SpeechConfig: %w", err)
	}
	defer speechConfig.Close()

	speechConfig.SetSpeechRecognitionLanguage(languageCode)

	// Apply parameters from `params` or `vendorConfig.OtherConfigs.config`
	// Example: Profanity option
	var profanityOption speech.ProfanityOption = speech.ProfanityOption_Masked // Default
	configMap, _ := vendorConfig.OtherConfigs["config"].(map[string]interface{})
	if paramProfanity, ok := params["profanity_option"].(string); ok {
		profanityOption = parseProfanityOption(paramProfanity)
	} else if cfgProfanity, ok := configMap["profanity_option"].(string); ok {
		profanityOption = parseProfanityOption(cfgProfanity)
	}
	speechConfig.SetProfanity(profanityOption)
	log.Printf("MicrosoftASRAdapter: Set profanity option to %v", profanityOption)

	// 2. Audio Fetching and Configuration
	audioFile, fileSize, err := a.MinioClient.GetFileReader(ctx, audioFilePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch audio file '%s' from MinIO: %w", audioFilePath, err)
	}
	defer audioFile.Close()

	// Using PullAudioInputStream for potentially large files
	// Note: For some audio formats, Azure might require specific headers or format hints.
	// For simple WAV/MP3, auto-detection often works.
	// If using a specific format, you might need to create AudioStreamFormat explicitly.
	// audioFormat, err := audio.GetDefaultInputFormat() // Or specify format
	// if err != nil {
	// 	return "", "", fmt.Errorf("failed to get default audio format: %w", err)
	// }
	// defer audioFormat.Close()
	// callback := NewReadCallback(audioFile)
	// pullStream, err := audio.CreatePullAudioInputStreamFromFormat(callback, audioFormat)

	// Simpler approach for common formats: read into buffer and use PushStream or FromBytes
	// This might be less memory efficient for very large files but simpler for MVP.
	audioBytes, err := io.ReadAll(io.LimitReader(audioFile, 100*1024*1024)) // Limit read to 100MB for safety
	if err != nil {
		return "", "", fmt.Errorf("failed to read audio file content: %w", err)
	}
	_ = fileSize // fileSize can be used with PushStream if needed

	pushStream, err := audio.CreatePushAudioInputStream()
	if err != nil {
		return "", "", fmt.Errorf("failed to create push audio input stream: %w", err)
	}
	defer pushStream.Close()

	_, err = pushStream.Write(audioBytes)
	if err != nil {
		return "", "", fmt.Errorf("failed to write audio data to push stream: %w", err)
	}
	// Signal end of stream
	pushStream.CloseStream()


	audioConfig, err := audio.NewAudioConfigFromStreamInput(pushStream)
	if err != nil {
		return "", "", fmt.Errorf("failed to create Azure AudioConfig: %w", err)
	}
	defer audioConfig.Close()

	// 3. Create SpeechRecognizer
	recognizer, err := speech.NewSpeechRecognizerFromConfig(speechConfig, audioConfig)
	if err != nil {
		return "", "", fmt.Errorf("failed to create Azure SpeechRecognizer: %w", err)
	}
	defer recognizer.Close()

	// 4. Perform Recognition
	log.Printf("Sending recognition request to Azure Speech Service for %s", audioFilePath)
	startTime := time.Now()
	task := recognizer.RecognizeOnceAsync()
	var outcome speech.SpeechRecognitionResult

	select {
	case outcome = <-task:
		// Successfully received result or error
	case <-time.After(60 * time.Second): // Timeout for the recognition task
		return "", `{"error": "Recognition timed out after 60 seconds"}`, fmt.Errorf("Azure Speech API recognition timed out")
	}
	latency := time.Since(startTime)
	log.Printf("Azure Speech Service call for %s completed in %v", audioFilePath, latency)

	defer outcome.Close()

	// 5. Response Handling
	if outcome.Error != nil {
		rawResponse = fmt.Sprintf(`{"error": "Recognition error: %s", "reason": "%s"}`, outcome.Error.Error(), outcome.Reason.String())
		return "", rawResponse, fmt.Errorf("Azure Speech API recognition error: %w, reason: %s", outcome.Error, outcome.Reason.String())
	}

	if outcome.Reason == speech.ResultReason_RecognizedSpeech {
		recognizedText = outcome.Text
		// Construct a more detailed raw response if needed
		rawResponseDetails := map[string]interface{}{
			"text":       outcome.Text,
			"duration":   outcome.Duration.String(),
			"offset":     outcome.Offset.String(),
			"properties": map[string]string{},
		}
		for _, key := range outcome.Properties.PropertyIds() {
			rawResponseDetails["properties"].(map[string]string)[key.String()] = outcome.Properties.GetProperty(key, "")
		}
		rawResponseBytes, marshalErr := json.Marshal(rawResponseDetails)
		if marshalErr != nil {
			log.Printf("Error marshalling Azure Speech API response details to JSON: %v.", marshalErr)
			rawResponse = fmt.Sprintf(`{"text": "%s", "marshalling_error": "%s"}`, outcome.Text, marshalErr.Error())
		} else {
			rawResponse = string(rawResponseBytes)
		}
		log.Printf("MicrosoftASRAdapter: Successfully recognized text for '%s': %s", audioFilePath, recognizedText)
		return recognizedText, rawResponse, nil
	} else if outcome.Reason == speech.ResultReason_NoMatch {
		rawResponse = `{"error": "No speech could be recognized", "reason": "NoMatch"}`
		return "", rawResponse, fmt.Errorf("no speech could be recognized from audio: %s", audioFilePath)
	} else {
		rawResponse = fmt.Sprintf(`{"error": "Recognition failed", "reason": "%s"}`, outcome.Reason.String())
		return "", rawResponse, fmt.Errorf("Azure Speech API recognition failed with reason: %s", outcome.Reason.String())
	}
}

// Helper to parse profanity option string to SDK type
func parseProfanityOption(s string) speech.ProfanityOption {
	switch strings.ToLower(s) {
	case "raw":
		return speech.ProfanityOption_Raw
	case "removed":
		return speech.ProfanityOption_Removed
	case "masked":
		return speech.ProfanityOption_Masked
	default:
		log.Printf("Unknown profanity option '%s', defaulting to Masked.", s)
		return speech.ProfanityOption_Masked
	}
}

// ReadCallback for PullAudioInputStream (alternative audio input method)
type ReadCallback struct {
	Reader io.Reader
}

func NewReadCallback(reader io.Reader) *ReadCallback {
	return &ReadCallback{Reader: reader}
}

func (r *ReadCallback) Read(buffer []byte) (uint32, error) {
	n, err := r.Reader.Read(buffer)
	if err != nil && err != io.EOF {
		log.Printf("ReadCallback error: %v", err)
		return uint32(n), err
	}
	if err == io.EOF && n == 0 {
		return 0, io.EOF // Signal end of stream properly
	}
	return uint32(n), nil
}

func (r *ReadCallback) GetProperty(id speech.PropertyID) string {
	return "" // Not used for basic file streaming
}

func (r *ReadCallback) Close() error {
	if c, ok := r.Reader.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

// Helper to get audio format from file extension (very basic)
func getAudioFormat(filePath string) *audio.AudioStreamFormat {
	// This is a very simplified example. Production code should inspect file headers.
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".wav":
		// Assuming standard WAV format, e.g., PCM 16kHz 16-bit mono
		// For more robust solution, parse WAV header or use AudioStreamFormat.GetWaveFormatPCM
		format, _ := audio.GetWaveFormatPCM(16000, 16, 1)
		return format
	// Add cases for MP3, OGG, etc. if needed, though PushStream handles some auto-detection.
	default:
		format, _ := audio.GetDefaultInputFormat()
		return format
	}
}
