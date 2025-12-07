# Architecture Documentation

## Overview

Go-based Personal Assistant API with AI-powered conversation and tool capabilities. Uses OpenAI GPT models, MongoDB for persistence, and implements clean architecture following Go best practices.

## System Architecture

```
HTTP/Twirp API (cmd/server)
        ↓
Chat Server (internal/chat/server.go)
   ├─→ Repository (MongoDB) - Conversation persistence
   └─→ Assistant (AI) - Title/Reply generation + Tool dispatch
            ↓
       Tools Package
       ├─→ Weather (forecast/current via WeatherAPI.com)
       ├─→ Date/Time (RFC3339)
       ├─→ Holidays (ICS calendar)
       └─→ TimeZone (convert times between zones)
```

## Key Components

### 1. Chat Server (`internal/chat/server.go`)
**Operations:**
- `StartConversation` - Creates conversation, generates title/reply **concurrently** (50% faster)
- `DescribeConversation` - Retrieves by ID

### 2. Assistant (`internal/chat/assistant/`)
**Architecture:** Functional options pattern for dependency injection

```go
// Default
assistant := assistant.New()

// With mocks (testing)
assistant := assistant.New(
    assistant.WithOpenAIClient(mockClient),
    assistant.WithWeatherClient(mockWeatherClient),
)
```

**Structure:**
```
assistant/
├── assistant.go        # Orchestrator with options pattern
├── tools/             # AI tool adapters
│   ├── tools.go       # Interface + dispatch
│   ├── weather.go
│   ├── date.go
│   ├── holidays.go
│   └── timezone.go
└── weather/           # HTTP client (reusable, no import cycle)
    ├── client.go
    └── types.go
```

### 3. Tools Interface
```go
type Tool interface {
    Name() string
    Definition() openai.ChatCompletionToolUnionParam
    Handle(ctx context.Context, args string) (string, error)
}
```

**Design:**
- Interface-based for testability
- Simple O(n) dispatch (fine for 4 tools)
- Each tool in separate file

### 4. Weather Package
**Why separate from tools?**
- Eliminates import cycle
- Reusable outside AI context
- Clear separation: HTTP client vs tool adapter

## Data Flow Examples

### StartConversation
```
Request → Validate → Create Conversation
                          ↓
        ┌─────────────────┴─────────────────┐
        │     Concurrent (sync.WaitGroup)   │
        ├─→ Generate Title (GPT-4o)         │
        └─→ Generate Reply (GPT-4.1)        │
        └─────────────────┬─────────────────┘
                          ↓
              Update & Return Response
```

### Tool Call
```
User: "Weather in Barcelona?"
  ↓
GPT-4.1 calls get_weather tool
  ↓
Dispatch finds WeatherTool → Handle()
  ↓
Parse args → Call weather.Client → Format
  ↓
Return to GPT → Natural language response
```

## Design Patterns

### Functional Options (DI)
```go
type Option func(*Assistant)

func WithWeatherClient(c *weather.Client) Option {
    return func(a *Assistant) { a.weatherClient = c }
}

func New(opts ...Option) *Assistant {
    a := &Assistant{
        cli: openai.NewClient(),
        weatherClient: weather.NewClient(),
    }
    for _, opt := range opts { opt(a) }
    // ... initialize tools
    return a
}
```

**Benefits:** Testable, backward compatible, idiomatic Go

### Concurrent Operations
```go
var wg sync.WaitGroup
wg.Add(2)

go func() {
    defer wg.Done()
    title, _ = a.Title(ctx, conv)
}()

go func() {
    defer wg.Done()
    reply, _ = a.Reply(ctx, conv)
}()

wg.Wait()
```

**Benefits:** ~50% faster, graceful degradation

## Testing

**Strategy:**
- Unit tests with mocks (~0.5s per package)
- Integration tests with real MongoDB
- Test fixtures for clean setup/teardown

**Coverage:** 57 tests across all packages ✅

