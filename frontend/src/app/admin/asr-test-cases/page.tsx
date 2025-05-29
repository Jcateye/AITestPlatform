'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation'; // If using for navigation after delete/create
import { ASRTestCase, listASRTestCases, deleteASRTestCase } from '@/lib/asrTestCaseService';

export default function ASRTestCasesPage() {
    const [testCases, setTestCases] = useState<ASRTestCase[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const router = useRouter();

    // Filter states
    const [languageFilter, setLanguageFilter] = useState('');
    const [tagsFilter, setTagsFilter] = useState('');

    const fetchTestCases = async () => {
        setIsLoading(true);
        setError(null);
        try {
            const params: { language_code?: string; tags?: string } = {};
            if (languageFilter) params.language_code = languageFilter;
            if (tagsFilter) params.tags = tagsFilter; // Assuming backend expects comma-separated string

            const data = await listASRTestCases(params);
            setTestCases(data);
        } catch (err: any) {
            setError(err.message || 'Failed to fetch test cases');
        } finally {
            setIsLoading(false);
        }
    };

    useEffect(() => {
        fetchTestCases();
    }, []); // Initial fetch

    const handleApplyFilters = () => {
        fetchTestCases();
    };
    
    const handleDelete = async (id: string) => {
        if (window.confirm('Are you sure you want to delete this test case?')) {
            try {
                await deleteASRTestCase(id);
                // Refetch or remove from local state
                setTestCases(prevTestCases => prevTestCases.filter(tc => tc.id.toString() !== id));
                alert('Test case deleted successfully.');
            } catch (err: any) {
                setError(err.message || 'Failed to delete test case');
                alert(`Error: ${err.message || 'Failed to delete test case'}`);
            }
        }
    };

    return (
        <div className="container mx-auto p-4">
            <h1 className="text-2xl font-bold mb-4">ASR Test Cases</h1>

            <div className="mb-4 flex space-x-2">
                <input
                    type="text"
                    placeholder="Language Code (e.g., en-US)"
                    value={languageFilter}
                    onChange={(e) => setLanguageFilter(e.target.value)}
                    className="p-2 border rounded text-black"
                />
                <input
                    type="text"
                    placeholder="Tags (comma-separated, e.g., noisy,short)"
                    value={tagsFilter}
                    onChange={(e) => setTagsFilter(e.target.value)}
                    className="p-2 border rounded text-black"
                />
                <button
                    onClick={handleApplyFilters}
                    className="bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded"
                >
                    Apply Filters
                </button>
                 <button
                    onClick={() => {
                        setLanguageFilter('');
                        setTagsFilter('');
                        fetchTestCases(); // Or pass empty strings to fetchTestCases if it handles reset
                    }}
                    className="bg-gray-300 hover:bg-gray-400 text-black font-bold py-2 px-4 rounded"
                >
                    Clear Filters
                </button>
            </div>


            <Link href="/admin/asr-test-cases/new" className="bg-green-500 hover:bg-green-700 text-white font-bold py-2 px-4 rounded mb-4 inline-block">
                Create New Test Case
            </Link>

            {isLoading && <p>Loading test cases...</p>}
            {error && <p className="text-red-500">Error: {error}</p>}

            {!isLoading && !error && (
                <table className="min-w-full table-auto border-collapse border border-slate-400">
                    <thead>
                        <tr>
                            <th className="border border-slate-300 px-4 py-2">Name</th>
                            <th className="border border-slate-300 px-4 py-2">Language</th>
                            <th className="border border-slate-300 px-4 py-2">Tags</th>
                            <th className="border border-slate-300 px-4 py-2">Created At</th>
                            <th className="border border-slate-300 px-4 py-2">Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        {testCases.map((tc) => (
                            <tr key={tc.id}>
                                <td className="border border-slate-300 px-4 py-2">{tc.name}</td>
                                <td className="border border-slate-300 px-4 py-2">{tc.language_code || 'N/A'}</td>
                                <td className="border border-slate-300 px-4 py-2">
                                    {tc.tags && tc.tags.length > 0 ? tc.tags.join(', ') : 'None'}
                                </td>
                                <td className="border border-slate-300 px-4 py-2">{new Date(tc.created_at).toLocaleDateString()}</td>
                                <td className="border border-slate-300 px-4 py-2">
                                    <Link href={`/admin/asr-test-cases/edit/${tc.id}`} className="text-blue-500 hover:underline mr-2">
                                        Edit
                                    </Link>
                                    <button
                                        onClick={() => handleDelete(tc.id.toString())}
                                        className="text-red-500 hover:underline"
                                    >
                                        Delete
                                    </button>
                                </td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            )}
             {!isLoading && !error && testCases.length === 0 && (
                <p className="mt-4">No test cases found.</p>
            )}
        </div>
    );
}
