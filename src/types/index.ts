/**
 * Core type definitions for DrogonClaw
 * Comprehensive TypeScript interfaces for all system components
 */

import { z } from 'zod';

// ============================================================================
// Configuration Types
// ============================================================================

export const AIProviderSchema = z.enum(['claude', 'openai', 'ollama', 'local']);
export type AIProvider = z.infer<typeof AIProviderSchema>;

export const ConfigSchema = z.object({
  aiProvider: AIProviderSchema.default('claude'),
  anthropicApiKey: z.string().optional(),
  openaiApiKey: z.string().optional(),
  ollamaUrl: z.string().optional().default('http://localhost:11434'),
  ollamaModel: z.string().optional().default('mistral'),
  gatewayPort: z.number().default(18789),
  gatewayHost: z.string().default('localhost'),
  databasePath: z.string().default('./data/drogonclaw.db'),
  logLevel: z.enum(['debug', 'info', 'warn', 'error']).default('info'),
  sessionTimeout: z.number().default(3600000), // 1 hour
  maxToolTimeout: z.number().default(300000), // 5 minutes
  toolWhitelist: z.array(z.string()).default([
    'nmap',
    'netstat',
    'curl',
    'wget',
    'dig',
    'whois',
    'traceroute',
  ]),
  telegramToken: z.string().optional(),
  telegramChatId: z.string().optional(),
  enableWebSocket: z.boolean().default(true),
  enableTelegram: z.boolean().default(false),
});

export type Config = z.infer<typeof ConfigSchema>;

// ============================================================================
// Session Types
// ============================================================================

export const SessionStatusSchema = z.enum([
  'pending',
  'active',
  'paused',
  'completed',
  'failed',
]);
export type SessionStatus = z.infer<typeof SessionStatusSchema>;

export interface Session {
  id: string;
  target: string;
  strategy: string;
  status: SessionStatus;
  startTime: number;
  endTime?: number;
  findings: Finding[];
  metadata: Record<string, unknown>;
  notes: string;
  createdAt: number;
  updatedAt: number;
}

// ============================================================================
// Finding Types
// ============================================================================

export const SeveritySchema = z.enum(['critical', 'high', 'medium', 'low', 'info']);
export type Severity = z.infer<typeof SeveritySchema>;

export interface Finding {
  id: string;
  sessionId: string;
  title: string;
  description: string;
  severity: Severity;
  type: string; // 'port_open', 'service_detected', 'vulnerability', 'info', etc.
  target: string;
  evidence: string[];
  remediation?: string;
  references: string[];
  discoveredAt: number;
  toolUsed?: string;
}

// ============================================================================
// Tool Types
// ============================================================================

export const ToolStatusSchema = z.enum(['pending', 'running', 'completed', 'failed']);
export type ToolStatus = z.infer<typeof ToolStatusSchema>;

export interface ToolExecution {
  id: string;
  sessionId: string;
  toolName: string;
  args: string[];
  status: ToolStatus;
  output: string;
  error?: string;
  startTime: number;
  endTime?: number;
  duration?: number;
}

export interface ToolDefinition {
  name: string;
  description: string;
  category: string;
  args: ToolArgument[];
  timeout: number;
}

export interface ToolArgument {
  name: string;
  description: string;
  required: boolean;
  type: 'string' | 'number' | 'boolean' | 'array';
}

// ============================================================================
// Skill Types
// ============================================================================

export interface Skill {
  id: string;
  name: string;
  description: string;
  category: string;
  priority: number;
  tools: string[];
  preconditions?: string[];
  instructions: string;
  expectedOutputs: string[];
  version: string;
  author: string;
}

// ============================================================================
// AI Model Types
// ============================================================================

export interface ModelMessage {
  role: 'user' | 'assistant' | 'system';
  content: string;
}

export interface ModelResponse {
  content: string;
  toolCalls: ToolCall[];
  stopReason: string;
}

export interface ToolCall {
  id: string;
  name: string;
  args: Record<string, unknown>;
}

// ============================================================================
// Agent Loop Types
// ============================================================================

export interface AgentContext {
  session: Session;
  config: Config;
  messages: ModelMessage[];
  systemPrompt: string;
  skillRegistry: Skill[];
  toolDefinitions: ToolDefinition[];
}

export interface AgentIteration {
  iterationNumber: number;
  timestamp: number;
  thinkingTime?: number;
  modelUsed: string;
  response: ModelResponse;
  toolExecutions: ToolExecution[];
  findingsDiscovered: Finding[];
  nextAction: 'continue' | 'pause' | 'complete' | 'error';
}

// ============================================================================
// API Types
// ============================================================================

export interface ApiRequest {
  method: string;
  path: string;
  params?: Record<string, unknown>;
  body?: Record<string, unknown>;
}

export interface ApiResponse<T = unknown> {
  success: boolean;
  data?: T;
  error?: string;
  timestamp: number;
}

export interface WebSocketMessage {
  type:
    | 'session_update'
    | 'finding_discovered'
    | 'tool_executed'
    | 'agent_thinking'
    | 'agent_complete'
    | 'error';
  sessionId: string;
  payload: unknown;
  timestamp: number;
}

// ============================================================================
// Report Types
// ============================================================================

export interface Report {
  sessionId: string;
  target: string;
  strategy: string;
  duration: number;
  startTime: number;
  endTime: number;
  summary: string;
  findings: Finding[];
  findings_by_severity: Record<string, Finding[]>;
  recommendations: string[];
  generatedAt: number;
}

// ============================================================================
// Error Types
// ============================================================================

export class DrogonError extends Error {
  constructor(
    public code: string,
    message: string,
    public details?: Record<string, unknown>,
  ) {
    super(message);
    this.name = 'DrogonError';
  }
}

export class ConfigError extends DrogonError {
  constructor(message: string, details?: Record<string, unknown>) {
    super('CONFIG_ERROR', message, details);
    this.name = 'ConfigError';
  }
}

export class ModelError extends DrogonError {
  constructor(message: string, details?: Record<string, unknown>) {
    super('MODEL_ERROR', message, details);
    this.name = 'ModelError';
  }
}

export class ToolError extends DrogonError {
  constructor(message: string, details?: Record<string, unknown>) {
    super('TOOL_ERROR', message, details);
    this.name = 'ToolError';
  }
}

export class StorageError extends DrogonError {
  constructor(message: string, details?: Record<string, unknown>) {
    super('STORAGE_ERROR', message, details);
    this.name = 'StorageError';
  }
}

// ============================================================================
// Strategy Types
// ============================================================================

export type Strategy = 'light' | 'thorough' | 'deep' | 'aggressive';

export interface StrategyDefinition {
  name: Strategy;
  description: string;
  skillIds: string[];
  timeout: number;
  agentThinkingLevel: 'minimal' | 'normal' | 'extended';
}

// ============================================================================
// Logger Types
// ============================================================================

export interface LogEntry {
  level: 'debug' | 'info' | 'warn' | 'error';
  message: string;
  timestamp: number;
  context?: Record<string, unknown>;
  sessionId?: string;
}
