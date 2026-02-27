// Agent types
export interface Agent {
  name: string
  role: string
  goal: string
  backstory: string
  compact_backstory?: string
  tools: string[]
  memory: MemoryConfig
  quick_mode?: boolean
  provider?: string
  provider_model?: string
  api_key?: string
  work_dir?: string
}

// Provider types
export interface ProviderInfo {
  name: string
  label: string
  needs_key: boolean
}

export interface ClaudeCodeStatus {
  available: boolean
  path?: string
  error?: string
}

export interface MemoryConfig {
  type: string
  max_tokens: number
}

// Task types
export interface Task {
  name: string
  description: string
  agent: string
  inputs: TaskInput[]
  outputs: TaskOutput[]
  output_schema?: OutputSchema
  dependencies: string[]
  timeout: string
  retry: RetryConfig
}

export interface TaskInput {
  name: string
  type: string
  required: boolean
  default?: string
}

export interface TaskOutput {
  name: string
  type: string
}

export interface OutputSchema {
  format: string
  strict: boolean
  fields: OutputField[]
}

export interface OutputField {
  name: string
  type: string
  description: string
  required: boolean
}

export interface RetryConfig {
  max_attempts: number
  backoff: string
}

// Crew types
export interface Crew {
  name: string
  description: string
  agents: string[]
  process: 'sequential' | 'parallel' | 'hierarchical'
  concurrency?: number
  error_strategy?: string
  max_iterations?: number
  tasks: CrewTask[]
  manager?: string
}

export interface CrewTask {
  name: string
  agent: string
  depends_on?: string[]
  description?: string
  timeout?: string
  retry?: RetryConfig
  inputs?: TaskInput[]
  outputs?: TaskOutput[]
}

// Run types
export interface Run {
  id: string
  crew_name: string
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'
  started_at: string
  completed_at?: string
  error?: string
  tasks: TaskState[]
  inputs?: Record<string, unknown>
  outputs?: Record<string, unknown>
  metadata?: Record<string, unknown>
}

export interface TaskState {
  name: string
  agent: string
  status: 'pending' | 'running' | 'completed' | 'failed' | 'skipped'
  started_at?: string
  completed_at?: string
  error?: string
  attempts: number
  inputs?: Record<string, unknown>
  outputs?: Record<string, unknown>
}

// Tool types
export interface Tool {
  name: string
  description: string
  provider: string
}

// SSE event types
export interface SSEEvent {
  type: string
  timestamp: string
  run_id?: string
  data?: Record<string, unknown>
}

// Chat types
export interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
  timestamp: Date
}

// Validation result
export interface ValidationResult {
  valid: boolean
  errors: string[]
}

// Template types
export interface TemplateCategory {
  id: string
  label: string
  icon: string
}

export interface TemplateMeta {
  category: string
  complexity: 'beginner' | 'intermediate' | 'advanced'
}

export interface TemplateAgent {
  name: string
  role: string
  goal: string
  backstory: string
  tools: string[]
  memory: MemoryConfig
  meta: TemplateMeta
}

export interface TemplateTask {
  name: string
  description: string
  agent: string
  inputs: TaskInput[]
  outputs: TaskOutput[]
  dependencies: string[]
  timeout: string
  retry: RetryConfig
  meta: TemplateMeta
}

export interface TemplateCrew {
  name: string
  description: string
  agents: string[]
  process: string
  tasks: CrewTask[]
  meta: TemplateMeta
}

export interface ImportRequest {
  type: 'agent' | 'task' | 'crew'
  name: string
  overrides?: Record<string, unknown>
}

export interface OnboardingStatus {
  showOnboarding: boolean
  hasAgents: boolean
  hasTasks: boolean
  hasCrews: boolean
  hasRuns: boolean
}
