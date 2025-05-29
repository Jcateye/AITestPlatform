# MVP Setup and Usage Guide - Unified AI Evaluation Platform

## 1. Overview

This guide describes how to set up and run the Minimum Viable Product (MVP) of the Unified AI Evaluation Platform. The MVP focuses on evaluating Automatic Speech Recognition (ASR) services using a mock vendor adapter. It allows administrators to:

*   Manage ASR vendor configurations.
*   Manage ASR test cases (audio files and ground truth).
*   Create and run ASR evaluation jobs.
*   View evaluation results, including metrics like Word Error Rate (WER) and Character Error Rate (CER).

## 2. Prerequisites

*   **Go:** Version 1.20+ (or as specified in `go.mod`)
*   **Node.js:** Version 18+ (for running the Next.js frontend)
*   **Docker:** Latest version (for easily running PostgreSQL and MinIO)
*   **`psql`:** PostgreSQL command-line tool (optional, for manual database checks).
*   **`git`:** For cloning the repository.

## 3. Database & Object Storage Setup

### 3.1. PostgreSQL (Database)

1.  **Run PostgreSQL using Docker:**
    ```bash
    docker run --name aieval-db \
      -e POSTGRES_USER=admin \
      -e POSTGRES_PASSWORD=admin \
      -e POSTGRES_DB=aieval_db \
      -p 5432:5432 \
      -d postgres:15
    ```
    *   This command starts a PostgreSQL container named `aieval-db`.
    *   Username: `admin`, Password: `admin`, Database: `aieval_db`.
    *   The database will be accessible on `localhost:5432`.

2.  **Initialize Database Schema:**
    *   Once the backend is set up (see section 4), it will expect the tables defined in `backend/internal/datastore/init.sql`.
    *   You can manually apply this schema using `psql` or a database GUI tool:
        ```bash
        psql -h localhost -U admin -d aieval_db -f backend/internal/datastore/init.sql
        ```
        (You might be prompted for the password `admin`).

### 3.2. MinIO (Object Storage)

1.  **Run MinIO using Docker:**
    ```bash
    docker run --name aieval-minio \
      -p 9000:9000 \
      -p 9001:9001 \
      -e "MINIO_ROOT_USER=minioadmin" \
      -e "MINIO_ROOT_PASSWORD=minioadmin" \
      -d quay.io/minio/minio server /data --console-address ":9001"
    ```
    *   This starts a MinIO container named `aieval-minio`.
    *   API Endpoint: `localhost:9000`
    *   Console: `http://localhost:9001`
    *   Access Key (Root User): `minioadmin`
    *   Secret Key (Root Password): `minioadmin`

2.  **Create MinIO Bucket:**
    *   Open the MinIO console at `http://localhost:9001` and log in with `minioadmin`/`minioadmin`.
    *   Create a new bucket. The default bucket name expected by the backend is `aieval-bucket`. You can configure this via environment variables (see `MINIO_BUCKET_NAME` below).

## 4. Backend Setup (`backend/` directory)

1.  **Navigate to the backend directory:**
    ```bash
    cd backend
    ```

2.  **Install Dependencies:**
    ```bash
    go mod tidy
    ```

3.  **Required Environment Variables:**
    Create a `.env` file in the `backend/` directory or set these variables in your shell environment:

    ```env
    # Admin Credentials for platform login
    ADMIN_USERNAME=admin
    ADMIN_PASSWORD=adminmvp # Change this for a real deployment

    # Database Connection
    DB_HOST=localhost
    DB_PORT=5432
    DB_USER=admin
    DB_PASSWORD=admin
    DB_NAME=aieval_db
    DB_SSLMODE=disable # Use "enable" for production with SSL

    # MinIO Object Storage Connection
    MINIO_ENDPOINT=localhost:9000
    MINIO_ACCESS_KEY_ID=minioadmin
    MINIO_SECRET_ACCESS_KEY=minioadmin
    MINIO_BUCKET_NAME=aieval-bucket # Must match the bucket you created
    MINIO_USE_SSL=false # Set to true if MinIO is configured with SSL

    # Gin Mode (optional, defaults to debug)
    GIN_MODE=debug # or "release" for production

    # Server Port (optional, defaults to 8080)
    SERVER_PORT=8080
    
    # Optional: Google Cloud Credentials (if not using OtherConfigs.google_credentials_path)
    # GOOGLE_APPLICATION_CREDENTIALS=/path/to/your/gcp-service-account.json
    ```

