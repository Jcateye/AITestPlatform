'use client';

import { useEffect, useState, FormEvent } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { ASRTestCase, listASRTestCases } from '@/lib/asrTestCaseService';
import { VendorConfig, listVendorConfigs } from '@/lib/vendorConfigService';
import { EvaluationJobPayload, createASRJob } from '@/lib/jobService';

export default function NewASRJobPage() {
    const router = useRouter();

    const [jobName, setJobName] = useState('');
    const [selectedTestCaseIds, setSelectedTestCaseIds] = useState<string[]>([]);
    const [selectedVendorConfigIds, setSelectedVendorConfigIds] = useState<string[]>([]);
    const [parameters, setParameters] = useState(''); // JSON string

    const [allTestCases, setAllTestCases] = useState<ASRTestCase[]>([]);
    const [allVendorConfigs, setAllVendorConfigs] = useState<VendorConfig[]>([]);

    const [isLoadingTestCases, setIsLoadingTestCases] = useState(true);
    const [isLoadingVendors, setIsLoadingVendors] = useState(true);
    const [isSubmitting, setIsSubmitting] = useState(false);

    const [error, setError] = useState<string | null>(null);
    const [successMessage, setSuccessMessage] = useState<string | null>(null);

    useEffect(() => {
        setIsLoadingTestCases(true);
        listASRTestCases()
            .then(data => setAllTestCases(data))
            .catch(err => setError('Failed to load ASR test cases: ' + err.message))
            .finally(() => setIsLoadingTestCases(false));

        setIsLoadingVendors(true);
        listVendorConfigs({ api_type: 'ASR' })
            .then(data => setAllVendorConfigs(data))
            .catch(err => setError('Failed to load ASR vendor configurations: ' + err.message))
            .finally(() => setIsLoadingVendors(false));
    }, []);

    const handleMultiSelectChange = (setter: React.Dispatch<React.SetStateAction<string[]>>) => (e: React.ChangeEvent<HTMLSelectElement>) => {
        const options = e.target.options;
        const value: string[] = [];
        for (let i = 0, l = options.length; i < l; i++) {
            if (options[i].selected) {
                value.push(options[i].value);
            }
        }
        setter(value);
    };

    const handleSubmit = async (event: FormEvent) => {
        event.preventDefault();
        setError(null);
        setSuccessMessage(null);

        if (selectedTestCaseIds.length === 0) {
            setError('At least one ASR Test Case must be selected.');
            return;
        }
        if (selectedVendorConfigIds.length === 0) {
            setError('At least one ASR Vendor Configuration must be selected.');
            return;
        }

        let parsedParameters: Record<string, any> | null = null;
        if (parameters.trim()) {
            try {
                parsedParameters = JSON.parse(parameters.trim());
            } catch (e) {
                setError('Parameters field contains invalid JSON.');
                return;
            }
        }

        setIsSubmitting(true);

        const payload: EvaluationJobPayload = {
            job_name: jobName.trim() || undefined, // Send undefined if empty, so backend sql.NullString works
            test_case_ids: selectedTestCaseIds.map(id => parseInt(id, 10)),
            vendor_config_ids: selectedVendorConfigIds.map(id => parseInt(id, 10)),
            parameters: parsedParameters,
        };

        try {
            const newJob = await createASRJob(payload);
            setSuccessMessage(`Job "${newJob.job_name || newJob.id}" created successfully (Status: ${newJob.status}). Redirecting to jobs list...`);
            setTimeout(() => {
                router.push('/admin/jobs');
            }, 3000);
        } catch (err: any) {
            setError(err.message || 'Failed to create ASR job.');
        } finally {
            setIsSubmitting(false);
        }
    };
    
    const commonSelectClasses = "mt-1 block w-full p-2 border border-gray-600 rounded-md shadow-sm bg-gray-700 text-white focus:ring-indigo-500 focus:border-indigo-500";

    return (
        <div className="container mx-auto p-4">
            <div className="flex justify-between items-center mb-6">
                <h1 className="text-2xl font-bold">Create New ASR Evaluation Job</h1>
                <Link href="/admin/jobs" className="bg-gray-500 hover:bg-gray-700 text-white font-bold py-2 px-4 rounded">
                    Back to Jobs List
                </Link>
            </div>

            {error && <p className="text-red-500 bg-red-100 p-3 rounded mb-4">{error}</p>}
            {successMessage && <p className="text-green-500 bg-green-100 p-3 rounded mb-4">{successMessage}</p>}

            <form onSubmit={handleSubmit} className="space-y-6 bg-gray-800 p-6 rounded-lg shadow-xl">
                <div>
                    <label htmlFor="jobName" className="block text-sm font-medium text-gray-300">Job Name (Optional)</label>
                    <input
                        type="text"
                        id="jobName"
                        value={jobName}
                        onChange={(e) => setJobName(e.target.value)}
                        className="mt-1 block w-full p-2 border border-gray-600 rounded-md shadow-sm bg-gray-700 text-white focus:ring-indigo-500 focus:border-indigo-500"
                    />
                </div>

                <div>
                    <label htmlFor="selectedTestCaseIds" className="block text-sm font-medium text-gray-300">ASR Test Cases <span className="text-red-500">*</span></label>
                    {isLoadingTestCases ? <p className="text-gray-400">Loading test cases...</p> :
                        <select
                            multiple
                            id="selectedTestCaseIds"
                            value={selectedTestCaseIds}
                            onChange={handleMultiSelectChange(setSelectedTestCaseIds)}
                            className={`${commonSelectClasses} h-40`}
                            required
                        >
                            {allTestCases.map(tc => (
                                <option key={tc.id} value={tc.id.toString()}>{tc.name} (ID: {tc.id})</option>
                            ))}
                        </select>
                    }
                     {allTestCases.length === 0 && !isLoadingTestCases && <p className="text-sm text-yellow-400 mt-1">No ASR test cases found. Please create some first.</p>}
                </div>

                <div>
                    <label htmlFor="selectedVendorConfigIds" className="block text-sm font-medium text-gray-300">ASR Vendor Configurations <span className="text-red-500">*</span></label>
                    {isLoadingVendors ? <p className="text-gray-400">Loading vendor configurations...</p> :
                        <select
                            multiple
                            id="selectedVendorConfigIds"
                            value={selectedVendorConfigIds}
                            onChange={handleMultiSelectChange(setSelectedVendorConfigIds)}
                            className={`${commonSelectClasses} h-32`}
                            required
                        >
                            {allVendorConfigs.map(vc => (
                                <option key={vc.id} value={vc.id.toString()}>{vc.name} (ID: {vc.id})</option>
                            ))}
                        </select>
                    }
                    {allVendorConfigs.length === 0 && !isLoadingVendors && <p className="text-sm text-yellow-400 mt-1">No ASR vendor configurations found. Please create some first.</p>}
                </div>

                <div>
                    <label htmlFor="parameters" className="block text-sm font-medium text-gray-300">Parameters (JSON string, Optional)</label>
                    <textarea
                        id="parameters"
                        value={parameters}
                        onChange={(e) => setParameters(e.target.value)}
                        rows={4}
                        placeholder='e.g., {"use_enhanced_model": true, "punctuation_level": "high"}'
                        className="mt-1 block w-full p-2 border border-gray-600 rounded-md shadow-sm bg-gray-700 text-white focus:ring-indigo-500 focus:border-indigo-500"
                    />
                </div>

                <div>
                    <button
                        type="submit"
                        disabled={isSubmitting || isLoadingTestCases || isLoadingVendors || allTestCases.length === 0 || allVendorConfigs.length === 0}
                        className="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50"
                    >
                        {isSubmitting ? 'Creating and Running Job...' : 'Create and Run ASR Job'}
                    </button>
                </div>
            </form>
        </div>
    );
}
