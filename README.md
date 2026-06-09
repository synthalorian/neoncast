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

1. ✅ Phase 1: HTTP server scaffold (echo) with static file serving
2. ✅ Phase 2: RSS 2.0 generator with iTunes extension
3. ✅ Phase 3: Directory watcher + auto-ingest pipeline
4. ✅ Phase 4: Episode metadata extraction (ffprobe, ID3)
5. ✅ Phase 5: Analytics middleware (download tracking)
6. ✅ Phase 6: WebSub publisher implementation
7. ✅ Phase 7: Web UI for upload and management
8. ✅ Phase 8: Single-binary build and release

---

## Admin Dashboard

The web UI is available at `/admin` and provides:

- **Podcast Settings**: Edit feed metadata (title, description, author, email, copyright, image URL, category, language, explicit flag)
- **Upload Episode**: Drag-and-drop or click-to-browse audio file upload (supports MP3, M4A, OGG, WAV)
- **Episode Management**: View all episodes in a sortable table, edit metadata inline, delete episodes
- **Analytics**: Download counts, unique listeners, recent activity (via `/api/analytics/*` endpoints)

## Getting Started

### Prerequisites

- Go toolchain

### Build

```bash
make build
```

Or manually:

```bash
go build -ldflags "-X neoncast/internal/version.Version=$(git describe --tags --always) -X neoncast/internal/version.Commit=$(git rev-parse --short HEAD) -X neoncast/internal/version.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o neoncast ./cmd/neoncast
```

### Run

```bash
./neoncast
```

The server starts on port 8080 by default. Visit `http://localhost:8080/admin` for the dashboard.

### Check Version

```bash
./neoncast -version
```

### Cross-Platform Release Build

Build binaries for all supported platforms:

```bash
make release
```

Or use the release script directly:

```bash
./scripts/release.sh v1.0.0
```

This produces binaries in `build/`:
- `neoncast-linux-amd64`
- `neoncast-linux-arm64`
- `neoncast-darwin-amd64`
- `neoncast-darwin-arm64`
- `neoncast-windows-amd64.exe`

### Run Tests

```bash
make test
```

---

## Architecture

See `PLAN.md` for detailed architecture decisions and implementation notes.

---

## License

MIT