4.  **Run the Backend Server:**
    *   Ensure your `backend/cmd/server/main.go` file is set up to initialize and run the server. The example `main` function is typically found within comments in `backend/internal/apigateway/router.go`. You'll need to move this to `cmd/server/main.go`.
    *   Once `main.go` is in place:
    ```bash
    go run cmd/server/main.go
    ```
    *   The backend server should start, typically on `http://localhost:8080`.

## 5. Frontend Setup (`frontend/` directory)

1.  **Navigate to the frontend directory:**
    ```bash
    cd frontend
    ```

2.  **Install Dependencies:**
    Choose one based on your preference (if `package-lock.json` exists, `npm` is preferred; if `yarn.lock` exists, `yarn` is preferred).
    ```bash
    npm install
    # OR
    # yarn install
    ```

3.  **Environment Variables (Optional):**
    *   The frontend is configured to proxy API requests starting with `/api/` to the backend (default `http://localhost:8080`). This is handled by `next.config.mjs` (or `next.config.js`).
    *   If you need to change the backend URL that the proxy targets, you might adjust it in `next.config.mjs` or set an environment variable if the config is set up to read one. For this MVP, the default proxy should work if the backend is on `localhost:8080`.
    *   No specific `.env` file is strictly required for the frontend to connect to the backend if using the proxy and default ports.

4.  **Run the Frontend Development Server:**
    ```bash
    npm run dev
    # OR
    # yarn dev
    ```
    *   The frontend development server will start, typically on `http://localhost:3000`.

## 6. Accessing the Application

*   **Frontend (User Interface):** `http://localhost:3000`
    *   Navigate to `/admin/login` (or the root which might redirect to login) to access admin functionalities. The default admin page after login should be `/admin/asr-test-cases`.
*   **Backend API:** `http://localhost:8080`
    *   The frontend interacts with this API. Direct interaction is possible via tools like Postman or `curl`.
*   **MinIO Console:** `http://localhost:9001`
    *   To check uploaded files.
*   **Database:** `localhost:5432`
    *   To inspect data directly (e.g., using `psql` or a GUI tool).

## 7. Initial `main.go` for Backend

If `backend/cmd/server/main.go` does not exist, create it with the following content (adapted from the example in `apigateway/router.go`):

```go
// backend/cmd/server/main.go
package main

import (
	"fmt" // Required for fmt.Sprintf
	"log"
	"os"
	"unified-ai-eval-platform/backend/internal/apigateway"
	"unified-ai-eval-platform/backend/internal/auth"
	"unified-ai-eval-platform/backend/internal/configmanagement" // For InitHandlers
	"unified-ai-eval-platform/backend/internal/datastore"
	"unified-ai-eval-platform/backend/internal/objectstore"
	"unified-ai-eval-platform/backend/internal/coreengine/vendoradapters" // For InitAdapterRegistry
)

func main() {
	log.Println("Starting Unified AI Evaluation Platform Backend...")

	// Load configurations at startup
	auth.LoadAdminCredentials()
	log.Println("Admin credentials loaded.")

	// Initialize DB connection
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbSSLMode := os.Getenv("DB_SSLMODE")

	if dbHost == "" { dbHost = "localhost" }
	if dbPort == "" { dbPort = "5432" }
	if dbUser == "" { dbUser = "postgres" } // Default user from Docker setup if not overridden
	if dbName == "" { dbName = "aieval_db" } // Default DB from Docker setup
	if dbSSLMode == "" { dbSSLMode = "disable" }
	if dbPassword == "" { log.Println("WARNING: DB_PASSWORD environment variable not set.") }


	dataSourceName := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	if err := datastore.InitDB(dataSourceName); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer datastore.DB.Close()
	log.Println("Database connection initialized successfully.")

	// Initialize MinIO Client (for global access by adapters if needed)
	if err := objectstore.InitMinioClient(); err != nil {
		log.Fatalf("Failed to initialize MinIO client: %v", err)
	}
	minioClient, err := objectstore.GetGlobalMinioClient()
    if err != nil {
        log.Fatalf("Failed to get global MinIO client: %v", err)
    }
	vendoradapters.InitAdapterRegistry(minioClient) // Pass it to the adapter registry
	log.Println("MinIO client initialized and passed to adapter registry.")
	
	// Pass the DB instance to handlers if they are structured to receive it
	// (Currently, many datastore functions use a global DB variable)
	configmanagement.InitHandlers(datastore.DB) 
	log.Println("Configmanagement handlers initialized.")


	// Setup router
	router := apigateway.SetupRouter()
	log.Println("HTTP router initialized.")

	// Start server
	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "8080" // Default port
	}
	log.Printf("Starting server on port :%s", serverPort)
	if err := router.Run(":" + serverPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
```

