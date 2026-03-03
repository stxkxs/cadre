import { NextResponse } from 'next/server';

const builtInTemplates = [
  {
    id: 'research-assistant',
    name: 'Research Assistant',
    description: 'Multi-step research workflow with web search and summarization',
    category: 'Research',
    nodeData: {
      type: 'agent',
      data: {
        label: 'Research Assistant',
        provider: 'anthropic',
        model: 'claude-sonnet-4-6',
        systemPrompt: 'You are a thorough research assistant. Analyze the given topic and provide a comprehensive, well-structured summary with key findings.',
        temperature: 0.3,
        maxTokens: 4096,
      },
    },
    isBuiltIn: true,
  },
  {
    id: 'code-reviewer',
    name: 'Code Reviewer',
    description: 'Automated code review with best practice suggestions',
    category: 'Development',
    nodeData: {
      type: 'agent',
      data: {
        label: 'Code Reviewer',
        provider: 'anthropic',
        model: 'claude-sonnet-4-6',
        systemPrompt: 'You are an expert code reviewer. Review the given code for bugs, security issues, performance problems, and adherence to best practices. Provide specific, actionable feedback.',
        temperature: 0.2,
        maxTokens: 4096,
      },
    },
    isBuiltIn: true,
  },
  {
    id: 'content-writer',
    name: 'Content Writer',
    description: 'Generate polished content with tone and style control',
    category: 'Content',
    nodeData: {
      type: 'agent',
      data: {
        label: 'Content Writer',
        provider: 'openai',
        model: 'gpt-4o',
        systemPrompt: 'You are a professional content writer. Create engaging, well-structured content based on the given brief. Match the requested tone and style.',
        temperature: 0.7,
        maxTokens: 4096,
      },
    },
    isBuiltIn: true,
  },
  {
    id: 'data-analyzer',
    name: 'Data Analyzer',
    description: 'Analyze datasets and generate insights',
    category: 'Analysis',
    nodeData: {
      type: 'agent',
      data: {
        label: 'Data Analyzer',
        provider: 'anthropic',
        model: 'claude-sonnet-4-6',
        systemPrompt: 'You are a data analysis expert. Analyze the provided data, identify patterns, trends, and anomalies. Present your findings with clear explanations and statistical reasoning.',
        temperature: 0.2,
        maxTokens: 4096,
      },
    },
    isBuiltIn: true,
  },
  {
    id: 'fast-summarizer',
    name: 'Fast Summarizer',
    description: 'Quick text summarization using Groq for speed',
    category: 'Utility',
    nodeData: {
      type: 'agent',
      data: {
        label: 'Fast Summarizer',
        provider: 'groq',
        model: 'llama-3.3-70b-versatile',
        systemPrompt: 'Summarize the following text concisely, capturing the key points and main ideas.',
        temperature: 0.3,
        maxTokens: 1024,
      },
    },
    isBuiltIn: true,
  },
  {
    id: 'code-generator',
    name: 'Code Generator',
    description: 'Generate code using Claude Code terminal integration',
    category: 'Development',
    nodeData: {
      type: 'agent',
      data: {
        label: 'Code Generator',
        provider: 'claude-code',
        model: 'claude-code',
        systemPrompt: 'Generate clean, well-documented code based on the requirements.',
        temperature: 0.3,
        maxTokens: 8192,
      },
    },
    isBuiltIn: true,
  },
  {
    id: 'translator',
    name: 'Translator',
    description: 'Translate text between languages while preserving tone and meaning',
    category: 'Utility',
    nodeData: {
      type: 'agent',
      data: {
        label: 'Translator',
        provider: 'openai',
        model: 'gpt-4o-mini',
        systemPrompt: 'You are an expert translator. Translate the given text accurately while preserving tone, nuance, and cultural context. If the target language is not specified, translate to English.',
        temperature: 0.3,
        maxTokens: 4096,
      },
    },
    isBuiltIn: true,
  },
  {
    id: 'email-drafter',
    name: 'Email Drafter',
    description: 'Draft professional emails with appropriate tone',
    category: 'Content',
    nodeData: {
      type: 'agent',
      data: {
        label: 'Email Drafter',
        provider: 'anthropic',
        model: 'claude-haiku-4-5',
        systemPrompt: 'You are a professional email writer. Draft clear, concise emails based on the given context. Match the appropriate level of formality.',
        temperature: 0.5,
        maxTokens: 2048,
      },
    },
    isBuiltIn: true,
  },
  {
    id: 'test-generator',
    name: 'Test Generator',
    description: 'Generate unit tests for code with edge case coverage',
    category: 'Development',
    nodeData: {
      type: 'agent',
      data: {
        label: 'Test Generator',
        provider: 'anthropic',
        model: 'claude-sonnet-4-6',
        systemPrompt: 'You are a testing expert. Generate comprehensive unit tests for the given code. Cover happy paths, edge cases, error scenarios, and boundary conditions.',
        temperature: 0.2,
        maxTokens: 4096,
      },
    },
    isBuiltIn: true,
  },
  {
    id: 'sentiment-analyzer',
    name: 'Sentiment Analyzer',
    description: 'Analyze text sentiment and emotional tone',
    category: 'Analysis',
    nodeData: {
      type: 'agent',
      data: {
        label: 'Sentiment Analyzer',
        provider: 'groq',
        model: 'llama-3.1-8b-instant',
        systemPrompt: 'Analyze the sentiment of the given text. Classify it as positive, negative, or neutral. Also identify the emotional tone (e.g., enthusiastic, frustrated, neutral). Output a JSON object with sentiment, confidence, and tone fields.',
        temperature: 0.1,
        maxTokens: 512,
      },
    },
    isBuiltIn: true,
  },
];

export async function GET() {
  return NextResponse.json(builtInTemplates);
}
