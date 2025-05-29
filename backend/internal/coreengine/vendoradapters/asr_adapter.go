package vendoradapters

import (
	"unified-ai-eval-platform/backend/internal/datastore"
	// "unified-ai-eval-platform/backend/internal/objectstore" // To be used by actual adapters
)

// ASRAdapter defines the interface for Automatic Speech Recognition vendor services.
type ASRAdapter interface {
	// Recognize transcribes audio from the given audioFilePath (which is an object key in object storage)
	// using the specified languageCode and vendor-specific parameters.
	// vendorConfig provides necessary API keys, endpoints, and other configurations.
	// It returns the recognized text and any error encountered.
	Recognize(audioFilePath string, languageCode string, params map[string]interface{}, vendorConfig *datastore.VendorConfig) (recognizedText string, rawResponse string, err error)
}

// Note: The `rawResponse` is added to the interface to allow storing the exact vendor output.
// For the mock adapter, this could be the same as recognizedText or a simple JSON string.