**Pattern:**
```go
func TestServer_StartConversation(t *testing.T) {
    t.Run("test name", WithFixture(func(t *testing.T, f *Fixture) {
        // Use f.Repository, f.CreateConversation()
        // Automatic cleanup
    }))
}
```

## Configuration

```bash
# Required
export OPENAI_API_KEY=sk-...
export WEATHER_API_KEY=...
export MONGO_URI=mongodb://localhost:27017

# Optional
export HOLIDAY_CALENDAR_LINK=https://...
```

## Adding a New Tool

1. **Create** `internal/chat/assistant/tools/mytool.go`:
```go
type MyTool struct{}

func NewMyTool() *MyTool { return &MyTool{} }
func (t *MyTool) Name() string { return "my_tool" }
func (t *MyTool) Definition() openai.ChatCompletionToolUnionParam { /* ... */ }
func (t *MyTool) Handle(ctx context.Context, args string) (string, error) { /* ... */ }
```

2. **Register** in `assistant.go`:
```go
a.tools = []tools.Tool{
    tools.NewWeatherTool(a.weatherClient),
    tools.NewMyTool(), // ← Add here
}
```

3. **Test** in `mytool_test.go`

## Technology Stack

| Layer | Tech | Purpose |
|-------|------|---------|
| API | Twirp | Protobuf RPC over HTTP |
| Language | Go 1.24.1 | Backend |
| Database | MongoDB | Persistence |
| AI | OpenAI GPT-4.1/4o | Reply/Title generation |
| Weather | WeatherAPI.com | Real-time data |
| Observability | OpenTelemetry | Metrics + Tracing |
| Testing | stdlib | Unit/integration |

## Observability

### Metrics (OpenTelemetry)
The application captures three key metrics via stdout exporter:

**1. Request Count** (`http.server.requests`)
- Counter tracking total HTTP requests
- Labels: `http.method`, `http.path`, `http.status_code`
- Use: Monitor traffic patterns, endpoint usage

**2. Request Duration** (`http.server.duration`)
- Histogram of request latency in milliseconds
- Labels: `http.method`, `http.path`, `http.status_code`
- Use: Identify slow endpoints, SLA monitoring

**3. Error Count** (`http.server.errors`)
- Counter for HTTP errors (status >= 400)
- Labels: `http.method`, `http.path`, `http.status_code`
- Use: Error rate tracking, alert on spikes

### Tracing (OpenTelemetry)
Distributed tracing captures request flow:
- **HTTP span**: Full request lifecycle with method, path, status
- **Context propagation**: Traces flow through assistant → tools → external APIs
- **Error marking**: Spans marked as errors when status >= 400
- **Stdout export**: Trace data printed to console for development

### Configuration
```go
// Initialize telemetry on startup
shutdown, _ := telemetry.InitTelemetry(ctx)
defer shutdown(ctx)

// Metrics exported every 30 seconds
// Traces batched and exported on shutdown
```

### Middleware Stack
```
Request → TracingMiddleware (create span)
       → MetricsMiddleware (record metrics)
       → Logger (existing)
       → Recovery (existing)
       → Handler
```

## Performance

- Title/Reply: ~3-5s (concurrent)
- Tool calls: +1-2s each
- DB ops: <100ms (local)

## Key Decisions

1. **Weather as separate package** - Avoids import cycles, increases reusability
2. **Functional options** - Enables testing with mocks, idiomatic Go
3. **Simple dispatch** - O(n) is fine for 4 tools, no premature optimization
4. **Concurrent title/reply** - 50% performance gain with sync.WaitGroup
5. **Interface-based tools** - Easy to test, extend, and maintain
6. **Stdout telemetry exporters** - Simple development setup, easy to switch to Prometheus/Jaeger later
7. **Graceful shutdown** - Ensures metrics/traces are flushed before exit

## Development Workflow

### Running Server
```bash
# Start MongoDB
make up

# Set environment
export OPENAI_API_KEY=sk-...
export WEATHER_API_KEY=...

# Run with telemetry
make run

# Metrics export every 30s to stdout
# Traces export on shutdown
```

## References

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)

