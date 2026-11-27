<p align="center">
  <img src="web/static/img/yoomates-logo.png" alt="YooMates logo" width="130" />
</p>

<br>

# <div align="center"><strong>YooMates</strong></div>

> A unified, high-performance team operating platform built with Go, native C++ audio processing, WebAssembly client routines, and server-rendered hypermedia.

---

## 1. System Surfaces

YooMates consolidates real-time team communication, organization-wide publishing, and personal workspace management into a unified interface:

- **Chat Hubs (`/hubs`)**: Persistent communication channels, direct messaging, real-time presence, typing indicators, threaded reactions, multipart file transfers, and low-latency native voice lounges.
- **Global Feed (`/global`)**: Chronological organization-wide publishing pipeline supporting interactive polls, nested threaded comments, dynamic engagement tracking, and cursor-based infinite stream pagination.
- **My Space (`/myspace`)**: High-density workspace offering hierarchical document management, a local-first markdown editor with debounced state synchronization, interactive Kanban boards, and structured backlog tracking.
- **Universal Quick Switcher (`Ctrl+K`)**: Global keyboard-driven command palette for instant sub-millisecond navigation across channels, posts, documents, and system actions.

---

## 2. System Architecture

YooMates is structured around a monolithic Go server runtime coupled with a specialized C++ real-time audio daemon and client-side WebAssembly verification routines:

```text
+---------------------------------------------------------------------------------+
|                               Client Browser                                    |
|  (DOM Surface | HTMX Hypermedia | WebSockets | IndexedDB | WebAssembly | VAD)   |
+---------------------------------------------------------------------------------+
       |                                                 |
       | HTTP/1.1 & WebSocket Frames                     | RTP Audio Packets
       v                                                 v
+-----------------------------------+             +-------------------------------+
|       Go Core Monolith            |             |   C++ Real-Time Voice Engine  |
|          (`cmd/server`)           |             |       (`engine/audio`)        |
|                                   |             |                               |
|  |-- HTTP Router & Multiplexer    |             |  |-- Opus Frame Demuxer       |
|  |-- Argon2id Session Manager     |             |  |-- Dynamic Jitter Buffer    |
|  |-- Template Engine & OOB Swaps  |             |  |-- Energy-based VAD         |
|  |-- WebSocket Hub & Fan-out      |             |  |-- Comfort Noise Generator  |
|  |-- SQLite3 Engine (WAL Mode)    |             |  |-- Multi-Peer Audio Mixer   |
+-----------------------------------+             +-------------------------------+
       |                                                         |
       +------------------- Inter-Process IPC -------------------+
```

### Component Topology & Data Flow

```text
HTTP Request / WebSocket Frame
|
+-- [Edge / Ingress]
|   |-- TLS Termination & Static Asset Cache (Cache-Control: immutable)
|   |-- Session Authentication & CSRF Validation (Argon2id + HTTP-only Cookie)
|
+-- [Routing & Dispatch Layer]
|   |-- /hubs    --> Hub Handler --> WebSocket Client Hub (Goroutine Fanout)
|   |-- /global  --> Feed Handler --> SQL Pagination Query --> Template Fragment
|   |-- /myspace --> Space Handler --> Document Storage Engine --> Markdown Partial
|
+-- [Storage Subsystem]
|   |-- SQLite Database (Write-Ahead Logging enabled, Busy Timeout: 5000ms)
|   |-- Prepared statement caching across all query paths
|
+-- [Client Relay Subsystem]
    |-- HTMX: Swaps HTML partials directly into targeted DOM elements
    |-- WebSocket: Out-of-band (OOB) attribute updates for zero-redraw synchronization
    |-- IndexedDB: Local persistence ensuring 0ms cold-start document & message cache
    |-- WebAssembly: Client-side in-memory inverted search index for instant queries
```

---

## 3. Real-Time Communication & Voice Subsystems

### WebSocket Message Fan-Out Model

