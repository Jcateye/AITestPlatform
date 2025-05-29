'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { createASRTestCase } from '@/lib/asrTestCaseService';
import Link from 'next/link';

export default function NewASRTestCasePage() {
    const router = useRouter();
    const [name, setName] = useState('');
    const [languageCode, setLanguageCode] = useState('');
    const [groundTruthText, setGroundTruthText] = useState('');
    const [tags, setTags] = useState(''); // Comma-separated
    const [description, setDescription] = useState('');
    const [audioFile, setAudioFile] = useState<File | null>(null);
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [successMessage, setSuccessMessage] = useState<string | null>(null);

    const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        if (event.target.files && event.target.files[0]) {
            setAudioFile(event.target.files[0]);
        }
    };

    const handleSubmit = async (event: React.FormEvent) => {
        event.preventDefault();
        setError(null);
        setSuccessMessage(null);

        if (!name.trim()) {
            setError('Name is required.');
            return;
        }
        if (!audioFile) {
            setError('Audio file is required.');
            return;
        }

        setIsSubmitting(true);

        const formData = new FormData();
        formData.append('name', name);
        formData.append('audio_file', audioFile);
        if (languageCode.trim()) formData.append('language_code', languageCode);
        if (groundTruthText.trim()) formData.append('ground_truth_text', groundTruthText);
        
        // Backend expects tags as a JSON array string, e.g., ["short", "noisy"]
        if (tags.trim()) {
            const tagsArray = tags.split(',').map(tag => tag.trim()).filter(tag => tag);
            if (tagsArray.length > 0) {
                 formData.append('tags', JSON.stringify(tagsArray));
            }
        }
       
        if (description.trim()) formData.append('description', description);

        try {
            await createASRTestCase(formData);
            setSuccessMessage('Test case created successfully! Redirecting to list...');
            setTimeout(() => {
                router.push('/admin/asr-test-cases');
            }, 2000);
        } catch (err: any) {
            setError(err.message || 'Failed to create test case.');
        } finally {
            setIsSubmitting(false);
        }
    };

    return (
        <div className="container mx-auto p-4">
            <div className="flex justify-between items-center mb-4">
                <h1 className="text-2xl font-bold">Create New ASR Test Case</h1>
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
                        id="name"
                        value={name}
                        onChange={(e) => setName(e.target.value)}
                        className="mt-1 block w-full p-2 border border-gray-600 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500 bg-gray-700 text-white"
                        required
                    />
                </div>

                <div>
                    <label htmlFor="audioFile" className="block text-sm font-medium text-gray-300">Audio File <span className="text-red-500">*</span></label>
                    <input
                        type="file"
                        id="audioFile"
                        onChange={handleFileChange}
                        accept="audio/*"
                        className="mt-1 block w-full text-sm text-gray-400 file:mr-4 file:py-2 file:px-4 file:rounded-full file:border-0 file:text-sm file:font-semibold file:bg-indigo-50 file:text-indigo-700 hover:file:bg-indigo-100"
                        required
                    />
                </div>

                <div>
                    <label htmlFor="languageCode" className="block text-sm font-medium text-gray-300">Language Code (e.g., en-US)</label>
                    <input
                        type="text"
                        id="languageCode"
                        value={languageCode}
                        onChange={(e) => setLanguageCode(e.target.value)}
                        className="mt-1 block w-full p-2 border border-gray-600 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500 bg-gray-700 text-white"
                    />
                </div>

                <div>
                    <label htmlFor="groundTruthText" className="block text-sm font-medium text-gray-300">Ground Truth Text</label>
                    <textarea
                        id="groundTruthText"
                        value={groundTruthText}
                        onChange={(e) => setGroundTruthText(e.target.value)}
                        rows={3}
                        className="mt-1 block w-full p-2 border border-gray-600 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500 bg-gray-700 text-white"
                    />
                </div>

                <div>
                    <label htmlFor="tags" className="block text-sm font-medium text-gray-300">Tags (comma-separated, e.g., short,noisy)</label>
                    <input
                        type="text"
                        id="tags"
                        value={tags}
                        onChange={(e) => setTags(e.target.value)}
                        className="mt-1 block w-full p-2 border border-gray-600 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500 bg-gray-700 text-white"
                    />
                </div>
                
                <div>
                    <label htmlFor="description" className="block text-sm font-medium text-gray-300">Description</label>
                    <textarea
                        id="description"
                        value={description}
                        onChange={(e) => setDescription(e.target.value)}
                        rows={3}
                        className="mt-1 block w-full p-2 border border-gray-600 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500 bg-gray-700 text-white"
                    />
                </div>

                <div>
                    <button
                        type="submit"
                        disabled={isSubmitting}
                        className="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50"
                    >
                        {isSubmitting ? 'Creating...' : 'Create Test Case'}
                    </button>
                </div>
            </form>
        </div>
    );
}
