# MCP Architecture & Flow

```text
AI Agent / LLM Client (Cursor, Claude, Antigravity)
          │ (JSON-RPC 2.0 stdio)
          ▼
GCP-Calculator-MCP Server (mcp/server.go)
          │
          ▼
Calculator Engine (calculator/)
          │
          ├─────────────────────────┐
          ▼                         ▼
Catalog & Rates           Cost Calculation
(pricing.json, catalog.json) (Compute, GKE, Storage, SQL, Run, BigQuery)
```
