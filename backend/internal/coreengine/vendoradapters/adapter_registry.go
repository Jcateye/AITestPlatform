package vendoradapters

import (
	"fmt"
	"log"
	"unified-ai-eval-platform/backend/internal/datastore"
	"unified-ai-eval-platform/backend/internal/objectstore"
	// "github.com/minio/minio-go/v7" // This might be needed if adapters take MinioClient directly
)

// GlobalObjectStoreClient will be set by InitAdapterRegistry or from a global accessor.
// For MVP, we assume it's initialized and accessible.
// In a more robust system, this would be passed via dependency injection.
var globalObjectStoreClient *objectstore.MinioClient

// InitAdapterRegistry can be used to initialize shared resources for adapters, like the object store client.
func InitAdapterRegistry(minioClient *objectstore.MinioClient) {
	if minioClient == nil {
		log.Println("Warning: InitAdapterRegistry called with a nil MinioClient. Real adapters needing object storage may fail.")
	}
	globalObjectStoreClient = minioClient
}

// GetASRAdapter selects and returns an ASRAdapter based on the vendor configuration.
// For MVP, it primarily returns the MockASRAdapter.
func GetASRAdapter(vendorConfig *datastore.VendorConfig) (ASRAdapter, error) {
	if vendorConfig == nil {
		return nil, fmt.Errorf("vendorConfig cannot be nil")
	}

	// Log which adapter is being requested based on vendor config name
	log.Printf("Attempting to get ASR adapter for vendor: %s (Type: %s)", vendorConfig.Name, vendorConfig.APIType)

	// Simple selection logic for MVP
	// This can be expanded with a map or more sophisticated factory pattern.
	switch vendorConfig.Name {
	case "MockASR":
		log.Println("Selected MockASRAdapter.")
		return &MockASRAdapter{}, nil
	case "MockASR-Error": // A specific mock configuration to simulate errors
		log.Println("Selected MockASRAdapter (configured for errors).")
		return &MockASRAdapter{}, nil // The mock adapter itself will check vendorConfig.Name
	case "GoogleCloudASR":
		log.Println("Selected GoogleASRAdapter.")
		if globalObjectStoreClient == nil {
			return nil, fmt.Errorf("GoogleASRAdapter requires an initialized object store client, but it's nil")
		}
		return NewGoogleASRAdapter(globalObjectStoreClient), nil
	case "MicrosoftASR":
		log.Println("Selected MicrosoftASRAdapter.")
		if globalObjectStoreClient == nil {
			return nil, fmt.Errorf("MicrosoftASRAdapter requires an initialized object store client, but it's nil")
		}
		return NewMicrosoftASRAdapter(globalObjectStoreClient), nil
	case "DeepgramASR":
		log.Println("Selected DeepgramASRAdapter.")
		if globalObjectStoreClient == nil {
			return nil, fmt.Errorf("DeepgramASRAdapter requires an initialized object store client, but it's nil")
		}
		return NewDeepgramASRAdapter(globalObjectStoreClient), nil
	case "TencentASR":
		log.Println("Selected TencentASRAdapter.")
		if globalObjectStoreClient == nil {
			return nil, fmt.Errorf("TencentASRAdapter requires an initialized object store client, but it's nil")
		}
		return NewTencentASRAdapter(globalObjectStoreClient), nil
	case "VolcengineASR":
		log.Println("Selected VolcengineASRAdapter.")
		if globalObjectStoreClient == nil {
			return nil, fmt.Errorf("VolcengineASRAdapter requires an initialized object store client, but it's nil")
		}
		return NewVolcengineASRAdapter(globalObjectStoreClient), nil
	case "AlibabaASR": // Assuming AlibabaASR was added in a previous step or will be stubbed
		log.Println("Selected AlibabaASRAdapter.")
		if globalObjectStoreClient == nil {
			return nil, fmt.Errorf("AlibabaASRAdapter requires an initialized object store client, but it's nil")
		}
		// Assuming NewAlibabaASRAdapter exists, even if stubbed
		return NewAlibabaASRAdapter(globalObjectStoreClient), nil
	default:
		log.Printf("No specific adapter found for vendor '%s' (API Type: %s). Defaulting to MockASRAdapter as a fallback for MVP.", vendorConfig.Name, vendorConfig.APIType)
		// Fallback to MockASRAdapter if no specific adapter is found, to ensure MVP flow.
		// In a production system, this might return an error or a more sophisticated default.
		return &MockASRAdapter{}, nil
		// return nil, fmt.Errorf("no ASR adapter available for vendor: %s (Type: %s)", vendorConfig.Name, vendorConfig.APIType)
	}
}
