-- Table for Admin Users
CREATE TABLE admin_users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    hashed_password VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Table for Vendor Configurations
CREATE TABLE vendor_configs (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL, -- e.g., Google ASR, Azure TTS
    api_type VARCHAR(50) NOT NULL, -- ASR, TTS, LLM
    api_key TEXT,
    api_secret TEXT,
    api_endpoint TEXT,
    supported_models JSONB, -- e.g., [{"model_id": "google-long", "name": "Google Latest Long"}, ...]
    other_configs JSONB, -- For any other vendor-specific settings
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Table for ASR Test Cases
CREATE TABLE asr_test_cases (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    language_code VARCHAR(20), -- e.g., en-US, zh-CN
    audio_file_path TEXT NOT NULL, -- Path/key in object storage
    ground_truth_text TEXT,
    tags JSONB, -- e.g., ["short_audio", "noisy"]
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Table for Evaluation Jobs
CREATE TABLE evaluation_jobs (
    id SERIAL PRIMARY KEY,
    job_name VARCHAR(255),
    job_type VARCHAR(50) NOT NULL, -- ASR, TTS, LLM
    status VARCHAR(50) NOT NULL, -- e.g., pending, running, completed, failed
    vendor_config_ids JSONB, -- Array of vendor_config_id used in this job
    test_case_ids JSONB, -- Array of test_case_id used in this job for ASR/TTS, or prompt_ids for LLM
    parameters JSONB, -- Specific parameters used for this job run
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Table for ASR Evaluation Results
CREATE TABLE asr_evaluation_results (
    id SERIAL PRIMARY KEY,
    job_id INTEGER REFERENCES evaluation_jobs(id) ON DELETE CASCADE,
    asr_test_case_id INTEGER REFERENCES asr_test_cases(id) ON DELETE CASCADE,
    vendor_config_id INTEGER REFERENCES vendor_configs(id) ON DELETE SET NULL,
    recognized_text TEXT,
    cer FLOAT,
    wer FLOAT,
    ser FLOAT,
    latency_ms INTEGER, -- Latency in milliseconds
    raw_vendor_response JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Comments for PostgreSQL schema (as requested in task, though typically not part of init.sql)
-- admin_users: Stores credentials for platform administrators.
-- vendor_configs: Manages API credentials and configurations for various AI service vendors.
--   - name: User-friendly name for the configuration (e.g., "Google Cloud Speech-to-Text").
--   - api_type: Specifies the type of service (ASR, TTS, LLM).
--   - supported_models: JSON array detailing models available under this configuration.
--   - other_configs: Flexible JSON field for additional settings.
-- asr_test_cases: Contains information about test audio files for Automatic Speech Recognition.
--   - audio_file_path: Reference to the audio file's location in MinIO.
--   - ground_truth_text: The correct transcription of the audio.
--   - tags: JSON array for categorizing test cases.
-- evaluation_jobs: Tracks evaluation tasks initiated by users.
--   - job_type: The type of evaluation being performed (ASR, TTS, LLM).
--   - status: Current state of the job.
--   - vendor_config_ids: JSON array of vendor configurations being tested.
--   - test_case_ids: JSON array of test cases (or prompts for LLM) included in the job.
-- asr_evaluation_results: Stores the outcomes of ASR evaluation tasks.
--   - job_id: Foreign key linking to the evaluation_jobs table.
--   - asr_test_case_id: Foreign key linking to the asr_test_cases table.
--   - vendor_config_id: Foreign key linking to the vendor_configs table.
--   - cer, wer, ser: Common ASR accuracy metrics (Character Error Rate, Word Error Rate, Sentence Error Rate).
--   - latency_ms: Time taken by the vendor to process the audio.
--   - raw_vendor_response: Full JSON response from the vendor API for detailed analysis.
