# OpenSearch Database Management

This directory contains configuration and scripts for managing Celeste's OpenSearch indices.

## Quick Start

### Health Check
```bash
./scripts/validate.sh
```

### Reindex All Indices
```bash
./scripts/reindex.sh
```

### Cleanup & Optimize
```bash
./scripts/cleanup.sh
```

---

## Indices Overview

### 1. `celeste_capabilities`
**Purpose:** Store Celeste's comprehensive functionality documentation for RAG (Retrieval-Augmented Generation)

**Contents:**
- All 7 Celeste projects with descriptions
- Feature listings per project
- API endpoints and integration points
- Use cases and examples

**Priority:** HIGH (Retrieves first when user asks "what can you do")

**Mapping:** `indices/celeste_capabilities/mapping.json`

**Seed Data:** `indices/celeste_capabilities/documents.jsonl`

**Query Example:**
```json
{
  "query": {
    "match": {
      "capability_name": "chat"
    }
  }
}
```

---

### 2. `celeste_emotes`
**Purpose:** Retrieve contextually appropriate 7TV emotes based on message intent

**Contents:**
- 7TV emote catalog
- Vibe tags (smug, lewd, cursed, chaotic, etc.)
- Usage samples and contexts
- Intent-action mappings

**Priority:** MEDIUM

**Mapping:** `indices/celeste_emotes/mapping.json`

**Seed Data:** `indices/celeste_emotes/embedded_emote_samples_500.json`

**RAG Configuration:**
- Features: `[vibe_tags, intent_action_classification]`
- Frequency cap: 2 emotes per message

---

### 3. `celeste_user_profiles`
**Purpose:** Store user behavior data for moderation and personalization

**Contents:**
- User IDs and usernames
- Behavior scores (0-100)
- Infractions and warnings
- Chat history samples (recent quotes)
- Last seen timestamp
- Interaction patterns

**Priority:** MEDIUM

**Mapping:** `indices/celeste_user_profiles/mapping.json`

**Query Example:**
```json
{
  "query": {
    "match": {
      "username": "example_user"
    }
  }
}
```

**Retrieval Hints:**
- Normalize username to lowercase, strip leading `@`
- Prefer documents ending with `twitch_users/user_{name}_wrapped.json`
- If multiple matches, choose newest by `processed_date`

---

### 4. `celeste_chat_logs`
**Purpose:** Store sample chat interactions for pattern learning (LOW priority for general queries)

**Contents:**
- Sample chat messages
- Interaction patterns
- User motifs (gooning jokes, baiting, etc.)
- Response contexts

**Priority:** LOW (Use only when explicitly relevant)

**Mapping:** `indices/celeste_chat_logs/mapping.json`

**Note:** This index should not be searched unless the query explicitly requests chat history or interaction examples.

---

## NIKKE Sub-Agent Indices

The NIKKE sub-agent (`nikke-agent/opensearch/`) manages its own indices:

### NIKKE Indices (Managed by nikke-agent/)
- `nikke_characters` - Character database
- `nikke_tiers` - Tier lists and rankings
- `nikke_guides` - Build guides and farm routes
- `nikke_union_data` - Protected union-specific data (encrypted, auth-required)

These are NOT queried by main Celeste agent; they're handled by the NIKKE sub-agent when routing occurs.

---

## Retrieval Strategy

When the Celeste agent receives a query, it retrieves from indices in this priority order:

1. **celeste_capabilities** (if user asks "what can you do")
2. **celeste_user_profiles** (if query mentions a username)
3. **celeste_emotes** (if intent suggests emote context)
4. **celeste_chat_logs** (if explicitly requested, low priority)

**NIKKE queries** are routed to the NIKKE sub-agent instead (see `routing/routing_rules.json`).

---

## Scripts

### `reindex.sh`
Reindexes all Celeste indices from scratch.

**Usage:**
```bash
./opensearch/scripts/reindex.sh [--all | --indices celeste_capabilities,celeste_user_profiles]
```

**Options:**
- `--all`: Reindex all indices
- `--indices`: Comma-separated list of specific indices to reindex

---

### `cleanup.sh`
Removes duplicate and noisy data from indices.

**Usage:**
```bash
./opensearch/scripts/cleanup.sh [--dry-run]
```

**Actions:**
- Remove duplicate chat log entries
- Remove outdated user profiles (older than 60 days)
- Optimize index sizes
- Fix malformed documents

**Note:** Use `--dry-run` to preview changes without applying them.

---

### `seed_capabilities.sh`
Seeds the `celeste_capabilities` index with capability documents from `Celeste_Capabilities.json`.

**Usage:**
```bash
./opensearch/scripts/seed_capabilities.sh
```

**Updates from:**
- `Celeste_Capabilities.json`
- Regenerates all capability documents
- Useful after adding new projects or features

---

### `validate.sh`
Validates OpenSearch health and index status.

**Usage:**
```bash
./opensearch/scripts/validate.sh
```

**Checks:**
- OpenSearch cluster health (green/yellow/red)
- Index status and shard allocation
- Document counts per index
- Index size statistics

---

## Manual Index Management

### Get Index Health
```bash
curl -X GET "${OPENSEARCH_ENDPOINT}/_cluster/health"
```

### Get Index Stats
```bash
curl -X GET "${OPENSEARCH_ENDPOINT}/${INDEX_NAME}/_stats"
```

### Add Document to Index
```bash
curl -X POST "${OPENSEARCH_ENDPOINT}/${INDEX_NAME}/_doc" \
  -H 'Content-Type: application/json' \
  -d '{...document...}'
```

### Update Index Mapping
```bash
curl -X PUT "${OPENSEARCH_ENDPOINT}/${INDEX_NAME}/_mapping" \
  -H 'Content-Type: application/json' \
  -d '{...mapping...}'
```

---

## Environment Variables

Required:
- `OPENSEARCH_ENDPOINT` - Base URL of OpenSearch cluster (e.g., `https://opensearch.example.com:9200`)
- `OPENSEARCH_USER` - Username for authentication
- `OPENSEARCH_PASSWORD` - Password for authentication

Optional:
- `OPENSEARCH_TIMEOUT_MS` - Request timeout in milliseconds (default: 10000)
- `OPENSEARCH_VERIFY_SSL` - Verify SSL certificates (default: true)

---

## Troubleshooting

### "Index not found" error
- Run `./scripts/reindex.sh --indices celeste_capabilities`
- Verify OPENSEARCH_ENDPOINT is correct

### "Cluster health is RED"
- Check cluster status: `curl -X GET "${OPENSEARCH_ENDPOINT}/_cluster/health"`
- May need to increase shard replication or disk space

### "Noisy retrieval results"
- Run `./scripts/cleanup.sh --dry-run` to identify problematic documents
- Review retrieval scoring in `routing/routing_rules.json`

---

## Monitoring

Monitor OpenSearch indices regularly:

1. **Index size growth:**
   ```bash
   ./scripts/validate.sh | grep "index.store.size"
   ```

2. **Document counts:**
   ```bash
   ./scripts/validate.sh | grep "docs.count"
   ```

3. **Cluster health:**
   ```bash
   curl -X GET "${OPENSEARCH_ENDPOINT}/_cluster/health?pretty"
   ```

---

## Related Files

- `../routing/routing_rules.json` - Retrieval priority and routing logic
- `../Celeste_Capabilities.json` - Capability documents to be indexed
- `../celeste_essence.json` - System prompt referencing these indices
