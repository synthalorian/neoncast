# neoncast — Implementation Plan

## Project Overview

Self-hosted podcast hosting + RSS generator with built-in analytics, chapter markers, and WebSub support. Single binary deploy.

**Language:** Go  
**Constraint:** Nothing new except glue  
**Stack:** echo, bleve, minio (S3-compatible), ffprobe

---

## Phase Breakdown

### Phase 1: HTTP server scaffold (echo) with static file serving

**Goal:** Phase 1: HTTP server scaffold (echo) with static file serving

**Deliverables:**
- [ ] Core implementation
- [ ] Tests
- [ ] Documentation update

**Notes:**
- 

---

### Phase 2: RSS 2.0 generator with iTunes extension

**Goal:** Phase 2: RSS 2.0 generator with iTunes extension

**Deliverables:**
- [ ] Core implementation
- [ ] Tests
- [ ] Documentation update

**Notes:**
- 

---

### Phase 3: Directory watcher + auto-ingest pipeline

**Goal:** Phase 3: Directory watcher + auto-ingest pipeline

**Deliverables:**
- [ ] Core implementation
- [ ] Tests
- [ ] Documentation update

**Notes:**
- 

---

### Phase 4: Episode metadata extraction (ffprobe, ID3)

**Goal:** Phase 4: Episode metadata extraction (ffprobe, ID3)

**Deliverables:**
- [ ] Core implementation
- [ ] Tests
- [ ] Documentation update

**Notes:**
- 

---

### Phase 5: Analytics middleware (download tracking)

**Goal:** Phase 5: Analytics middleware (download tracking)

**Deliverables:**
- [ ] Core implementation
- [ ] Tests
- [ ] Documentation update

**Notes:**
- 

---

### Phase 6: WebSub publisher implementation

**Goal:** Phase 6: WebSub publisher implementation

**Deliverables:**
- [ ] Core implementation
- [ ] Tests
- [ ] Documentation update

**Notes:**
- 

---

### Phase 7: Web UI for upload and management

**Goal:** Phase 7: Web UI for upload and management

**Deliverables:**
- [ ] Core implementation
- [ ] Tests
- [ ] Documentation update

**Notes:**
- 

---

### Phase 8: Single-binary build and release

**Goal:** Phase 8: Single-binary build and release

**Deliverables:**
- [ ] Core implementation
- [ ] Tests
- [ ] Documentation update

**Notes:**
- 

---

## Architecture Notes

### Key Decisions

- 

### Data Flow

```
[Input] → [Parse] → [Transform] → [Output]
```

### Error Handling Strategy

- 

---

## Testing Strategy

- Unit tests for core functions
- Integration tests for full pipeline
- Benchmarks for performance-critical paths

---

## Open Questions

1. 
2. 

---

*Generated for opencode sprint. Implement phase by phase. DO NOT RESEARCH. Build directly.*
