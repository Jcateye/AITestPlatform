package objectstore

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinioClient holds the MinIO client and bucket name.
type MinioClient struct {
	Client     *minio.Client
	BucketName string
}

var globalMinioClient *MinioClient

// InitMinioClient initializes the global MinIO client from environment variables.
// This should be called at application startup.
func InitMinioClient() error {
	endpoint := os.Getenv("MINIO_ENDPOINT")
	accessKeyID := os.Getenv("MINIO_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("MINIO_SECRET_ACCESS_KEY")
	bucketName := os.Getenv("MINIO_BUCKET_NAME")
	useSSLStr := os.Getenv("MINIO_USE_SSL")

	if endpoint == "" || accessKeyID == "" || secretAccessKey == "" || bucketName == "" {
		return fmt.Errorf("MINIO_ENDPOINT, MINIO_ACCESS_KEY_ID, MINIO_SECRET_ACCESS_KEY, and MINIO_BUCKET_NAME must be set")
	}

	useSSL, err := strconv.ParseBool(useSSLStr)
	if err != nil {
		log.Printf("Warning: MINIO_USE_SSL environment variable is not a valid boolean ('%s'). Defaulting to false. Error: %v", useSSLStr, err)
		useSSL = false // Default to false if not set or invalid
	}

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize MinIO client: %w", err)
	}

	// Check if bucket exists, create if not
	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("failed to check if MinIO bucket '%s' exists: %w", bucketName, err)
	}
	if !exists {
		log.Printf("MinIO bucket '%s' does not exist. Attempting to create it.", bucketName)
		err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{}) // Use default region
		if err != nil {
			return fmt.Errorf("failed to create MinIO bucket '%s': %w", bucketName, err)
		}
		log.Printf("MinIO bucket '%s' created successfully.", bucketName)
	} else {
		log.Printf("MinIO bucket '%s' already exists.", bucketName)
	}

	globalMinioClient = &MinioClient{
		Client:     minioClient,
		BucketName: bucketName,
	}
	log.Println("MinIO client initialized successfully.")
	return nil
}

// GetGlobalMinioClient returns the initialized global MinIO client.
// This is a convenience function; dependency injection is preferred for larger applications.
func GetGlobalMinioClient() (*MinioClient, error) {
	if globalMinioClient == nil {
		return nil, fmt.Errorf("MinIO client not initialized. Call InitMinioClient first")
	}
	return globalMinioClient, nil
}

// UploadFile uploads a file to the configured MinIO bucket and returns the unique object name.
// objectName is generated internally to ensure uniqueness.
func (mc *MinioClient) UploadFile(ctx context.Context, originalFilename string, reader io.Reader, size int64, contentType string) (string, error) {
	if mc.Client == nil {
		return "", fmt.Errorf("MinIO client not initialized properly in MinioClient struct")
	}
	if mc.BucketName == "" {
		return "", fmt.Errorf("MinIO bucket name not configured in MinioClient struct")
	}

	// Generate a unique object name using UUID and preserving the original file extension
	uniqueID := uuid.New().String()
	extension := filepath.Ext(originalFilename)
	objectName := fmt.Sprintf("%s%s", uniqueID, extension)

	uploadInfo, err := mc.Client.PutObject(ctx, mc.BucketName, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to MinIO (bucket: %s, object: %s): %w", mc.BucketName, objectName, err)
	}

	log.Printf("Successfully uploaded '%s' of size %d to MinIO. ETag: %s", objectName, uploadInfo.Size, uploadInfo.ETag)
	return objectName, nil
}

// DeleteFile deletes a file from the configured MinIO bucket.
func (mc *MinioClient) DeleteFile(ctx context.Context, objectName string) error {
	if mc.Client == nil {
		return fmt.Errorf("MinIO client not initialized properly in MinioClient struct")
	}
	if mc.BucketName == "" {
		return fmt.Errorf("MinIO bucket name not configured in MinioClient struct")
	}

	err := mc.Client.RemoveObject(ctx, mc.BucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object '%s' from MinIO bucket '%s': %w", objectName, mc.BucketName, err)
	}

	log.Printf("Successfully deleted object '%s' from MinIO bucket '%s'.", objectName, mc.BucketName)
	return nil
}

// GetFileLink generates a presigned URL for an object.
// This is useful for providing temporary access to files.
// Note: For MVP, direct download through a handler might be simpler if presigned URLs add too much complexity.
// func (mc *MinioClient) GetFileLink(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
// 	if mc.Client == nil {
// 		return "", fmt.Errorf("MinIO client not initialized")
// 	}
// 	reqParams := make(url.Values)
// 	// reqParams.Set("response-content-disposition", "attachment; filename=\""+objectName+"\"") // To force download
// 	presignedURL, err := mc.Client.PresignedGetObject(ctx, mc.BucketName, objectName, expiry, reqParams)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to generate presigned URL for object '%s': %w", objectName, err)
// 	}
// 	return presignedURL.String(), nil
// }

// GetFileBytes retrieves a file from MinIO as a byte slice.
func (mc *MinioClient) GetFileBytes(ctx context.Context, objectName string) ([]byte, error) {
	if mc.Client == nil {
		return nil, fmt.Errorf("MinIO client not initialized properly in MinioClient struct")
	}
	if mc.BucketName == "" {
		return nil, fmt.Errorf("MinIO bucket name not configured in MinioClient struct")
	}

	object, err := mc.Client.GetObject(ctx, mc.BucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object '%s' from bucket '%s': %w", objectName, mc.BucketName, err)
	}
	defer object.Close()

	// Check object stats to prevent reading excessively large files into memory if needed,
	// though for typical audio files this might be acceptable.
	// stat, err := object.Stat()
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get object stats for '%s': %w", objectName, err)
	// }
	// if stat.Size > someSensibleLimit {
	//  return nil, fmt.Errorf("file %s is too large (%d bytes)", objectName, stat.Size)
	// }

	data, err := io.ReadAll(object)
	if err != nil {
		return nil, fmt.Errorf("failed to read object '%s' data: %w", objectName, err)
	}

	return data, nil
}

// GetFileReader retrieves a file from MinIO as an io.ReadCloser.
// The caller is responsible for closing the reader.
func (mc *MinioClient) GetFileReader(ctx context.Context, objectName string) (io.ReadCloser, int64, error) {
	if mc.Client == nil {
		return nil, 0, fmt.Errorf("MinIO client not initialized properly in MinioClient struct")
	}
	if mc.BucketName == "" {
		return nil, 0, fmt.Errorf("MinIO bucket name not configured in MinioClient struct")
	}

	object, err := mc.Client.GetObject(ctx, mc.BucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get object '%s' from bucket '%s': %w", objectName, mc.BucketName, err)
	}

	stat, err := object.Stat()
	if err != nil {
		object.Close() // Close object if stat fails
		return nil, 0, fmt.Errorf("failed to get object stats for '%s': %w", objectName, err)
	}

	return object, stat.Size, nil
}
