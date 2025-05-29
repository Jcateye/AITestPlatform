'use client';

import { useEffect, useState, FormEvent } from 'react';
import { useRouter, useParams } from 'next/navigation';
import Link from 'next/link';
import { ASRTestCase, ASRTestCaseMetadata, getASRTestCase, updateASRTestCase } from '@/lib/asrTestCaseService';

export default function EditASRTestCasePage() {
    const router = useRouter();
    const params = useParams();
    const id = params.id as string;

    const [testCase, setTestCase] = useState<ASRTestCase | null>(null);
    const [formData, setFormData] = useState<Partial<ASRTestCaseMetadata>>({
        name: '',
        language_code: '',
        ground_truth_text: '',
        tags: [],
        description: '',
    });
    const [isLoading, setIsLoading] = useState(true);
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [successMessage, setSuccessMessage] = useState<string | null>(null);

    useEffect(() => {
        if (id) {
            setIsLoading(true);
            getASRTestCase(id)
                .then((data) => {
                    setTestCase(data);
                    setFormData({
                        name: data.name || '',
                        language_code: data.language_code || '',
                        ground_truth_text: data.ground_truth_text || '',
                        tags: data.tags || [], // Assuming tags are string[]
                        description: data.description || '',
                    });
                })
                .catch((err) => {
                    setError(err.message || 'Failed to fetch test case data.');
                })
                .finally(() => {
                    setIsLoading(false);
                });
        }
    }, [id]);

    const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
        const { name, value } = e.target;
        setFormData(prev => ({ ...prev, [name]: value }));
    };

    const handleTagsChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        // Assuming tags are input as comma-separated string and stored as string[] in form data
        const tagsArray = e.target.value.split(',').map(tag => tag.trim()).filter(tag => tag);
        setFormData(prev => ({ ...prev, tags: tagsArray }));
    };

    const handleSubmit = async (event: FormEvent) => {
        event.preventDefault();
        setError(null);
        setSuccessMessage(null);

        if (!formData.name?.trim()) {
            setError('Name is required.');
            return;
        }
        
        setIsSubmitting(true);

        try {
            // Ensure tags are correctly formatted if needed by the backend
            // The service `updateASRTestCase` expects data.tags to be string[]
            // which will be JSON.stringified. The Go backend should handle it.
            const payload: Partial<ASRTestCaseMetadata> = {
                ...formData,
                // If language_code, ground_truth_text, description are empty, send null or undefined
                // The current setup sends empty strings, which sql.NullString in Go handles as Valid=true, String="".
                // To send SQL NULL, these should be explicitly set to null.
                language_code: formData.language_code || null,
                ground_truth_text: formData.ground_truth_text || null,
                description: formData.description || null,
                // tags: formData.tags // already in correct format (string[])
            };


            await updateASRTestCase(id, payload);
            setSuccessMessage('Test case updated successfully! Redirecting to list...');
            setTimeout(() => {
                router.push('/admin/asr-test-cases');
            }, 2000);
        } catch (err: any) {
            setError(err.message || 'Failed to update test case.');
        } finally {
            setIsSubmitting(false);
        }
    };

    if (isLoading) return <p className="container mx-auto p-4">Loading test case...</p>;
    if (error && !testCase) return <p className="container mx-auto p-4 text-red-500">Error: {error}</p>; // Error and no data to show form
    if (!testCase) return <p className="container mx-auto p-4">Test case not found.</p>;


    return (
        <div className="container mx-auto p-4">
            <div className="flex justify-between items-center mb-4">
                <h1 className="text-2xl font-bold">Edit ASR Test Case: {testCase.name}</h1>
                <Link href="/admin/asr-test-cases" className="bg-gray-500 hover:bg-gray-700 text-white font-bold py-2 px-4 rounded">
                    Back to List
                </Link>
            </div>

            {error && <p className="text-red-500 bg-red-100 p-3 rounded mb-4">{error}</p>}
            {successMessage && <p className="text-green-500 bg-green-100 p-3 rounded mb-4">{successMessage}</p>}

            <form onSubmit={handleSubmit} className="space-y-4">
                <div>
                    <label htmlFor="name" className="block text-sm font-medium text-gray-300">Name <span className="text-red-500">*</span></label>
                    <input
                        type="text"
                        name="name"
                        id="name"
                        value={formData.name || ''}
                        onChange={handleChange}
                        className="mt-1 block w-full p-2 border border-gray-600 rounded-md shadow-sm bg-gray-700 text-white"
                        required
                    />
                </div>

                <div>
                    <label htmlFor="language_code" className="block text-sm font-medium text-gray-300">Language Code</label>
                    <input
                        type="text"
                        name="language_code"
                        id="language_code"
                        value={formData.language_code || ''}
                        onChange={handleChange}
                        className="mt-1 block w-full p-2 border border-gray-600 rounded-md shadow-sm bg-gray-700 text-white"
                    />
                </div>

                <div>
                    <label htmlFor="ground_truth_text" className="block text-sm font-medium text-gray-300">Ground Truth Text</label>
                    <textarea
                        name="ground_truth_text"
                        id="ground_truth_text"
                        value={formData.ground_truth_text || ''}
                        onChange={handleChange}
                        rows={3}
                        className="mt-1 block w-full p-2 border border-gray-600 rounded-md shadow-sm bg-gray-700 text-white"
                    />
                </div>
                
                <div>
                    <label htmlFor="tags" className="block text-sm font-medium text-gray-300">Tags (comma-separated)</label>
                    <input
                        type="text"
                        name="tags"
                        id="tags"
                        value={formData.tags?.join(',') || ''} // Convert array to comma-separated string for input
                        onChange={handleTagsChange} // Custom handler for tags
                        className="mt-1 block w-full p-2 border border-gray-600 rounded-md shadow-sm bg-gray-700 text-white"
                    />
                </div>

                <div>
                    <label htmlFor="description" className="block text-sm font-medium text-gray-300">Description</label>
                    <textarea
                        name="description"
                        id="description"
                        value={formData.description || ''}
                        onChange={handleChange}
                        rows={3}
                        className="mt-1 block w-full p-2 border border-gray-600 rounded-md shadow-sm bg-gray-700 text-white"
                    />
                </div>
                
                <div className="pt-2">
                    <p className="text-sm text-gray-400">Audio File: <code className="bg-gray-600 p-1 rounded">{testCase.audio_file_path}</code></p>
                    <p className="text-xs text-gray-500">Audio file cannot be changed via this form in the current version.</p>
                </div>

                <div>
                    <button
                        type="submit"
                        disabled={isSubmitting || isLoading}
                        className="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50"
                    >
                        {isSubmitting ? 'Updating...' : 'Update Test Case'}
                    </button>
                </div>
            </form>
        </div>
    );
}
