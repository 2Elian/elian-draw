# Go Backend for next-ai-draw-io

Go backend replacement for the Next.js API routes, using Gin framework.

## Quick Start

```bash
# 1. Copy environment configuration
cp .env.example .env
# Edit .env with your API keys

# 2. Build and run
make run

# 3. Or run directly with environment variables
PORT=3001 AI_PROVIDER=openai AI_MODEL=gpt-4o go run ./cmd/server/
```

## Architecture

```
├── cmd/server/main.go          # Entry point, Gin router
├── internal/
│   ├── config/config.go        # Environment + ai-models.json loading
│   ├── handler/
│   │   ├── chat.go             # POST /api/chat (core)
│   │   ├── middleware.go       # CORS, access-code, error recovery
│   │   └── misc.go             # Other API endpoints
│   ├── agent/
│   │   ├── loop.go             # Custom tool-calling loop
│   │   ├── tools.go            # Tool definitions (4 tools)
│   │   ├── prompt.go           # System prompt assembly
│   │   └── repair.go           # JSON repair for truncated LLM output
│   ├── provider/
│   │   ├── factory.go          # Provider factory (20+ providers)
│   │   └── openai.go           # OpenAI-compatible API client
│   ├── sse/writer.go           # UIMessageStreamResponse SSE writer
│   ├── model/message.go        # Type definitions
│   └── util/cache.go           # Cached response lookup
├── docs/shape-libraries/       # Shape library docs (from original project)
└── Makefile
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/chat` | POST | Core AI chat with SSE streaming |
| `/api/config` | GET | Public app configuration |
| `/api/server-models` | GET | Server models from ai-models.json |
| `/api/validate-diagram` | POST | Diagram validation |
| `/api/validate-model` | POST | Model configuration validation |
| `/api/log-feedback` | POST | User feedback logging |
| `/api/log-save` | POST | Save event logging |
| `/api/parse-url` | POST | URL parsing |
| `/api/verify-access-code` | POST | Access code verification |
| `/health` | GET | Health check |

## Electron Integration

Set `GO_BACKEND_PATH` environment variable to the Go binary path. The Electron app will start it alongside the Next.js frontend and proxy `/api/*` requests via Next.js rewrites.

```bash
# Development: run Go backend separately
cd go-backend && make dev

# In another terminal, run Next.js with proxy
GO_BACKEND_URL=http://localhost:3001 npm run dev
```

## SSE Protocol

The backend uses Vercel AI SDK v6's UIMessageStreamResponse format:

```
data: {"type":"start","messageId":"msg-123"}
data: {"type":"start-step"}
data: {"type":"text-delta","id":"text-1","delta":"Hello"}
data: {"type":"tool-input-start","toolCallId":"call-1","toolName":"display_diagram"}
data: {"type":"tool-input-delta","toolCallId":"call-1","inputTextDelta":"<mxCell..."}
data: {"type":"tool-input-available","toolCallId":"call-1","toolName":"display_diagram","input":{"xml":"..."}}
data: {"type":"finish-step"}
data: {"type":"finish","finishReason":"stop","messageMetadata":{"totalTokens":150}}
data: [DONE]
```
