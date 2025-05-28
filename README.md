# AITestPlatform

Unified AI Evaluation Platform Architecture and Feature Design
I. Core Concepts and Objectives
Standardization: Establish unified test case formats, evaluation workflows, parameter definitions, and metric calculation methods to ensure comparability конкуре́нтность between different vendors and models.

Automation: Support batch testing tasks, automatically call vendor APIs, and automatically collect and process evaluation results, reducing manual intervention.

Scalability & Extensibility: The system design should be easy to scale horizontally to handle a large number of test tasks and facilitate future integration of new AI components, new vendors, or new evaluation dimensions.

Usability: Provide an intuitive and friendly web interface for efficient use by testers, product managers, operations personnel, and other roles.

Visualization: Clearly and multi-dimensionally display evaluation results, supporting graphical comparative analysis to help quickly identify the strengths and weaknesses of various models.

Configurability: Allow users to flexibly configure the detailed parameters of each component to simulate real-world business scenario calls.

II. System Architecture
graph TD
    subgraph "User Layer"
        WebApp[Frontend Web Application (Next.js)]
    end

    subgraph "Application Layer (Golang)"
        APIGateway[API Gateway / BFF]
        AuthSimplified[Simplified Admin Access]
        JobManagement[Job Management Module]
        ConfigManagement[Configuration Management Module (Vendors/Parameters/Test Cases)]
    end

    subgraph "Core Engine Layer (Golang)"
        MQ[Message Queue (e.g., RabbitMQ, Kafka)]
        JobScheduler[Job Scheduler]
        EvaluationEngine[Evaluation Execution Engine]
        VendorAdapters[Vendor Adapter Module]
        MetricsCalculator[Metrics Calculation Module]
    end

    subgraph "Data Storage Layer"
        Database[Relational Database (PostgreSQL/MySQL)]
        ObjectStorage[Object Storage (MinIO/S3)]
        Cache[Cache (Redis) - Optional]
    end

    subgraph "3rd Party Services"
        ASR_Google[Google Cloud Speech-to-Text]
        ASR_MS[Microsoft Azure Speech Service]
        ASR_Deepgram[Deepgram ASR]
        ASR_Tencent[Tencent Cloud Speech Recognition]
        ASR_Ali[Alibaba Cloud Intelligent Speech Interaction]
        ASR_Volc[Volcengine Speech Recognition]

        TTS_11Labs[ElevenLabs TTS]
        TTS_Cartesia[Cartesia TTS]
        TTS_MS_TTS[Microsoft Azure Speech Service TTS]
        TTS_Google_TTS[Google Cloud Text-to-Speech]
        TTS_Ali_TTS[Alibaba Cloud Intelligent Speech Interaction TTS]

        LLM_Azure_GPT4o[Azure OpenAI GPT-4o]
        LLM_Azure_GPT4omini[Azure OpenAI GPT-4o-mini]
    end

    %% Connections
    WebApp -- HTTP/WebSocket --> APIGateway
    APIGateway -- Authenticated Access --> JobManagement
    APIGateway -- Authenticated Access --> ConfigManagement

    JobManagement -- Create/Update Jobs --> Database
    JobManagement -- Push Job ID --> MQ
    ConfigManagement -- CRUD --> Database

    JobScheduler -- Listen --> MQ
    JobScheduler -- Get Job Details --> JobManagement
    JobScheduler -- Dispatch Job --> EvaluationEngine

    EvaluationEngine -- Call --> VendorAdapters
    EvaluationEngine -- Get Raw Results --> VendorAdapters
    EvaluationEngine -- Call --> MetricsCalculator
    EvaluationEngine -- Store Evaluation Data/Results --> Database
    EvaluationEngine -- Store Audio Files --> ObjectStorage
    EvaluationEngine -- Update Job Status --> JobManagement

    VendorAdapters -- API Call --> ASR_Google
    VendorAdapters -- API Call --> ASR_MS
    VendorAdapters -- API Call --> ASR_Deepgram
    VendorAdapters -- API Call --> ASR_Tencent
    VendorAdapters -- API Call --> ASR_Ali
    VendorAdapters -- API Call --> ASR_Volc

    VendorAdapters -- API Call --> TTS_11Labs
    VendorAdapters -- API Call --> TTS_Cartesia
    VendorAdapters -- API Call --> TTS_MS_TTS
    VendorAdapters -- API Call --> TTS_Google_TTS
    VendorAdapters -- API Call --> TTS_Ali_TTS

    VendorAdapters -- API Call --> LLM_Azure_GPT4o
    VendorAdapters -- API Call --> LLM_Azure_GPT4omini

    MetricsCalculator -- Calculate Metrics --> EvaluationEngine