```text
User Event (Chat / Reaction / Typing)
|
+-- dumb-pipe.js event relay
    |
    v
WebSocket Connection (Goroutine writePump / readPump)
|
+-- Central Broadcast Hub (RWMutex protected client registry)
    |
    +-- JSON Event Validation & SQLite WAL persistence
    |
    +-- Concurrent Fan-out to Subscribed Room Channel Peers
        |
        v
    Targeted Out-of-Band (OOB) HTML Swap Fragment
        |
        v
    DOM Injected without Page Refresh or Virtual DOM Reconciliation
```

### C++ Native Audio Pipeline

```text
Microphone Input
|
+-- WebRTC / RTP Opus Packets
    |
    v
Voice Gateway Port (UDP / IPC)
|
+-- Opus Frame Header Parsing & Sequence Verification
|
+-- Adaptive Jitter Buffer (Target Delay: 20ms, Loss Concealment)
|
+-- Voice Activity Detection (Energy & Zero-Crossing Thresholding)
|
+-- Comfort Noise Generation (CNG on silence packet drops)
|
+-- 48kHz Fullband PCM Frame Output / Downmix
```

---

## 4. Performance Specifications & Telemetry

Empirical load test metrics recorded under sustained concurrent connections:

| Metric | Target Specification | Observed Benchmark |
|:---|:---|:---|
| **Cold Start Duration** | < 25 ms | ~ 12 ms |
| **P99 Message Broadcast (1k clients)** | < 2.0 ms | 0.84 ms |
| **Max Broadcast Throughput** | > 200,000 msgs/sec | ~ 412,000 msgs/sec |
| **Server Resident Memory (Base)** | < 15 MB | ~ 4.2 MB |
| **Voice Engine Processing Latency** | < 10 ms | ~ 3.1 ms |
| **Client Script Bundle Size** | < 10 KB | ~ 6.8 KB (unminified) |

*Detailed benchmark methodology, concurrency scenarios, and telemetry scripts are available in [BENCHMARKS.md](BENCHMARKS.md).*

---

## 5. Build, Verification & Execution

### Prerequisites

- **Go**: 1.22+
- **C++ Compiler**: GCC 11+ / Clang 13+ / MSVC 2022 supporting C++17
- **CMake**: 3.20+ (for building native audio engine)
- **Make** (optional, for automation tasks)

### Compilation

```bash
# Compile Go server monolith
go build -ldflags="-s -w" -o bin/yoomates.exe ./cmd/server

# Compile C++ native audio engine
mkdir -p engine/audio/build
cmake -S engine/audio -B engine/audio/build -DCMAKE_BUILD_TYPE=Release
cmake --build engine/audio/build --config Release
```

### Execution

```bash
# Start the YooMates server
./bin/yoomates.exe

# The server listens on http://localhost:8080 by default.
```

### Verification & Testing

```bash
# Run unit test suite across all modules
go test -v -count=1 ./...

# Run native C++ audio engine validation
./bin/yoomates-voice.exe --test

# Execute concurrent load and throughput benchmarks
go run cmd/benchmark/main.go
```

---

## 6. Global Keyboard Navigation

| Key Combination | Action Scope | Operation |
|:---|:---|:---|
| <kbd>Ctrl+K</kbd> / <kbd>Cmd+K</kbd> | Global | Open Universal Quick Switcher (Command Palette) |
| <kbd>Ctrl+F</kbd> / <kbd>Cmd+F</kbd> | Hubs | Focus client-side WebAssembly channel search |
| <kbd>Esc</kbd> | Global | Close active modal, quick switcher, or clear search filter |
| <kbd>↑</kbd> / <kbd>↓</kbd> | Quick Switcher | Navigate search items and execution actions |
| <kbd>Enter</kbd> | Quick Switcher | Dispatch selected navigation or action |

---

## 7. Governance, Contribution & License

- **Contribution Guidelines**: Please review [CONTRIBUTING.md](CONTRIBUTING.md) before opening pull requests. All changes must satisfy our test coverage, concurrency safety, and telemetry criteria.
- **License**: YooMates is proprietary software. Review [LICENSE](LICENSE) for terms of access and usage restrictions.

---
<div align="center">
  <strong>(c) 2026 Voxion Labs & Rudranarayan Jena</strong>
</div>
