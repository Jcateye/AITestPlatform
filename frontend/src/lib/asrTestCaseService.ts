// Define the base URL for the API. Adjust if your proxy/gateway is different.
// Assuming Next.js proxy is set up or direct backend URL for client-side calls.
const API_BASE_URL = '/api/admin/asr-test-cases'; // This might be proxied by Next.js to the backend

export interface ASRTestCase {
    id: number;
    name: string;
    language_code?: string | null;
    audio_file_path: string;
    ground_truth_text?: string | null;
    tags?: string[]; // Assuming backend sends tags as an array of strings if parsed from JSON
    description?: string | null;
    created_at: string; // ISO date string
    updated_at: string; // ISO date string
}

// For updates, we typically don't send all fields.
// Audio file path is not updatable via metadata endpoint.
export interface ASRTestCaseMetadata {
    name?: string;
    language_code?: string | null;
    ground_truth_text?: string | null;
    tags?: string[]; // Will be stringified if backend expects JSON string for tags
    description?: string | null;
}

async function handleResponse<T>(response: Response): Promise<T> {
    if (!response.ok) {
        const errorData = await response.json().catch(() => ({ message: 'Unknown error occurred' }));
        throw new Error(errorData.message || `API request failed with status ${response.status}`);
    }
    // For DELETE requests or others that might not return JSON body
    if (response.status === 204 || response.headers.get("content-length") === "0") {
        return null as T; // Or appropriate type for no content
    }
    return response.json();
}


export async function createASRTestCase(formData: FormData): Promise<ASRTestCase> {
    const response = await fetch(`${API_BASE_URL}`, {
        method: 'POST',
        body: formData,
        // Headers are automatically set by browser for FormData, including Content-Type: multipart/form-data
        // Add authorization headers if needed, e.g., if using token-based auth not handled by cookies
    });
    return handleResponse<ASRTestCase>(response);
}

export async function listASRTestCases(params: { language_code?: string; tags?: string } = {}): Promise<ASRTestCase[]> {
    const queryParams = new URLSearchParams();
    if (params.language_code) {
        queryParams.append('language_code', params.language_code);
    }
    if (params.tags) {
        queryParams.append('tags', params.tags); // Assuming backend expects comma-separated string for tags query
    }
    const response = await fetch(`${API_BASE_URL}?${queryParams.toString()}`, {
        method: 'GET',
    });
    return handleResponse<ASRTestCase[]>(response);
}

export async function getASRTestCase(id: string): Promise<ASRTestCase> {
    const response = await fetch(`${API_BASE_URL}/${id}`, {
        method: 'GET',
    });
    return handleResponse<ASRTestCase>(response);
}

export async function updateASRTestCase(id: string, data: Partial<ASRTestCaseMetadata>): Promise<ASRTestCase> {
    // Backend's ASRTestCase `tags` field is JSONB.
    // If `data.tags` is an array, it should be stringified before sending,
    // or ensure backend handler for `UpdateASRTestCase` can parse it if sent as actual JSON array.
    // The backend `UpdateASRTestCase` in Go expects a map[string]interface{},
    // and for "tags", it expects `json.RawMessage` or a string that can be parsed to it.
    // So, sending `data` as JSON where `tags` is an array of strings should be fine.

    const payload = { ...data };
    // If tags are being sent and are an array, they're fine as is for JSON.
    // If backend expects stringified JSON for tags in `map[string]interface{}` for `json.RawMessage` this is not it.
    // The Go backend's `UpdateASRTestCase` handler for `json.RawMessage` should correctly handle a JSON array of strings.

    const response = await fetch(`${API_BASE_URL}/${id}`, {
        method: 'PUT',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
    });
    return handleResponse<ASRTestCase>(response);
}

export async function deleteASRTestCase(id: string): Promise<void> {
    const response = await fetch(`${API_BASE_URL}/${id}`, {
        method: 'DELETE',
    });
    // Delete might return 200 with a message or 204 No Content.
    // handleResponse will need to accommodate this.
    // Updated handleResponse to manage 204.
    await handleResponse<void>(response); // Or any appropriate type for a successful delete
}