Component Description:

Frontend Web Application (Next.js):

User interface for creating and managing test jobs, configuring vendors and parameters, managing test cases, visualizing evaluation results.

API Gateway / BFF (Golang):

Unified entry point for frontend-backend interaction, responsible for request routing, protocol conversion, aggregating backend service responses, initial parameter validation, and handling admin access.

Simplified Admin Access (Golang):

Handles access control based on a pre-configured super administrator account. No user registration or complex roles.

Job Management Module (Golang):

Manages the lifecycle of test jobs (creation, queuing, executing, completed, failed).

Provides job querying, retrying, and cancellation operations.

Configuration Management Module (Golang):

Manages various configuration information in the system:

Vendor Configuration: API Keys, Secrets, Endpoints, supported model lists, etc., for each AI service provider.

Parameter Templates: Predefined or custom common parameter combinations for each vendor's components.

Test Case Library: Stores and manages test cases for ASR, TTS, and LLM, supporting classification by language, scenario, etc.

Language Management: Maintains the list of languages supported for evaluation.

Message Queue (e.g., RabbitMQ, Kafka):

Used for asynchronous processing of evaluation jobs, decoupling job submission and actual execution, improving system concurrency and stability.

Job Scheduler (Golang):

Retrieves pending evaluation jobs from the message queue.

Schedules and dispatches jobs to instances of the Evaluation Execution Engine based on priority, system load, etc.

Evaluation Execution Engine (Golang):

Core processing unit responsible for executing specific evaluation jobs.

Calls corresponding third-party AI services via Vendor Adapters based on job configuration.

Invokes the Metrics Calculation Module for quantitative assessment after obtaining raw output.

Stores evaluation results, raw data, and intermediate files in the database and object storage.

Vendor Adapter Module (Golang):

Encapsulates unified calling interfaces for each integrated third-party AI service (e.g., Google ASR, Azure OpenAI).

Handles authentication, parameter mapping, request construction, response parsing, and error handling for each vendor's API. This is key to system extensibility.

Metrics Calculation Module (Golang):

Implements corresponding evaluation metric calculation logic based on the characteristics of different components (ASR, TTS, LLM).

Relational Database (PostgreSQL/MySQL):

Stores structured data, such as vendor configurations, test case metadata, job information, evaluation results (metrics), parameter templates.

Object Storage (MinIO/S3):

Stores unstructured data, such as raw audio files for ASR testing, TTS-generated audio files, and complete LLM conversation histories (if archiving is needed).

Cache (Redis - Optional):

Used to cache hot data, such as frequently used vendor configurations and test cases, to improve access speed.

III. Feature Breakdown
1. General and Basic Functional Modules
Simplified User Access:

No user registration system.

Access controlled by a single, pre-configured super administrator account (e.g., credentials set in environment variables or a configuration file).

All functionalities are available to the administrator.

Vendor and Model Management:

Vendor Management: CRUD (Create, Read, Update, Delete) for integrated AI service vendor information.

Authentication Configuration: Securely store and manage API Keys, Secrets, Endpoints, etc., for each vendor.

Model Management: Maintain a list of specific models provided by each vendor (e.g., gpt-4o, gpt-4o-mini) and their characteristics.

Parameter Definition: Define configurable parameters for each model of each vendor, including their value ranges, default values, and descriptions.

Test Case Management:

Multi-component Support: Separately manage test cases for ASR, TTS, and LLM.

Case Format:

ASR: Audio file + ground truth text.

TTS: Input text + (optional) desired audio feature description/reference audio.

LLM: Prompt + (optional) desired response pattern/ground truth/evaluation points.

Batch Operations: Support batch import/export of test cases (CSV, JSON, Excel, etc.).

Versioning: (Optional) Version control for test case sets.

Tagging and Classification: Classify and tag test cases by language, business scenario, difficulty, etc.

Parameter Template Management:

Allow users to create and save common parameter combination templates for specific models of specific vendors.

Templates can be quickly selected when creating test jobs.

Language Management:

Maintain the list of languages supported for evaluation (e.g., zh-CN, en-US, ja-JP).

Test cases and evaluation jobs need to be associated with a language.

Dashboard:

Display system overview: job execution statistics, average performance trends for each component, performance of frequently used vendors, etc.

2. ASR (Speech Recognition) Evaluation Module
Test Job Creation:

Select one or more ASR vendors/models for comparative testing.

Select the test language.

Select a test case set or upload new test audio files (supports wav, mp3, pcm, etc., batch upload).

