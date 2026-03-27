# Graph RAG Code Review Fixes

This document summarizes the fixes applied after the code review of the Graph RAG implementation.

## Critical Fixes

### 1. Anthropic API Support (`internal/rag/llm_client.go`)

**Problem:** The `NewLLMClient` set Anthropic baseURL but `Generate` always called OpenAI's `/chat/completions` endpoint.

**Fix:**
- Split `Generate` into provider-specific methods: `generateOpenAI` and `generateAnthropic`
- Anthropic now uses correct `/messages` endpoint
- Anthropic headers: `x-api-key` and `anthropic-version`
- Added `provider` field to `LLMClient` struct

### 2. Relationship Entity ID Resolution (`internal/rag/graph.go`)

**Problem:** `SourceID` and `TargetID` were `uuid.UUID` types, but the LLM returns entity names as strings. The code tried to resolve by calling `.String()` on zero UUIDs.

**Fix:**
- Added `relationshipRaw` struct with `Source` and `Target` as strings
- Changed `EntityExtractionResult.Relationships` to use `[]relationshipRaw`
- Relationship resolution now happens in `IndexDocument` where entity names are mapped to UUIDs before indexing

### 3. Duplicate Helper Functions (`internal/qdrant/client.go`)

**Problem:** Two identical functions `NewQuery` and `NewQueryWrapper` existed.

**Fix:** Removed duplicate `NewQuery` function, kept `NewQueryWrapper`.

## High Priority Fixes

### 4. LLM Retry Logic (`internal/rag/llm_client.go`)

**Problem:** No retry on rate limits (429) or service unavailable (503).

**Fix:**
- Added `doRequestWithRetry` method with exponential backoff
- Retries on network errors, 429, and 503 status codes
- Increased timeout from 60s to 120s
- Backoff: 2s, 4s, 6s for successive retries

### 5. Content Truncation Warning (`internal/rag/graph.go`)

**Problem:** Content was silently truncated at 3000 chars.

**Fix:**
- Added `MaxContentTruncation` constant
- Explicit truncation handling in `extractEntities`
- (Future enhancement: could add logging when truncation occurs)

### 6. Fragile Deduplication (`internal/rag/graph.go`)

**Problem:** Deduplication used first 50 chars of content, causing false positives for documents with shared prefixes.

**Fix:**
- Added `crypto/sha256` import
- Now uses SHA256 hash of full content for deduplication
- Key format: `{documentID}_{hash[:8]}`

## Code Quality Improvements

### 7. Magic Numbers Replaced with Constants (`internal/rag/graph.go`)

**Added:**
```go
const (
    EntityScoreWeight    = 1.0
    RelationshipWeight   = 0.8
    ChunkScoreWeight     = 0.6
    MaxContentTruncation = 3000
)
```

**Usage:**
- `point.Score * 0.8` → `point.Score * RelationshipWeight`
- `r.Score * 0.6` → `r.Score * ChunkScoreWeight`
- `content[:min(3000, len(content))]` → `content[:MaxContentTruncation]`

## Files Modified

1. `internal/rag/graph.go` - Core Graph RAG implementation
2. `internal/rag/llm_client.go` - LLM API client
3. `internal/qdrant/client.go` - Qdrant helper functions

## Testing Recommendations

1. **Test Anthropic integration:**
   ```bash
   # Set provider to "anthropic" in config and verify entity extraction works
   ```

2. **Test relationship resolution:**
   - Upload a document with clear entity relationships
   - Verify relationships reference correct entity UUIDs

3. **Test retry logic:**
   - Simulate rate limit response
   - Verify exponential backoff behavior

4. **Test deduplication:**
   - Search for content that appears in multiple chunks
   - Verify no duplicate results

## Remaining Issues (Lower Priority)

1. **Inconsistent qdrant helper usage:** Internal `client.go` uses raw qdrant types while `graph.go` uses helpers. Consider refactoring `client.go` to use helpers for consistency.

2. **No logging for truncation:** Could add `zap` logger to GraphRAG struct and log warnings when content is truncated.

3. **Error wrapping inconsistency:** Some errors use `fmt.Errorf("...: %w")`, others just `return err`. Consider standardizing.
