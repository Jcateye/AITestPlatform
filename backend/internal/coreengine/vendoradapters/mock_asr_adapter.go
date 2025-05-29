package vendoradapters

import (
	"fmt"
	"log"
	"time"
	"unified-ai-eval-platform/backend/internal/datastore"
)

// MockASRAdapter is a mock implementation of the ASRAdapter interface.
type MockASRAdapter struct{}

// Recognize simulates an ASR transcription.
func (m *MockASRAdapter) Recognize(audioFilePath string, languageCode string, params map[string]interface{}, vendorConfig *datastore.VendorConfig) (string, string, error) {
	log.Printf("MockASRAdapter: Recognize called for audio file '%s', language '%s', vendor '%s'", audioFilePath, languageCode, vendorConfig.Name)

	// Simulate network latency
	time.Sleep(500 * time.Millisecond) // 0.5 seconds delay

	// Predefined dummy result
	mockText := fmt.Sprintf("Mock recognition for %s: Hello world, this is a test for language %s.", audioFilePath, languageCode)
	if vendorConfig.Name == "MockASR-Error" {
		mockRawResponse := `{"error": "Simulated error from MockASR-Error vendor"}`
		return "", mockRawResponse, fmt.Errorf("simulated error from MockASR-Error for file %s", audioFilePath)
	}
	
	// For successful mock, raw response can be simple JSON
	mockRawResponse := fmt.Sprintf(`{"transcription": "%s", "confidence": 0.95, "simulated": true}`, mockText)


	log.Printf("MockASRAdapter: Successfully recognized text for '%s'", audioFilePath)
	return mockText, mockRawResponse, nil
}
