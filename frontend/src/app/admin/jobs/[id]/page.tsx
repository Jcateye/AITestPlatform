'use client';

import { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import { EvaluationJob, ASREvaluationResult, getJob, getJobResults } from '@/lib/jobService';
import { ASRTestCase, listASRTestCases } from '@/lib/asrTestCaseService';
import { VendorConfig, listVendorConfigs } from '@/lib/vendorConfigService';

interface EnrichedASREvaluationResult extends ASREvaluationResult {
    testCaseName?: string;
    vendorName?: string;
    groundTruth?: string | null;
}

export default function JobDetailsPage() {
    const params = useParams();
    const id = params.id as string;
    const router = useRouter();

    const [job, setJob] = useState<EvaluationJob | null>(null);
    const [results, setResults] = useState<EnrichedASREvaluationResult[]>([]);
    const [allTestCases, setAllTestCases] = useState<ASRTestCase[]>([]);
    const [allVendorConfigs, setAllVendorConfigs] = useState<VendorConfig[]>([]);
    
    const [isLoadingJob, setIsLoadingJob] = useState(true);
    const [isLoadingResults, setIsLoadingResults] = useState(true);
    const [isLoadingAuxData, setIsLoadingAuxData] = useState(true); // For test cases and vendors
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        if (id) {
            setIsLoadingJob(true);
            getJob(id)
                .then(data => setJob(data))
                .catch(err => setError(err.message || 'Failed to fetch job details.'))
                .finally(() => setIsLoadingJob(false));

            setIsLoadingResults(true);
            getJobResults(id)
                .then(data => setResults(data as EnrichedASREvaluationResult[])) // Initial cast
                .catch(err => setError(err.message || 'Failed to fetch job results.'))
                .finally(() => setIsLoadingResults(false));
            
            setIsLoadingAuxData(true);
            Promise.all([
                listASRTestCases(),
                listVendorConfigs({ api_type: 'ASR' }) // Assuming ASR job for now
            ]).then(([tcData, vcData]) => {
                setAllTestCases(tcData);
                setAllVendorConfigs(vcData);
            }).catch(err => {
                setError((prevError) => (prevError ? prevError + "; " : "") + "Failed to fetch auxiliary data (test cases/vendors): " + err.message);
            }).finally(() => setIsLoadingAuxData(false));
        }
    }, [id]);

    // Enrich results once all data is loaded
    useEffect(() => {
        if (results.length > 0 && allTestCases.length > 0 && allVendorConfigs.length > 0) {
            setResults(prevResults => prevResults.map(res => {
                const testCase = allTestCases.find(tc => tc.id === res.asr_test_case_id);
                const vendorConfig = allVendorConfigs.find(vc => vc.id === res.vendor_config_id);
                return {
                    ...res,
                    testCaseName: testCase?.name || `ID: ${res.asr_test_case_id}`,
                    vendorName: vendorConfig?.name || `ID: ${res.vendor_config_id}`,
                    groundTruth: testCase?.ground_truth_text,
                };
            }));
        }
    }, [results.length, allTestCases.length, allVendorConfigs.length]); // Dependencies to re-trigger enrichment

    const isLoading = isLoadingJob || isLoadingResults || isLoadingAuxData;

    if (isLoading) {
        return <p className="container mx-auto p-4">Loading job details and results...</p>;
    }
    if (error && !job) { // If job fetch failed, show error prominently
        return <p className="container mx-auto p-4 text-red-500 bg-red-100 p-3 rounded">Error: {error}</p>;
    }
    if (!job) {
        return <p className="container mx-auto p-4">Job not found.</p>;
    }

    return (
        <div className="container mx-auto p-4">
            <div className="flex justify-between items-center mb-6">
                <h1 className="text-3xl font-bold">Job Details: {job.job_name || `Job ID ${job.id}`}</h1>
                <Link href="/admin/jobs" className="bg-gray-500 hover:bg-gray-700 text-white font-bold py-2 px-4 rounded">
                    Back to Jobs List
                </Link>
            </div>
            
            {error && <p className="text-red-500 bg-red-100 p-3 rounded mb-4">Error fetching some data: {error}</p>}


            <div className="bg-gray-800 shadow-xl rounded-lg p-6 mb-8 text-white">
                <h2 className="text-xl font-semibold mb-4 border-b border-gray-700 pb-2">Job Summary</h2>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <p><strong>Job ID:</strong> {job.id}</p>
                    <p><strong>Name:</strong> {job.job_name || 'N/A'}</p>
                    <p><strong>Type:</strong> {job.job_type}</p>
                    <p><strong>Status:</strong> <span className={`px-2 py-1 text-sm font-semibold rounded-full ${
                                            job.status === 'COMPLETED' ? 'bg-green-600 text-green-100' :
                                            job.status === 'RUNNING' ? 'bg-yellow-600 text-yellow-100' :
                                            job.status === 'PENDING' ? 'bg-blue-600 text-blue-100' :
                                            job.status === 'FAILED' ? 'bg-red-600 text-red-100' :
                                            'bg-gray-600 text-gray-100'
                                        }`}>{job.status}</span></p>
                    <p><strong>Created At:</strong> {new Date(job.created_at).toLocaleString()}</p>
                    <p><strong>Started At:</strong> {job.started_at ? new Date(job.started_at).toLocaleString() : 'N/A'}</p>
                    <p><strong>Completed At:</strong> {job.completed_at ? new Date(job.completed_at).toLocaleString() : 'N/A'}</p>
                    <p><strong>Test Case IDs:</strong> {job.test_case_ids.join(', ')}</p>
                    <p><strong>Vendor Config IDs:</strong> {job.vendor_config_ids.join(', ')}</p>
                    {job.parameters && <p className="md:col-span-2"><strong>Parameters:</strong> <pre className="bg-gray-700 p-2 rounded mt-1 text-sm">{JSON.stringify(job.parameters, null, 2)}</pre></p>}
                </div>
            </div>

            <h2 className="text-2xl font-bold mb-4">Evaluation Results</h2>
            {isLoadingResults && <p>Loading results...</p>}
            {!isLoadingResults && results.length === 0 && <p className="text-gray-400">No evaluation results found for this job.</p>}
            {!isLoadingResults && results.length > 0 && (
                <div className="overflow-x-auto shadow-md rounded-lg">
                    <table className="min-w-full table-auto border-collapse border border-gray-700 bg-gray-800 text-white">
                        <thead className="bg-gray-700">
                            <tr>
                                <th className="border border-gray-600 px-3 py-2">Test Case</th>
                                <th className="border border-gray-600 px-3 py-2">Vendor</th>
                                <th className="border border-gray-600 px-3 py-2">Ground Truth</th>
                                <th className="border border-gray-600 px-3 py-2">Recognized Text</th>
                                <th className="border border-gray-600 px-3 py-2 text-center">CER</th>
                                <th className="border border-gray-600 px-3 py-2 text-center">WER</th>
                                <th className="border border-gray-600 px-3 py-2 text-center">Latency (ms)</th>
                            </tr>
                        </thead>
                        <tbody>
                            {results.map((res) => (
                                <tr key={res.id} className="hover:bg-gray-700">
                                    <td className="border border-gray-600 px-3 py-2">{res.testCaseName}</td>
                                    <td className="border border-gray-600 px-3 py-2">{res.vendorName}</td>
                                    <td className="border border-gray-600 px-3 py-2 text-sm">{res.groundTruth || 'N/A'}</td>
                                    <td className="border border-gray-600 px-3 py-2 text-sm">{res.recognized_text || 'N/A'}</td>
                                    <td className="border border-gray-600 px-3 py-2 text-center">{res.cer?.toFixed(3) ?? 'N/A'}</td>
                                    <td className="border border-gray-600 px-3 py-2 text-center">{res.wer?.toFixed(3) ?? 'N/A'}</td>
                                    <td className="border border-gray-600 px-3 py-2 text-center">{res.latency_ms ?? 'N/A'}</td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            )}
        </div>
    );
}
