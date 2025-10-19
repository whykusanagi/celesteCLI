# OpenSearch Integration for Celeste CLI

## Overview

The Celeste CLI now stores structured conversation data in DigitalOcean Spaces at `s3://whykusanagi/celeste/conversations/` for OpenSearch consumption. This enables Retrieval-Augmented Generation (RAG) capabilities for better contextual responses.

## S3 Path Structure

```
s3://whykusanagi/celeste/conversations/
├── 2025/
│   ├── 01/
│   │   ├── 15/
│   │   │   ├── 1757744113479692000.json
│   │   │   └── 1757744113479692001.json
│   │   └── 16/
│   └── 02/
└── ...
```

## Conversation Data Schema

Each conversation is stored as a JSON file with the following structure:

### Core Fields
- `id`: Unique conversation identifier (timestamp-based)
- `timestamp`: ISO 8601 timestamp
- `user_id`: User identifier (default: "kusanagi")
- `content_type`: Type of content requested (tweet, ytdesc, etc.)
- `tone`: Requested tone (teasing, cute, dramatic, etc.)
- `game`: Game context
- `persona`: Celeste persona used
- `prompt`: Original user prompt
- `response`: Celeste's generated response
- `success`: Whether the request was successful

### OpenSearch RAG Fields
- `intent`: High-level intent (content_creation, general_interaction)
- `purpose`: Specific purpose (tweet, ytdesc, discord, etc.)
- `platform`: Target platform (twitter, youtube, discord, etc.)
- `sentiment`: Overall sentiment (positive, negative, neutral, mixed)
- `topics`: Extracted topics/keywords for semantic search
- `tags`: Searchable tags for filtering
- `context`: Additional context information

### Metadata Fields
- `tokens_used`: API token usage statistics
- `command_line`: Full CLI command used
- `api_endpoint`: API endpoint used

## OpenSearch Index Mapping

For optimal search performance, create an OpenSearch index with the following mapping:

```json
{
  "mappings": {
    "properties": {
      "id": { "type": "keyword" },
      "timestamp": { "type": "date" },
      "user_id": { "type": "keyword" },
      "content_type": { "type": "keyword" },
      "tone": { "type": "keyword" },
      "game": { "type": "keyword" },
      "persona": { "type": "keyword" },
      "prompt": { "type": "text", "analyzer": "standard" },
      "response": { "type": "text", "analyzer": "standard" },
      "intent": { "type": "keyword" },
      "purpose": { "type": "keyword" },
      "platform": { "type": "keyword" },
      "sentiment": { "type": "keyword" },
      "topics": { "type": "keyword" },
      "tags": { "type": "keyword" },
      "context": { "type": "text" },
      "success": { "type": "boolean" },
      "tokens_used": { "type": "object" },
      "metadata": { "type": "object" }
    }
  }
}
```

## Usage

### Enable S3 Sync
```bash
./celestecli --type tweet --game "NIKKE" --tone "teasing" --sync
```

### Configuration
Create `~/.celeste.cfg`:
```bash
access_key_id=your_digitalocean_spaces_access_key
secret_access_key=your_digitalocean_spaces_secret_key
endpoint=https://sfo3.digitaloceanspaces.com
region=sfo3
bucket_name=whykusanagi
```

## RAG Benefits

The structured data enables:

1. **Semantic Search**: Find similar conversations by content, tone, or game
2. **Context Retrieval**: Retrieve relevant past conversations for context
3. **Pattern Analysis**: Analyze successful conversation patterns
4. **Personalization**: Build user-specific conversation history
5. **Quality Improvement**: Learn from past interactions

## Example Queries

### Find Similar Conversations
```json
{
  "query": {
    "bool": {
      "must": [
        { "term": { "game": "nikke" } },
        { "term": { "tone": "teasing" } },
        { "term": { "platform": "twitter" } }
      ]
    }
  }
}
```

### Semantic Search
```json
{
  "query": {
    "multi_match": {
      "query": "bunny suit gaming stream",
      "fields": ["prompt", "response", "topics"]
    }
  }
}
```

### Recent Successful Conversations
```json
{
  "query": {
    "bool": {
      "must": [
        { "term": { "success": true } },
        { "range": { "timestamp": { "gte": "now-7d" } } }
      ]
    }
  },
  "sort": [{ "timestamp": { "order": "desc" } }]
}
```