This `main.go` includes initialization for the MinIO client and passing it to the adapter registry. Ensure all imported packages are correct based on your project structure.

## 8. Configuring ASR Vendor Adapters

The platform supports multiple ASR vendors. Configurations for these vendors are managed through the UI ("Vendor Configurations" section in the admin panel) and are crucial for the system to interact with the respective ASR services.

**General Configuration Approach:**

*   **Name:** This is a user-defined name for the configuration (e.g., "MyGoogleASR-US-Central", "Deepgram-Nova2-Finance"). However, to use the specific pre-built adapters, the name **must match** the keys used in the backend's adapter registry:
    *   `MockASR` (for successful mock responses)
    *   `MockASR-Error` (to simulate errors from the mock adapter)
    *   `GoogleASR`
    *   `MicrosoftASR`
    *   `DeepgramASR`
    *   `TencentASR`
    *   `AlibabaASR` (Currently stubbed - will return a mock error)
    *   `VolcengineASR`
*   **API Type:** Should be set to "ASR" for speech-to-text services.
*   **APIKey & APISecret:** These fields are used for primary authentication credentials.
    *   `APIKey`: Typically holds the primary key, access key ID, or subscription key.
    *   `APISecret`: Typically holds the secret key associated with the APIKey. Some vendors might only use an API Key and not a secret.
*   **API Endpoint:** Optional. If provided, it can override the default API endpoint hardcoded in some adapters (e.g., TencentASR).
*   **OtherConfigs (JSON):** This field is critical for vendor-specific settings not covered by the standard fields. It's a JSON object where you can define various parameters.
    *   **Regions:** Many cloud providers require a region specification (e.g., `azure_region`, `tencent_region`, `volcengine_region`).
    *   **Application/Project IDs:** Some vendors require an App ID or Project ID (e.g., `alibaba_app_key`, `tencent_app_id`, `volcengine_app_id`).
    *   **Credential Files:** For services like Google Cloud, you can specify a path to a service account JSON file (`google_credentials_path`) if you are not using environment variables for authentication. The path should be accessible from the backend server's filesystem.
    *   **Nested `config` Object:** A common pattern is to include a `config` object within `OtherConfigs` to pass through default parameters to the ASR engine (like model name, punctuation settings, specific engine types). For example:
        ```json
        {
          "config": {
            "model": "specific-model-name",
            "punctuate": true
          }
        }
        ```
        The keys and values within this `config` object are vendor-specific and interpreted by the respective adapter.

### Vendor-Specific Configuration Details:

Below are guidelines for configuring each supported ASR vendor. Remember to create these configurations via the UI under "Vendor Configurations".

#### **MockASR / MockASR-Error**
*   **Name:** `MockASR` (for successful mock responses) or `MockASR-Error` (to simulate errors).
*   **API Type:** `ASR`
*   **APIKey, APISecret, OtherConfigs:** Not strictly required by the mock adapter, can be left blank or filled with dummy data.

#### **Google Cloud Speech-to-Text (GoogleASR)**
*   **Name:** `GoogleASR`
*   **API Type:** `ASR`
*   **APIKey / APISecret:** Not directly used if using a service account JSON file or ADC. Can be left blank.
*   **OtherConfigs (JSON):**
    *   `google_credentials_path`: (Optional) Absolute path to the Google Cloud service account JSON key file on the server running the backend. If not provided, the adapter will try to use credentials from the `GOOGLE_APPLICATION_CREDENTIALS` environment variable or Application Default Credentials (ADC).
        *   Example: `{"google_credentials_path": "/app/secrets/gcp-credentials.json"}`
    *   `config`: (Optional) Object for additional recognition parameters.
        *   Example: `{"config": {"model": "telephony", "useEnhanced": true, "sample_rate_hertz": 8000, "encoding": "MULAW"}}`
        *   Refer to Google Cloud Speech-to-Text documentation for available `RecognitionConfig` fields.

