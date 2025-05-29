import { ASRTestCase } from './asrTestCaseService'; // Assuming ASRTestCase type might be useful
import { VendorConfig } from './vendorConfigService'; // Assuming VendorConfig type might be useful

// Base URL for job-related APIs
const API_BASE_URL = '/api/admin/jobs'; // Assuming Next.js proxy

// --------------- TypeScript Types ---------------

export interface EvaluationJob {
    id: number;
    job_name?: string | null;
    job_type: string; // "ASR", "TTS", "LLM"
    status: string; // "PENDING", "RUNNING", "COMPLETED", "FAILED"
    vendor_config_ids: number[]; // Backend sends JSON array of numbers
    test_case_ids: number[];     // Backend sends JSON array of numbers
    parameters?: Record<string, any> | null; // Parsed JSON from backend
    created_at: string; // ISO date string
    updated_at: string; // ISO date string
    started_at?: string | null; // ISO date string
    completed_at?: string | null; // ISO date string
}

export interface EvaluationJobPayload {
    job_name?: string;
    test_case_ids: number[];
    vendor_config_ids: number[];
    parameters?: Record<string, any> | string | null; // Allow string for JSON input, then parse
}

export interface ASREvaluationResult {
    id: number;
    job_id: number;
    asr_test_case_id: number;
    vendor_config_id: number;
    recognized_text?: string | null;
    cer?: number | null;
    wer?: number | null;
    ser?: number | null; // Optional for MVP
    latency_ms?: number | null;
    raw_vendor_response?: Record<string, any> | null; // Parsed JSON
    created_at: string; // ISO date string

    // For frontend display convenience, these can be populated after fetching
    test_case_name?: string;
    vendor_name?: string;
    ground_truth?: string | null;
}


// --------------- Helper Functions ---------------

async function handleResponse<T>(response: Response): Promise<T> {
    if (!response.ok) {
        const errorData = await response.json().catch(() => ({ message: 'Unknown error occurred' }));
        throw new Error(errorData.message || `API request failed with status ${response.status}`);
    }
    if (response.status === 204 || response.headers.get("content-length") === "0") {
        return null as T;
    }
    const data = await response.json();
    // Assuming backend sends vendor_config_ids and test_case_ids as JSON arrays of numbers.
    // If they are stringified JSON within a JSON payload, they'd need parsing here.
    // Based on Go backend, they should be proper JSON arrays.
    return data;
}


// --------------- API Client Functions ---------------

export async function createASRJob(payload: EvaluationJobPayload): Promise<EvaluationJob> {
    let parametersToSend: string | null = null;
    if (payload.parameters) {
        if (typeof payload.parameters === 'string') {
            try {
                // Validate JSON string if it's a string
                JSON.parse(payload.parameters);
                parametersToSend = payload.parameters; // Send as string if it's valid JSON string
            } catch (e) {
                throw new Error("Parameters field contains invalid JSON.");
            }
        } else {
             parametersToSend = JSON.stringify(payload.parameters); // Stringify if it's an object
        }
    }


    const response = await fetch(`${API_BASE_URL}/asr`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            job_name: payload.job_name,
            test_case_ids: payload.test_case_ids,
            vendor_config_ids: payload.vendor_config_ids,
            parameters: parametersToSend, // Send stringified JSON or null
        }),
        // Add authorization headers if needed
    });
    return handleResponse<EvaluationJob>(response);
}

export async function getJob(id: string): Promise<EvaluationJob> {
    const response = await fetch(`${API_BASE_URL}/${id}`, {
        method: 'GET',
        // Add authorization headers if needed
    });
    const job = await handleResponse<EvaluationJob>(response);
    // Ensure IDs are numbers, not strings, if backend sends them as such in JSON
    // This should be handled by Go's json.Marshal of []int correctly.
    // job.vendor_config_ids = (job.vendor_config_ids || []).map(Number);
    // job.test_case_ids = (job.test_case_ids || []).map(Number);
    return job;
}

export async function listJobs(params: { job_type?: string } = {}): Promise<EvaluationJob[]> {
    const queryParams = new URLSearchParams();
    if (params.job_type) {
        queryParams.append('job_type', params.job_type);
    }
    const response = await fetch(`${API_BASE_URL}?${queryParams.toString()}`, {
        method: 'GET',
        // Add authorization headers if needed
    });
    const jobs = await handleResponse<EvaluationJob[]>(response);
    // return jobs.map(job => ({
    //     ...job,
    //     vendor_config_ids: (job.vendor_config_ids || []).map(Number),
    //     test_case_ids: (job.test_case_ids || []).map(Number),
    // }));
    return jobs;
}

export async function getJobResults(jobId: string): Promise<ASREvaluationResult[]> {
    const response = await fetch(`${API_BASE_URL}/${jobId}/results`, {
        method: 'GET',
        // Add authorization headers if needed
    });
    return handleResponse<ASREvaluationResult[]>(response);
}
