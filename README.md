# neoncast

> Self-hosted podcast hosting + RSS generator with built-in analytics, chapter markers, and WebSub support. Single binary deploy.

**Language:** Go  
**Constraint:** Nothing new except glue  
**Stack:** echo, bleve, minio (S3-compatible), ffprobe

---

## Features

- Drop MP3s in a directory, get a feed
- Auto-generated RSS 2.0 with iTunes tags
- Chapter marker support (MP3chap / JSON sidecar)
- Built-in analytics (downloads, geo, client)
- WebSub pub/sub for feed updates
- Web UI for episode management
- Single binary, zero-config defaults

---

## Development Plan

1. Phase 1: HTTP server scaffold (echo) with static file serving
2. Phase 2: RSS 2.0 generator with iTunes extension
3. Phase 3: Directory watcher + auto-ingest pipeline
4. Phase 4: Episode metadata extraction (ffprobe, ID3)
5. Phase 5: Analytics middleware (download tracking)
6. Phase 6: WebSub publisher implementation
7. Phase 7: Web UI for upload and management
8. Phase 8: Single-binary build and release

---

## Getting Started

### Prerequisites

- Go toolchain

### Build

```bash
# See PLAN.md for detailed build instructions per phase
cd neoncast
```

### Run

```bash
# See PLAN.md for run instructions
```

---

## Architecture

See `PLAN.md` for detailed architecture decisions and implementation notes.

---

## License

MIT
