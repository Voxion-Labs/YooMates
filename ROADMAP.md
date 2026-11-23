# YooMates Architecture Migration & Execution Roadmap

> **Authoritative Specification Document**
> Governed by Voxion Labs architectural standards.

---

## 1. Core Product Identity & Pillars

YooMates is designed around three tightly integrated pillars:
1. **Hubs (Primary / Default View)**: Discord-like real-time team collaboration with guilds/servers, channels (text/audio/video), direct messages, message reactions, threads, typing indicators, and presence.
2. **Global**: Twitter/X-like organization-wide publishing feed with posts, comments, likes, bookmarks, quote posts, polls, and infinite scroll.
3. **My Space**: Notion-like personal execution workspace with nested hierarchical documents, markdown/block-based editing, reminders, and kanban/list views.

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

### **Phase 0: Project Governance & Safety Automation** `[IN PROGRESS]`
- [x] **Sub-phase 0.1**: Master blueprint & rules documentation (`ROADMAP.md`).
- [ ] **Sub-phase 0.2**: `voxion-bot` commit manifest generator & secure GitHub Action workflow (strict fork isolation & write permissions).
- [ ] **Sub-phase 0.3**: Rebase-and-push safe `Makefile` with clean sync & manifest automation.
- [ ] **Sub-phase 0.4**: Phase 0 verification and awaiting user approval.

---

### **Phase 1: Foundation, DB Migration & Go Core** `[PENDING]`
- [ ] **Sub-phase 1.1**: Go module initialization (`go mod init yoomates`) & multi-stage `Dockerfile` + `docker-compose.yml` (Go, PostgreSQL, MinIO/Local Storage).
- [ ] **Sub-phase 1.2**: Pure SQL schema migration (`migrations/*.sql`) converting Prisma PostgreSQL models to raw, optimized SQL with `pgx/v5` connection pooling.
- [ ] **Sub-phase 1.3**: Native Go session-based authentication engine (Argon2id hashing, HTTP-only secure cookie sessions, OAuth2 handlers).

---

### **Phase 2: Core Chat "Hubs" Engine (Default Discord Parity)** `[PENDING]`
- [ ] **Sub-phase 2.1**: Native Go WebSocket Hub with goroutine connection pools (replacing Pusher vendor lock-in).
- [ ] **Sub-phase 2.2**: 3-column Hubs chat layout in Go Templates + HTMX (Server rail, Channels tree, Message stream, Active thread drawer).
- [ ] **Sub-phase 2.3**: Real-time chat interactions via `hx-ext="ws"` (live messaging, message editing, soft-delete, typing indicators, emoji reactions).
- [ ] **Sub-phase 2.4**: Local/S3 multipart attachment upload pipeline (replacing UploadThing).

---

### **Phase 3: Client-Side Resilience & Dumb-Pipe WASM/Cache** `[PENDING]`
- [ ] **Sub-phase 3.1**: Dumb Pipe JS (<100 lines) + Service Worker & IndexedDB caching for 0ms instant offline chat load.
- [ ] **Sub-phase 3.2**: WASM module integration (C++ / Go compiled to `.wasm`) for instant local search and client-side message filtering.

---

### **Phase 4: High-Performance C++ Audio/Media Engine** `[PENDING]`
- [ ] **Sub-phase 4.1**: C++ / WebRTC low-latency audio packet processing & voice channel gateway.
- [ ] **Sub-phase 4.2**: Media transcode & audio buffer optimization.

---

### **Phase 5: Global Feed & My Space Migration** `[PENDING]`
- [ ] **Sub-phase 5.1**: Global Feed (Twitter/X style) with chronological posts, quote posts, comments, likes, polls, and HTMX infinite scroll.
- [ ] **Sub-phase 5.2**: My Space (Notion style) minimalist block/markdown editor with local-first autosave and kanban/list views.

---

### **Phase 6: Hardening, Benchmarking & Legacy Cleanup** `[PENDING]`
- [ ] **Sub-phase 6.1**: Monochromatic / utilitarian styling audit and keyboard shortcuts (`Ctrl+K`, quick switcher).
- [ ] **Sub-phase 6.2**: Concurrency load benchmarking (Go WebSocket vs legacy Pusher/Node).
- [ ] **Sub-phase 6.3**: Complete deprecation & removal of legacy TS/Next.js files (`package.json`, `node_modules`, `src/app`).
