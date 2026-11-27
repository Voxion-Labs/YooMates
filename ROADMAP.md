# YooMates Architecture Migration & Execution Roadmap

> **Authoritative Specification Document**
> Governed by Voxion Labs architectural standards.

---

## 1. Core Product Identity & Pillars

YooMates is designed around three tightly integrated pillars:
1. **Hubs (Primary / Default View)**: Real-time team collaboration with guilds/servers, channels (text/audio/video), direct messages, message reactions, threads, typing indicators, and presence.
2. **Global**: Organization-wide publishing feed with posts, comments, likes, bookmarks, quote posts, polls, and infinite scroll.
3. **My Space**: Personal execution workspace with nested hierarchical documents, markdown/block-based editing, reminders, and kanban/list views.

---

## 2. Non-Negotiable Engineering Rules & Constraints

1. **Zero `node_modules` & Zero TypeScript at Runtime**:
   - The entire backend is powered by a single, high-performance Go binary.
   - No npm dependencies, no Node.js runtime, no Next.js build overhead.
2. **JavaScript as a "Dumb Pipe"**:
   - Any client-side JavaScript must remain strictly under ~100 lines of pure vanilla JS.
   - JS has **zero business logic** and **zero state management**.
   - JS acts purely as a transport pipe: loading WASM, relaying DOM events, and passing WebSocket/IndexedDB byte buffers.
3. **Hypermedia & Server-Driven UI (Go Templates + HTMX)**:
   - Go's `html/template` delivers pre-rendered HTML partials with <5ms TTFB.
   - HTMX handles dynamic swapping, real-time WebSocket streams (`hx-ext="ws"`), and inline mutations.
4. **Local DB & WASM Client Cache**:
   - Instant 0ms offline chat loading using IndexedDB / Cache API and client-side WASM engine (C++ / Go).
   - Local search and client-side caching without backend latency.
5. **Aesthetic Direction: Engineered Minimalism**:
   - **STRICTLY NO "vibecoder" or cyberpunk tropes**: No gratuitous neon borders, no pulsating glow effects, no purple/cyan sci-fi cards.
   - Industrial, high-density, distraction-free UI (inspired by Linear, Sublime Text, GitHub, and Dieter Rams principles).
6. **Execution Discipline**:
   - **One sub-phase at a time.** Never execute an entire phase in one leap.
   - **Wait for explicit user approval** after each sub-phase before moving forward.
   - **NO external software installation** on the user's local operating system. Rely strictly on existing tools (Go 1.27, Git, MinGW Make) and container manifests.

---

## 3. Master Phases & Sub-phases Breakdown

### **Phase 0: Project Governance & Safety Automation** `[DONE]`
- [x] **Sub-phase 0.1**: Master blueprint & rules documentation (`ROADMAP.md`).
- [x] **Sub-phase 0.2**: `voxion-bot` commit manifest generator & secure GitHub Action workflow (strict fork isolation & write permissions).
- [x] **Sub-phase 0.3**: Rebase-and-push safe `Makefile` with clean sync & manifest automation.
- [x] **Sub-phase 0.4**: CI/CD pipeline (`.github/workflows/ci.yml`) for Go formatting, vet, tests, and multi-stage Docker build verification.
- [ ] **Sub-phase 0.5**: Phase 0 verification and awaiting user approval.

---

### **Phase 1: Foundation, DB Migration & Go Core** `[DONE]`
- [x] **Sub-phase 1.1**: Go module initialization (`go mod init yoomates`), multi-stage `Dockerfile`, full orchestration `docker-compose.yml` (Go, MySQL 8.4, Redis, MinIO local S3), and native `internal/config` configuration management.
- [x] **Sub-phase 1.2**: Pure SQL schema migration (`migrations/0001_initial_schema.sql`) converting Prisma models to raw, optimized MySQL 8.4 (InnoDB) SQL with `go-sql-driver/mysql` connection pooling and migration runner (`internal/db`).
- [x] **Sub-phase 1.3**: Native Go session-based authentication engine (Argon2id password hashing, HTTP-only secure cookie sessions, auth handlers, and middleware).

---

### **Phase 2: Core Chat "Hubs" Engine** `[DONE]`
- [x] **Sub-phase 2.1**: Native Go WebSocket Hub with goroutine connection pools, channel/server pub-sub routing, and ephemeral presence tracking (replacing Pusher vendor lock-in).
- [x] **Sub-phase 2.2**: 3-column Hubs chat layout in Go Templates + HTMX (Server rail, Channels tree, Message stream, Active thread/member drawer) with engineered minimalist styling and dumb-pipe JS.
- [x] **Sub-phase 2.3**: Real-time chat interactions via `hx-ext="ws"` (live messaging, message editing, soft-delete, typing indicators, emoji reactions) backed by MySQL repository and HTMX OOB swaps.
- [x] **Sub-phase 2.4**: Local/S3 multipart attachment upload pipeline (replacing UploadThing).

---

### **Phase 3: Client-Side Resilience & Dumb-Pipe WASM/Cache** `[DONE]`
- [x] **Sub-phase 3.1**: Dumb Pipe JS (<110 lines) + Service Worker & IndexedDB caching for 0ms instant offline chat load.
- [x] **Sub-phase 3.2**: WASM module integration (Go WebAssembly compiled to `.wasm`) for instant local search and client-side message filtering.

---

### **Phase 4: High-Performance C++ Audio/Media Engine** `[DONE]`
- [x] **Sub-phase 4.1**: C++ / WebRTC low-latency audio packet processing & voice channel gateway.
- [x] **Sub-phase 4.2**: Media transcode & audio buffer optimization.

---

### **Phase 5: Global Feed & My Space Migration** `[DONE]`
- [x] **Sub-phase 5.1**: Global Feed with chronological posts, quote posts, comments, likes, polls, and HTMX infinite scroll.
- [x] **Sub-phase 5.2**: My Space minimalist block/markdown editor with local-first autosave and kanban/list views.

---

### **Phase 6: Hardening, Benchmarking & Legacy Cleanup** `[DONE]`
- [x] **Sub-phase 6.1**: Monochromatic / utilitarian styling audit and keyboard shortcuts (`Ctrl+K`, quick switcher).
- [x] **Sub-phase 6.2**: Concurrency load benchmarking (Go WebSocket vs legacy Pusher/Node).
- [x] **Sub-phase 6.3**: Complete deprecation & removal of legacy TS/Next.js files (`package.json`, `node_modules`, `src/app`).
