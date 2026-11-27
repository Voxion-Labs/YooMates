# YooMates Operational & Contribution Protocols

YooMates operates under strict architectural constraints designed for deterministic, low-latency performance. We do not accept unoptimized allocations, unindexed queries, or concurrency mutations that compromise real-time synchronization. This repository is maintained as a proprietary, high-performance team operating platform.

If you intend to submit a Pull Request, you must adhere strictly to the following directives.

## 1. Architectural Standards
All code submitted to YooMates must adhere to our core system constraints:
* **Real-time Determinism:** Any modifications to the WebSocket hub or event fanout engine must guarantee sub-millisecond p99 dispatch and non-blocking channel delivery.
* **Database & Concurrency Strictness:** SQLite queries must be prepared statements executed under WAL mode. All shared state access across hubs or voice rooms must maintain strict mutex isolation and avoid lock contention.
* **Audio Engine Reliability:** Modifications to the C++ audio engine (`engine/audio`) must ensure bounded jitter buffers, zero memory leaks, and sub-10ms packet dispatch.
* **Client-Side Minimalism:** The client-side JavaScript layer (`web/static/js/dumb-pipe.js`) is strictly a stateless event relay (<230 lines total). No client-side business logic or third-party UI framework dependencies are permitted.

## 2. Pull Request (PR) Governance
Before initiating a pull request, ensure your branch satisfies this verification sequence:
1. **[METRICS] Telemetry & Benchmarks:** Provide execution telemetry for touched components (e.g., allocations/op, WebSocket broadcast latency, SQLite query duration).
2. **[TESTS] Automated Verification:** All unit tests (`make test` / `go test -v ./...`) and native C++ audio engine tests (`make test-audio` / `bin/yoomates-voice.exe --test`) must pass with zero failures.
3. **[FORMAT] Code Cleanliness:** Go source must be formatted via `gofmt -w .`, with zero linter errors or unused imports.
4. **[ISOLATION] Threat Model:** Prove that modifications do not introduce Cross-Site Scripting (XSS) in Hubs messaging or expose private My Space documents across tenant boundaries.

*Note: PRs failing to provide empirical telemetry or violating architectural constraints will be closed without review.*

## 3. Vulnerability Disclosure
**DO NOT** open public issues for authentication bypasses, real-time message hijacking, or database injection vulnerabilities. Public disclosure of critical threats compromises platform integrity.
* All security reports must be routed internally to security contacts.
* Contact the Lead Architect directly for secure transmission protocols.

## 4. Code of Conduct
We evaluate algorithmic efficiency and architectural integrity. Submissions will be scrutinized objectively based on concurrency safety, memory footprint, and system determinism. Keep technical discussions clinical, precise, and focused on system architecture.
