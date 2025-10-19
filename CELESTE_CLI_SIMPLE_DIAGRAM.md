# CelesteCLI Simple Architecture Diagram

## Main Flow Diagram

```mermaid
flowchart TD
    A[Start CLI] --> B[Parse Arguments]
    B --> C{Override Check}
    C -->|Yes| D[🔓 Override Mode]
    C -->|No| E[Load Configs]
    D --> E
    
    E --> F[Build Prompt]
    F --> G{NSFW Mode?}
    G -->|Yes| H[Venice.ai API]
    G -->|No| I[CelesteAI API]
    
    H --> J[Uncensored Response]
    I --> K[Standard Response]
    
    J --> L[Display Output]
    K --> L
    
    L --> M{Sync?}
    M -->|Yes| N[Upload to S3]
    M -->|No| O[End]
    N --> O
    
    style A fill:#e1f5fe
    style D fill:#ffcdd2
    style H fill:#f8bbd9
    style I fill:#c8e6c9
    style O fill:#e1f5fe
```

## Bot Integration Flow

```mermaid
sequenceDiagram
    participant User
    participant Bot
    participant CLI
    participant API
    participant S3
    
    User->>Bot: Send message
    Bot->>CLI: Set environment variables
    Note over Bot,CLI: CELESTE_USER_ID<br/>CELESTE_PLATFORM<br/>CELESTE_CHANNEL_ID
    
    CLI->>CLI: Parse arguments
    CLI->>CLI: Load configurations
    CLI->>CLI: Build prompt
    
    alt NSFW Mode
        CLI->>API: Venice.ai request
        API-->>CLI: Uncensored response
    else Standard Mode
        CLI->>API: CelesteAI request
        API-->>CLI: Standard response
    end
    
    CLI->>CLI: Parse response
    CLI->>S3: Upload conversation
    S3-->>CLI: Upload confirmation
    
    CLI-->>Bot: Return response
    Bot-->>User: Send response
```

## Content Types Mind Map

```mermaid
mindmap
  root((CelesteCLI Content Types))
    Social Media
      Twitter
        tweet
        tweet_image
        tweet_thread
        quote_tweet
        reply_snark
      TikTok
        tiktok
      Discord
        discord
    Video Content
      YouTube
        title
        ytdesc
      Twitch
        title
    Special Content
      Goodnight
        goodnight
      Birthday
        birthday
      Art
        pixivpost
        skebreq
        alt_text
```

## Configuration Hierarchy

```mermaid
graph TD
    A[Command Line Args] --> B[Environment Variables]
    B --> C[Config Files]
    C --> D[Default Values]
    
    subgraph "Config Files"
        E[~/.celeste.cfg<br/>DigitalOcean Spaces]
        F[~/.celesteAI<br/>CelesteAI & Venice.ai]
        G[personality.yml<br/>Personality Rules]
        H[scaffolding.json<br/>Prompt Templates]
    end
    
    subgraph "Environment Variables"
        I[CELESTE_USER_ID]
        J[CELESTE_PLATFORM]
        K[CELESTE_OVERRIDE_ENABLED]
        L[CELESTE_PGP_SIGNATURE]
    end
    
    C --> E
    C --> F
    C --> G
    C --> H
    
    B --> I
    B --> J
    B --> K
    B --> L
    
    style A fill:#e1f5fe
    style B fill:#f3e5f5
    style C fill:#e8f5e8
    style D fill:#fff3e0
```

## Error Handling Flow

```mermaid
graph TD
    A[Error Occurs] --> B{Error Type}
    
    B -->|Configuration| C[Load Defaults]
    B -->|API Error| D[Show Error Message]
    B -->|Network Error| E[Retry Logic]
    B -->|Parse Error| F[Exit with Error]
    
    C --> G[Continue Execution]
    D --> H[Exit with Error Code]
    E --> I{Retry Successful?}
    I -->|Yes| G
    I -->|No| H
    F --> H
    
    G --> J[Success]
    H --> K[Failure]
    
    style A fill:#ffebee
    style C fill:#e8f5e8
    style D fill:#ffcdd2
    style E fill:#fff3e0
    style F fill:#ffcdd2
    style G fill:#c8e6c9
    style H fill:#ffcdd2
    style I fill:#fff3e0
    style J fill:#c8e6c9
    style K fill:#ffcdd2
```

## Usage Examples

### Standard Content Generation
```bash
./celestecli --type tweet --game "NIKKE" --tone "teasing"
```

### NSFW Mode
```bash
./celestecli --nsfw --type tweet --game "NIKKE" --tone "explicit"
```

### Bot Integration
```bash
CELESTE_USER_ID="discord_user_123" CELESTE_PLATFORM="discord" ./celestecli --type tweet --game "NIKKE" --tone "teasing" --sync
```

### Override Commands
```bash
CELESTE_OVERRIDE_ENABLED="true" CELESTE_PGP_SIGNATURE="kusanagi-abyss-override" ./celestecli --type tweet --game "NIKKE" --tone "explicit"
```

## Key Features

- **14 Content Types**: Comprehensive content generation support
- **User Isolation**: Per-user conversation tracking for bots
- **Override Functionality**: PGP-signed override commands
- **NSFW Mode**: Venice.ai integration for uncensored content
- **External Scaffolding**: JSON-based prompt templates
- **OpenSearch Integration**: S3 upload for RAG
- **Comprehensive Error Handling**: Graceful error management
- **Bot Integration**: Discord and Twitch bot support
