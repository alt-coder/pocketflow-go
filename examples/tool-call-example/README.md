# PocketFlow-Go: Tool-Call Example

This example demonstrates how to build an interactive AI agent using the PocketFlow-Go framework. The agent engages in a conversation with the user and can decide to use external tools to fulfill requests. A key feature of this example is the user-centric permission model, where the agent explicitly asks for approval before executing any tool.

## Features

- **Interactive Chat**: A command-line interface for continuous conversation with the agent.
- **LLM-Powered Tool-Calling**: Leverages a Gemini LLM to intelligently decide when to use tools based on the user's request.
- **Structured Communication**: The agent and LLM communicate using a strict YAML format, ensuring reliable parsing of intents and tool commands.
- **User Permission Model**: Before executing a tool, the agent prompts the user for permission:
  - `y`: Yes, allow this single use.
  - `n`: No, do not use the tool for this request.
  - `a`: Always allow, and don't ask for permission for this specific tool again in the current session.
- **External Tool Management**: Uses the Model-Context-Protocol (MCP) to manage and interact with external tools (e.g., filesystem access, web search).
- **Flexible Configuration**: Configure the agent, LLM, and tools via a `config.json` file or environment variables.

## How it Works

The application is built around a `Workflow` from the PocketFlow-Go framework. This workflow contains a single, self-looping `ChatNode` that manages the entire agent interaction. The `ChatNode` implements the core `Prep`, `Exec`, and `Post` methods, which define the agent's lifecycle for each turn.

1.  **Initialization (`main.go`)**: The application starts by loading the configuration, initializing the Google Gemini LLM provider, and setting up the `MCPManager` which manages the lifecycle of external tool servers.

2.  **Workflow and State**:
    - A `Workflow` is created containing the central `ChatNode`.
    - The `AgentState` (`state.go`) is initialized to hold the conversation history.

3.  **The `ChatNode` Execution Cycle (`chat_node.go`)**: The workflow runs the `ChatNode` in a loop. Each iteration follows the `Prep -> Exec -> Post` pattern:

    -   **`Prep` (Prepare)**: This method sets the stage for the LLM call. It
        - Reads the current conversation history from the `AgentState`.
        - Prompts the user for new input from the command line.
        - Constructs a detailed system prompt, dynamically including the schemas of all available tools to let the LLM know what it can do.
        - Bundles the history and new input into a `ChatContext` object.

    -   **`Exec` (Execute)**: This is the simplest step. It takes the `ChatContext` from `Prep` and makes the API call to the Gemini LLM. It returns the raw response from the model.

    -   **`Post` (Process)**: This method handles the LLM's response and orchestrates the next steps. It
        - Parses the LLM's YAML-formatted response to extract the user-facing `response` and any requested `tool_calls`.
        - If tools were requested, it calls `AskToolPermission` to prompt the user for approval (`y/n/a`).
        - For each approved tool, it uses the `ToolManager` to execute the command via MCP.
        - It formats the tool results as a new message and adds both the LLM's text response and the tool results to the `AgentState`.
        - Finally, it returns a `core.Action` (e.g., `ActionSuccess`) to the workflow, which causes the loop to continue to the next `Prep` phase.
## Configuration

Create a `config.json` file in the `examples/tool-call-example` directory.

**1. API Keys:**
You need a Google AI API key for the Gemini LLM. You can set it in one of two ways:
- **(Recommended)** In `config.json`: