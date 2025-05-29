'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { EvaluationJob, listJobs } from '@/lib/jobService';

export default function ListJobsPage() {
    const [jobs, setJobs] = useState<EvaluationJob[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [jobTypeFilter, setJobTypeFilter] = useState(''); // e.g., "ASR"

    const fetchJobs = async () => {
        setIsLoading(true);
        setError(null);
        try {
            const params: { job_type?: string } = {};
            if (jobTypeFilter) {
                params.job_type = jobTypeFilter;
            }
            const data = await listJobs(params);
            setJobs(data);
        } catch (err: any) {
            setError(err.message || 'Failed to fetch jobs');
        } finally {
            setIsLoading(false);
        }
    };

    useEffect(() => {
        fetchJobs();
    }, []); // Initial fetch

    const handleFilterApply = () => {
        fetchJobs();
    };
    
    const handleClearFilter = () => {
        setJobTypeFilter('');
        // fetchJobs will be called by useEffect if jobTypeFilter was a dependency,
        // or call it manually if not. For explicit control:
        listJobs().then(data => setJobs(data)).catch(err => setError(err.message));
    };


    return (
        <div className="container mx-auto p-4">
            <div className="flex justify-between items-center mb-6">
                 <h1 className="text-2xl font-bold">Evaluation Jobs</h1>
                <Link href="/admin/jobs/asr/new" className="bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded">
                    Create New ASR Job
                </Link>
            </div>


            <div className="mb-4 flex space-x-2">
                <select
                    value={jobTypeFilter}
                    onChange={(e) => setJobTypeFilter(e.target.value)}
                    className="p-2 border rounded text-black bg-gray-200"
                >
                    <option value="">All Job Types</option>
                    <option value="ASR">ASR</option>
                    {/* Add other job types like TTS, LLM when available */}
                </select>
                <button
                    onClick={handleFilterApply}
                    className="bg-indigo-500 hover:bg-indigo-700 text-white font-bold py-2 px-4 rounded"
                >
                    Apply Filter
                </button>
                <button
                    onClick={handleClearFilter}
                    className="bg-gray-300 hover:bg-gray-400 text-black font-bold py-2 px-4 rounded"
                >
                    Clear Filter
                </button>
            </div>

            {isLoading && <p>Loading jobs...</p>}
            {error && <p className="text-red-500 bg-red-100 p-3 rounded">Error: {error}</p>}

            {!isLoading && !error && (
                <div className="overflow-x-auto shadow-md rounded-lg">
                    <table className="min-w-full table-auto border-collapse border border-gray-700 bg-gray-800 text-white">
                        <thead className="bg-gray-700">
                            <tr>
                                <th className="border border-gray-600 px-4 py-2">Job ID</th>
                                <th className="border border-gray-600 px-4 py-2">Job Name</th>
                                <th className="border border-gray-600 px-4 py-2">Type</th>
                                <th className="border border-gray-600 px-4 py-2">Status</th>
                                <th className="border border-gray-600 px-4 py-2">Created At</th>
                                <th className="border border-gray-600 px-4 py-2">Completed At</th>
                                <th className="border border-gray-600 px-4 py-2">Actions</th>
                            </tr>
                        </thead>
                        <tbody>
                            {jobs.map((job) => (
                                <tr key={job.id} className="hover:bg-gray-700">
                                    <td className="border border-gray-600 px-4 py-2 text-center">{job.id}</td>
                                    <td className="border border-gray-600 px-4 py-2">{job.job_name || 'N/A'}</td>
                                    <td className="border border-gray-600 px-4 py-2 text-center">{job.job_type}</td>
                                    <td className="border border-gray-600 px-4 py-2 text-center">
                                        <span className={`px-2 py-1 text-xs font-semibold rounded-full ${
                                            job.status === 'COMPLETED' ? 'bg-green-500 text-green-100' :
                                            job.status === 'RUNNING' ? 'bg-yellow-500 text-yellow-100' :
                                            job.status === 'PENDING' ? 'bg-blue-500 text-blue-100' :
                                            job.status === 'FAILED' ? 'bg-red-500 text-red-100' :
                                            'bg-gray-500 text-gray-100'
                                        }`}>
                                            {job.status}
                                        </span>
                                    </td>
                                    <td className="border border-gray-600 px-4 py-2 text-center">
                                        {new Date(job.created_at).toLocaleString()}
                                    </td>
                                     <td className="border border-gray-600 px-4 py-2 text-center">
                                        {job.completed_at ? new Date(job.completed_at).toLocaleString() : 'N/A'}
                                    </td>
                                    <td className="border border-gray-600 px-4 py-2 text-center">
                                        <Link href={`/admin/jobs/${job.id}`} className="text-indigo-400 hover:underline">
                                            View Details
                                        </Link>
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            )}
             {!isLoading && !error && jobs.length === 0 && (
                <p className="mt-4 text-gray-400">No jobs found matching the criteria.</p>
            )}
        </div>
    );
}
