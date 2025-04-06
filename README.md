# Model Context Protocol (MCP) - Go Implementation (WIP)

🚧 **Work in Progress** — This is an experimental implementation of the **Model Context Protocol (MCP)** in Go. It aims to provide a structured, extensible, and interoperable way to manage, stream, and persist model contexts across various AI/LLM runtimes.

---

## 🔍 Overview

The **Model Context Protocol (MCP)** defines a standard way to:

- Represent **contextual state** for models (e.g. LLM session state, memory, scratchpads).
- Serialize and deserialize context into a **portable format** (e.g. JSON, MessagePack, etc.).
- Define and stream **context updates** (e.g. `Add`, `Forget`, `Recall`, `Clear`, `Metadata`, etc.).
- Enable **modular backends** (file, memory, remote storage, event logs).

This Go library provides:

- 🧠 Core context model types (`Context`, `ContextUpdate`, `MemoryBlock`, etc.)
- 🔄 Encoders/decoders for standard formats (JSON, MsgPack planned)
- ⚙️ Streaming support via Go channels and/or gRPC (WIP)
- 💾 Storage backends (in-memory, file-based) — pluggable architecture
- 🛠️ Utilities for merging, pruning, chunking, and diffing context

---

## 📦 Installation

> Note: This package is **not yet stable**. API subject to change.

```bash
go get github.com/null-create/gomcp
```

---

## ✨ Quickstart

```go
import (
    mcp "github.com/null-create/gomcp"
)

func main() {
    ctx := mcp.NewContext("session-123")

    ctx.AddMemoryBlock(mcp.MemoryBlock{
        Role:    "user",
        Content: "How do I build a Go server?",
    })

    serialized, _ := ctx.ToJSON()
    fmt.Println(string(serialized))
}
```

---

## 🧱 Core Concepts

### `Context`

Represents a session or model runtime context. It may contain memory blocks, metadata, timestamps, etc.

### `MemoryBlock`

The atomic unit of context — can represent messages, instructions, files, summaries, etc.

### `ContextUpdate`

Represents a diff/change to a context. Useful for real-time or streaming updates.

---

## 🔧 Planned Features

- [ ] Streaming context updates over gRPC/WebSocket
- [ ] Context pruning and summarization
- [ ] Pluggable backend support (SQLite, Redis, S3)
- [ ] Context versioning and audit logs
- [ ] Schema validation (JSON Schema / Protobuf)
- [ ] Secure context signing + encryption

---

## 🧪 Project Status

This implementation is **early-stage** and under active development. Expect breaking changes as the protocol evolves.

Want to contribute or discuss ideas? Open an issue or start a discussion!

---

## 📂 Project Structure

```
/mcp
  /models        # Core structs (Context, MemoryBlock, Update)
  /client        # MCP client implementation
  /codec         # JSON/MsgPack (de)serialization
  /handlers      # Client and Server handlers
  /server        # MCP server implemnentation
  /backend       # Storage plugins
  /sse           #
  /stdio         #
  /examples      # Usage examples and CLI tools
```
