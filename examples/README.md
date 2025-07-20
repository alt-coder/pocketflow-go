# PocketFlow-Go Examples

This directory contains example applications demonstrating various PocketFlow-Go concepts and integrations.

## Available Examples

### Basic Chat Example (`basic-chat/`)
A simple terminal-based chat application that demonstrates:
- Three-phase execution model (Prep → Exec → Post)
- Self-looping node behavior
- LLM integration with Google GenAI
- Conversation state management

### LLM Utilities (`llm/`)
Generic LLM provider interfaces and implementations:
- Generic LLM provider interface
- Google Gemini client implementation
- Mock provider for testing
- Common message types and utilities

## Getting Started

Each example includes its own README with specific setup instructions. Generally, you'll need:

1. Go 1.23+ installed
2. Appropriate API keys (e.g., Google API key for Gemini)
3. Environment variables configured

## Running Examples

Navigate to the specific example directory and follow the README instructions. Most examples can be run with:

```bash
cd examples/basic-chat
go run .
```

## Contributing

When adding new examples:
1. Create a new directory under `examples/`
2. Include a comprehensive README
3. Add appropriate tests
4. Document the PocketFlow-Go concepts being demonstrated