#### **Microsoft Azure Speech Service (MicrosoftASR)**
*   **Name:** `MicrosoftASR`
*   **API Type:** `ASR`
*   **APIKey:** Your Azure Speech Service Subscription Key.
*   **APISecret:** Can be left blank.
*   **OtherConfigs (JSON):**
    *   `azure_region`: (Required) The region for your Azure Speech service (e.g., "eastus", "westus2").
    *   `config`: (Optional) Object for additional recognition parameters.
        *   Example: `{"azure_region": "eastus", "config": {"profanity_option": "Removed"}}` (Valid options: "Masked", "Removed", "Raw")

#### **Deepgram ASR (DeepgramASR)**
*   **Name:** `DeepgramASR`
*   **API Type:** `ASR`
*   **APIKey:** Your Deepgram API Key.
*   **APISecret:** Can be left blank.
*   **OtherConfigs (JSON):**
    *   `config`: (Optional) Object for additional query parameters sent to the Deepgram API.
        *   Example: `{"config": {"model": "nova-2-general", "punctuate": "true", "diarize": "true"}}`
        *   Refer to Deepgram API documentation for available parameters.

#### **Tencent Cloud Speech Recognition (TencentASR)**
*   **Name:** `TencentASR`
*   **API Type:** `ASR`
*   **APIKey:** Your Tencent Cloud SecretId.
*   **APISecret:** Your Tencent Cloud SecretKey.
*   **OtherConfigs (JSON):**
    *   `tencent_region`: (Required) The region for your Tencent Cloud ASR service (e.g., "ap-guangzhou", "ap-shanghai").
    *   `tencent_app_id`: (Optional but often needed) Your Tencent Cloud AppId (numeric, e.g., `1250000000`).
    *   `config`: (Optional) Object for additional parameters.
        *   Example: `{"tencent_region": "ap-guangzhou", "tencent_app_id": 1234567890, "config": {"engine_model_type": "16k_en_standard"}}`
        *   Refer to Tencent Cloud ASR documentation for `EngineModelType` and other parameters.

#### **Alibaba Cloud Intelligent Speech Interaction (AlibabaASR)**
*   **Name:** `AlibabaASR`
*   **API Type:** `ASR`
*   **Status:** **Currently Stubbed.** The adapter is present in the codebase but will return a mock error indicating "Alibaba ASR SDK not available in build environment" due to SDK download issues during development. Do not expect transcription results from this adapter in the MVP.
*   **Planned Configuration (for future use when fully implemented):**
    *   `APIKey`: Your Alibaba Cloud AccessKeyId.
    *   `APISecret`: Your Alibaba Cloud AccessKeySecret.
    *   `OtherConfigs (JSON)`:
        *   `alibaba_app_key`: (Required) Your Alibaba Cloud NLS AppKey.
        *   `alibaba_region_id`: (Optional, sometimes needed) e.g., "cn-shanghai".
        *   `config`: (Optional) e.g., `{"format": "pcm", "sample_rate": 16000, "enable_punctuation_prediction": true}`
        *   Example: `{"alibaba_app_key": "YOUR_APP_KEY_HERE", "alibaba_region_id": "cn-shanghai", "config": {"format": "wav"}}`

#### **Volcengine Speech Recognition (VolcengineASR)**
*   **Name:** `VolcengineASR`
*   **API Type:** `ASR`
*   **APIKey:** Your Volcengine AccessKeyID.
*   **APISecret:** Your Volcengine SecretAccessKey.
*   **OtherConfigs (JSON):**
    *   `volcengine_region`: (Required) The region for your Volcengine ASR service (e.g., "cn-north-1").
    *   `volcengine_app_id`: (Required) Your Volcengine AppId (string).
    *   `volcengine_cluster`: (Optional) Cluster name, if applicable.
    *   `config`: (Optional) Object for additional parameters.
        *   Example: `{"volcengine_region": "cn-north-1", "volcengine_app_id": "YOUR_APP_ID", "config": {"engine_type": "16k_en_phone_call_common", "add_punc": true}}`
        *   Refer to Volcengine Speech Recognition documentation for `EngineType`, `Format`, and other parameters.

**Important Note on Environment Variables vs. `OtherConfigs`:**
For some adapters (like Google Cloud), credentials can be supplied via environment variables (e.g., `GOOGLE_APPLICATION_CREDENTIALS`). If such environment variables are set in the backend's runtime environment, they might take precedence or be used as a fallback if specific paths are not provided in `OtherConfigs`.
The `main.go` example provided earlier shows how to load common environment variables for DB and Admin credentials. Vendor-specific credential environment variables (like `GOOGLE_APPLICATION_CREDENTIALS`) would also be set in the same environment where the backend server runs.