Associate or input ground truth text for each audio file.

Detailed Parameter Configuration:

Model selection (e.g., Google's latest_long, Microsoft's specific model version).

Enable/disable punctuation recognition.

Number formatting options.

Hotword/phrase list enhancement (Phrase Hints / Custom Vocabulary).

Domain-specific model options.

Other vendor-specific parameters.

Evaluation Execution and Results:

The system automatically distributes audio to selected ASR service providers.

Obtain recognized text results.

Core Evaluation Metrics:

Character Error Rate (CER): For character-based languages like Chinese.

Word Error Rate (WER): For word-based languages like English.

(Optional) Sentence Error Rate (SER).

Latency: Time from request submission to receiving the complete result.

(Optional) Throughput: Audio duration processed per unit of time.

Results Display and Analysis:

Clearly list the ground truth, recognition results from each vendor, various error rates, and latency for each audio file.

Support sorting, filtering, and grouping by vendor, language, error rate, etc.

Highlight differences between recognition results and ground truth (diff view).

Error type analysis (insertion, deletion, substitution).

Supported Vendors: Google, Microsoft, Deepgram, Tencent, Alibaba, Volcengine.

3. TTS (Text-to-Speech) Evaluation Module
Test Job Creation:

Select one or more TTS vendors/models for comparative testing.

Select the test language.

Input or batch upload text test cases for synthesis.

Detailed Parameter Configuration:

Voice/Speaker: Select different voice characteristics (gender, age, style).

Speed/Rate.

Pitch.

Volume.

Sample Rate: e.g., 16kHz, 24kHz.

Audio Format: e.g., MP3, WAV, PCM, Opus.

Emotion/Style: e.g., happy, sad, customer service, news anchor.

SSML (Speech Synthesis Markup Language) support.

Other vendor-specific parameters (e.g., 11labs voice cloning parameters, Cartesia low-latency parameters).

Evaluation Execution and Results:

The system automatically distributes text to selected TTS service providers.

Obtain generated audio files.

Core Evaluation Metrics:

Subjective Evaluation (MOS - Mean Opinion Score):

The platform provides an online audio playback feature for listening.

Invite testers to rate synthesized audio on multiple dimensions such as naturalness, clarity, emotional expression accuracy, and prosody (typically on a 1-5 scale).

The system records and calculates the average score and confidence interval.

Objective Evaluation (partially automatable):

Synthesis Latency: Time from request to receiving the first audio packet/complete audio.

Audio File Size.

(Advanced) Audio feature analysis: e.g., fundamental frequency, formants, spectrogram comparison (with reference audio or ideal model, relatively complex).

(Advanced) Prosody consistency, naturalness of pauses.

Results Display and Analysis:

List each text case, synthesized audio from each vendor (playable online), MOS scores, latency, file size, etc.

Support sorting, filtering, and grouping by vendor, language, speaker, MOS score, etc.

Provide a subjective evaluation scoring interface and statistical functions.

Supported Vendors: 11Labs, Cartesia, Microsoft, Google, Alibaba Cloud.

4. LLM (Large Language Model) Evaluation Module
Test Job Creation:

Select one or more LLM models for comparative testing (e.g., Azure OpenAI GPT-4o, Azure OpenAI GPT-4o-mini).

Select the test language.

Input or batch upload test Prompts (instructions/questions/dialogue context).

(Optional) Associate each Prompt with:

Ground Truth or reference answers.

Desired output characteristics (e.g., concise, detailed, specific format).

Evaluation dimensions to be assessed.

Detailed Parameter Configuration:

temperature

max_tokens (maximum output length)

top_p

presence_penalty

frequency_penalty

System Prompt

Dialogue history (Context) management method.

Function Calling / Tool Use related parameters (if the model supports and needs testing).

Evaluation Execution and Results:

The system automatically distributes Prompts (and relevant context and parameters) to selected LLM services.

Obtain LLM response text.

Core Evaluation Metrics (LLM evaluation is complex and multi-dimensional, usually combining automated and manual methods):

Objective Metrics (if ground truth is available or quantifiable):

Exact Match (EM): Suitable for scenarios with fixed and short answers.

BLEU, ROUGE, METEOR: Commonly used for tasks like translation and summarization, measuring overlap with reference answers.

F1 Score, Precision, Recall: Commonly used for tasks like classification and information extraction.

Pass@k for code generation tasks.

Manual Evaluation/Subjective Scoring:

Relevance: Is the answer on-topic?

Accuracy: Is the information correct?

Fluency: Is the language natural and smooth?

Coherence: Is the logic clear and contextually consistent?

