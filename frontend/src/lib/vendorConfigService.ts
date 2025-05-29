// Base URL for vendor configurations
const API_BASE_URL = '/api/admin/vendors'; // Assuming Next.js proxy

export interface VendorSupportedModel {
    model_id: string;
    name: string;
    // Add other relevant fields if any
}

export interface VendorConfig {
    id: number;
    name: string;
    api_type: string; // "ASR", "TTS", "LLM"
    api_key?: string | null;
    api_secret?: string | null;
    api_endpoint?: string | null;
    supported_models?: VendorSupportedModel[] | null; // Assuming backend sends parsed JSON
    other_configs?: Record<string, any> | null; // Parsed JSON
    created_at: string; // ISO date string
    updated_at: string; // ISO date string
}

async function handleResponse<T>(response: Response): Promise<T> {
    if (!response.ok) {
        const errorData = await response.json().catch(() => ({ message: 'Unknown error occurred' }));
        throw new Error(errorData.message || `API request failed with status ${response.status}`);
    }
    if (response.status === 204 || response.headers.get("content-length") === "0") {
        return null as T;
    }
    return response.json();
}

export async function listVendorConfigs(params: { api_type?: string } = {}): Promise<VendorConfig[]> {
    const queryParams = new URLSearchParams();
    if (params.api_type) {
        queryParams.append('api_type', params.api_type);
    }
    const response = await fetch(`${API_BASE_URL}?${queryParams.toString()}`, {
        method: 'GET',
        // Add authorization headers if needed
    });
    const configs = await handleResponse<VendorConfig[]>(response);
    
    // The backend might return supported_models and other_configs as stringified JSON
    // if not handled properly by JSONB scanning in Go.
    // For this service, we assume they are already parsed or the Go service handles it.
    // If they were strings, parsing would be needed here:
    // return configs.map(config => ({
    //   ...config,
    //   supported_models: typeof config.supported_models === 'string' ? JSON.parse(config.supported_models) : config.supported_models,
    //   other_configs: typeof config.other_configs === 'string' ? JSON.parse(config.other_configs) : config.other_configs,
    // }));
    return configs;
}

// Add other CRUD functions for vendor configs if needed in the future
// e.g., getVendorConfig(id: string), createVendorConfig(data: NewVendorConfig), etc.
