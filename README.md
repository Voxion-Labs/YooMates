<p align="center">
  <img src="web/static/img/yoomates-logo.png" alt="YooMates logo" width="180" />
</p>

<br>

# <div align="center"><strong>YooMates</strong></div>

> **The Engineered Minimalism Operating Surface for Modern Engineering Teams**  
> Powered by pure **Go (Golang)**, native **C++ low-latency audio**, **WebAssembly**, and **Go Templates + HTMX**.  
> **Zero `node_modules` | Zero TypeScript runtime overhead | <15ms cold start | Sub-millisecond p99 dispatch.**

---

## 1. Product Overview & Pillars

YooMates unifies team communication, organization-wide publishing, and personal execution into a single, cohesive operating surface:

- **Chat Hubs (`/hubs`)**:
  High-throughput real-time team collaboration with channels, direct messages, emoji reactions, threaded message drawers, typing indicators, multipart file attachments, and real-time C++ WebRTC/RTP voice lounges.
- **Global Feed (`/global`)**:
  Organization-wide chronological publishing stream with rich posts, interactive polls, nested comments, like tracking, and HTMX cursor-based infinite scroll.
- **My Space (`/myspace`)**:
  High-density personal execution workspace with nested document hierarchies, built-in Markdown editor with local-first 500ms debounced autosave, interactive Kanban boards, and compact list views.

---

## 2. Core Architecture & Engineering Principles

YooMates was migrated from a legacy Next.js / Node.js / Prisma prototype to a high-performance, single-binary architecture:

1. **Zero Runtime `node_modules` & Zero TypeScript**:
   - The entire backend runs as a single, statically compiled Go binary (<25MB) with zero external runtime dependencies.
2. **JavaScript as a "Dumb Pipe" (<220 Lines)**:
   - Client-side JavaScript (`web/static/js/dumb-pipe.js`) has **zero business logic** and **zero state management**.
   - Acts purely as an event relay: passing WebSocket frames, triggering WASM routines, audio VAD, and handling the global `Ctrl+K` quick switcher.
3. **Hypermedia & Server-Driven UI (Go `html/template` + HTMX)**:
   - Server-side pre-rendered HTML partials with <5ms TTFB.
   - Real-time DOM updates via WebSocket streams (`hx-ext="ws"`), out-of-band swaps, and optimistic updates.
4. **Client-Side Resilience & WebAssembly**:
   - Service Worker (`sw.js`) and IndexedDB (`idb-cache.js`) for 0ms instant offline message loading.
   - Pure Go compiled WebAssembly search engine (`search.wasm`) for instant, client-side message filtering.
5. **C++ Low-Latency Audio Engine (`engine/audio/`)**:
   - RFC 3550 RTP packet router, RFC 6716 Opus TOC parser, atomic spinlocks, adaptive jitter buffering with packet loss concealment (PLC), and comfort noise generation (CNG).
6. **Raw MySQL 8.4 (InnoDB) Persistence**:
   - Raw SQL migrations (`migrations/0001_initial_schema.sql`) utilizing ACID transactions, foreign keys, and indexes without ORM overhead.

---

## 3. Repository Structure

```text
├── 📂 cmd/
│   ├── 📂 server/             # Primary Go HTTP + WebSocket server entrypoint
│   ├── 📂 wasm/               # Go source compiled to web/static/wasm/search.wasm
│   └── 📂 benchmark/          # High-concurrency load benchmark CLI suite
├── 📂 engine/
│   └── 📂 audio/              # High-performance C++ RTP / Voice processing engine
│       ├── 📂 include/        # C++ headers (RTP, Jitter Buffer, VAD, Transcoder, CNG)
│       └── 📂 src/            # C++ implementation & verification suite
├── 📂 internal/
│   ├── 📂 auth/               # Argon2id password hashing, session tokens & middleware
│   ├── 📂 config/             # Environment & configuration loader
│   ├── 📂 db/                 # MySQL 8.4 database pool management
│   ├── 📂 feed/               # Global Feed domain models, handlers, repository & tests
│   ├── 📂 hubs/               # Chat Hubs WebSocket hub, routing, attachments & tests
│   ├── 📂 search/             # In-memory search & tokenization library
│   ├── 📂 space/              # My Space document tree, autosave, Kanban & markdown parser
│   └── 📂 voice/              # Go Voice Gateway connecting to C++ audio engine
├── 📂 migrations/             # Raw MySQL 8.4 InnoDB DDL schema migrations
├── 📂 scripts/                # Commit manifest & governance scripts (voxion-bot)
├── 📂 web/
│   ├── 📂 static/             # Static assets (CSS, JS dumb-pipe, WASM binary)
│   └── 📂 templates/          # Go HTML templates & HTMX partials
├── 📄 BENCHMARKS.md           # Concurrency load benchmark telemetry report
├── 📄 ROADMAP.md              # Authoritative multi-phase migration roadmap
├── 📄 Dockerfile              # Multi-stage production container build
├── 📄 docker-compose.yml      # Local orchestration (MySQL 8.4, Redis, MinIO, YooMates)
└── 📄 Makefile                # Automation commands (rebase pull/push, tests, benchmarks)
```

---

## 4. Quickstart & Local Development

### Prerequisites
- **Go 1.22+**
- **Docker & Docker Compose**
- **C++ Compiler (g++ / MinGW / Clang)**

### 1. Start Infrastructure (MySQL 8.4, Redis, MinIO)
```bash
docker compose up -d db redis s3
```

### 2. Run Database Migrations
Migrations run automatically on container boot, or can be applied using raw MySQL:
```bash
# Apply schema directly
docker compose exec db mysql -u yoomates -p yoomates < migrations/0001_initial_schema.sql
```

### 3. Compile WebAssembly & C++ Audio Engine
```bash
make wasm
make audio-engine
```

### 4. Run the Server
```bash
make run
# Or directly: go run ./cmd/server/main.go
```
The server will boot in **< 15ms** and serve requests at `http://localhost:8080`.

---

## 5. Testing & Benchmarks

### Run Unit Tests
```bash
make test
# Or: go test -v -count=1 ./...
```

### Run C++ Audio Engine Verification
```bash
make test-audio
```

### Run Concurrency Load Benchmarks
```bash
make benchmark
```
*Full benchmarking methodology and comparison telemetry available in [BENCHMARKS.md](BENCHMARKS.md).*

---

## 6. Global Keyboard Shortcuts

| Shortcut | Action |
|:---|:---|
| <kbd>Ctrl+K</kbd> / <kbd>Cmd+K</kbd> | Open Universal Quick Switcher (Command Palette) |
| <kbd>Ctrl+F</kbd> / <kbd>Cmd+F</kbd> | Focus Instant WebAssembly Channel Search |
| <kbd>Esc</kbd> | Dismiss active modal / quick switcher / clear search |
| <kbd>↑</kbd> / <kbd>↓</kbd> | Navigate results in Quick Switcher |
| <kbd>↵</kbd> | Execute selected navigation or command |

---

<br>
<div align="right">
  <b>Rudranarayan Jena</b><br>
  <i>Founder @ Voxion Labs</i>
</div>

---
<div align="center">
  <strong>(c) 2026 Voxion Labs & Rudranarayan Jena</strong>
</div>