Completeness: Does it answer all aspects of the question?

Conciseness: Is it brief and to the point, without redundant information?

Helpfulness: Is the answer practically helpful to the user?

Safety: Does it contain harmful, biased, or inappropriate content?

Instruction Following: Did it follow the Prompt's instructions?

Creativity: (Specific scenarios)

The platform provides a manual scoring interface, supporting custom scoring dimensions and standards.

Model-based Evaluation (Advanced):

Use another (usually more powerful or specially trained) LLM to evaluate the output quality of the target LLM.

Response Latency: Time to first token, Time to complete generation.

Token Usage / Cost.

Results Display and Analysis:

List each Prompt, response text from each model, manual scores (various dimensions), automatically calculated objective metrics, latency, cost, etc.

Support side-by-side comparison of outputs from different models.

Support sorting, filtering, and grouping by model, language, Prompt category, score, etc.

Provide detailed scoring records and statistics.

Supported Vendors/Models: Azure OpenAI GPT-4o, Azure OpenAI GPT-4o-mini.

5. Results Analysis and Reporting Module
Comparative Analysis:

Support horizontal and vertical comparison across vendors, models, parameter configurations, and test case sets.

Graphical display (bar charts, line charts, radar charts, scatter plots, box plots, etc.) of evaluation results.

E.g., WER comparison of different ASR vendors for a specific language; MOS score comparison of different speakers for the same TTS model; accuracy and manual score comparison of different LLMs on specific task types.

Historical Trend Analysis:

Track performance changes of specific vendors/models at different time points and versions.

Report Generation and Export:

Support summarizing evaluation job results, comparative analysis charts, etc., into evaluation reports.

Support exporting reports in PDF, Excel, HTML, etc., formats.

Custom report templates.

Results Sharing:

(Optional) Support sharing evaluation results or reports with team members via links.

IV. Technology Stack (As specified by you)
Backend Development Language: Golang

Recommended Frameworks: Gin (Web framework), gRPC (Internal service communication)

Database Driver: database/sql + specific database driver (e.g., pq for PostgreSQL, go-sql-driver/mysql for MySQL)

Frontend Framework: Next.js (React)

UI Libraries: Material UI, Ant Design, Tailwind CSS (choose as needed)

State Management: Zustand, Redux Toolkit, React Context

Data Fetching: SWR, React Query

Database: PostgreSQL (powerful, suitable for complex queries and data analysis) or MySQL.

Message Queue: RabbitMQ (comprehensive features, supports multiple message patterns) or Kafka (high throughput, suitable for large-scale data streams). For job queue scenarios, RabbitMQ is often easier to set up and manage.

Object Storage: MinIO (open-source, self-hosted, S3 compatible) or cloud provider's object storage (e.g., AWS S3, Azure Blob Storage, Google Cloud Storage).

V. Development Iteration Suggestions
MVP (Minimum Viable Product) Stage:

Core Functionality: First, implement the complete evaluation workflow for one component (e.g., ASR).

Simplified admin login (pre-configured account).

Manual configuration of 1-2 ASR vendors.

Support uploading a small number of audio files and ground truths.

Manual triggering of evaluation jobs.

Backend synchronous execution of evaluations (no message queue or complex scheduling initially).

Basic CER/WER calculation.

Simple list-based results display.

Goal: Quickly validate the core workflow and gather early user feedback.

Iteration 1: Refine Basic Infrastructure and Core Component Support

Introduce a message queue for asynchronous job processing.

Improve the job management module and scheduler.

Gradually integrate all specified ASR vendors.

Implement the core evaluation workflow for the TTS component (parameter configuration, MOS subjective scoring interface, synthesis latency).

Integrate 1-2 TTS vendors.

Optimize frontend interaction and add basic chart displays.

Iteration 2: LLM Evaluation and Advanced Features

Implement the core evaluation workflow for the LLM component (parameter configuration, manual scoring interface, basic objective metrics like EM).

Integrate the specified LLM models.

Enhance test case management (batch import/export, classification tags).

Develop the parameter template feature.

Improve results analysis and comparison capabilities.

Iteration 3: Platformization and Experience Optimization

Enhance the dashboard.

Implement report generation and export.

Performance optimization and stability improvements.

Continuously optimize UI/UX based on user feedback.

Continuous Evolution:

Explore more advanced evaluation metrics and automated assessment methods (e.g., LLM-assisted evaluation).

Support more AI components and vendors.

Integrate CI/CD pipelines for automated testing and deployment.

This plan provides a relatively comprehensive blueprint. In actual execution, adjustments can be made based on team resources and business priorities. Good luck with your project!
