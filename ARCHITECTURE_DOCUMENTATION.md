# CelesteCLI Architecture Documentation

## Overview

This document provides comprehensive architectural diagrams and flowcharts for the CelesteCLI tool, showing the function calling flow, bot integration, and system architecture.

## Main Function Flow

```mermaid
flowchart TD
    A[Start: ./celestecli] --> B[Parse Command Line Flags]
    B --> C{Override Permissions Check}
    C -->|Override Enabled| D[🔓 Override Mode Active]
    C -->|Normal Mode| E[Load Personality Configuration]
    D --> E
    
    E --> F[Load Scaffolding Configuration]
    F --> G[Build Prompt with Scaffolding]
    G --> H[Add Personality Prompt]
    H --> I[Add Context & Media if provided]
    
    I --> J{NSFW Mode?}
    J -->|Yes| K[Load Venice.ai Config]
    J -->|No| L[Load CelesteAI Config]
    
    K --> M[🔥 NSFW Mode: Venice.ai Request]
    M --> N[Venice.ai API Call]
    N --> O[Return Uncensored Response]
    
    L --> P[Get API Credentials]
    P --> Q[Build Chat Request]
    Q --> R[Add Function Calls if needed]
    R --> S[Send HTTP Request to CelesteAI]
    
    S --> T[Parse Response]
    T --> U{Debug Mode?}
    U -->|Yes| V[Show Raw JSON]
    U -->|No| W[Extract Content]
    
    W --> X[Display Response]
    X --> Y{Sync Flag?}
    Y -->|Yes| Z[Create Conversation Entry]
    Y -->|No| AA[End]
    
    Z --> BB[Upload to S3/DigitalOcean Spaces]
    BB --> CC[Log Success/Failure]
    CC --> AA
    
    style A fill:#e1f5fe,stroke:#01579b,stroke-width:2px
    style D fill:#ffcdd2,stroke:#c62828,stroke-width:2px
    style M fill:#f8bbd9,stroke:#ad1457,stroke-width:2px
    style AA fill:#e1f5fe,stroke:#01579b,stroke-width:2px
```

## Bot Integration Architecture

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

## Content Types Overview

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

## System Architecture

```mermaid
graph TB
    subgraph "User Input"
        A[Command Line Args]
        B[Environment Variables]
        C[Config Files]
    end
    
    subgraph "Configuration Layer"
        D[~/.celeste.cfg]
        E[~/.celesteAI]
        F[personality.yml]
        G[scaffolding.json]
    end
    
    subgraph "Core Processing"
        H[Flag Parsing]
        I[Override Check]
        J[Config Loading]
        K[Prompt Building]
    end
    
    subgraph "API Layer"
        L[CelesteAI API]
        M[Venice.ai API]
        N[IGDB API]
        O[S3/DigitalOcean]
    end
    
    subgraph "Output Processing"
        P[Response Parsing]
        Q[Content Extraction]
        R[Error Handling]
        S[User Feedback]
    end
    
    subgraph "Storage Layer"
        T[Conversation Entry]
        U[S3 Upload]
        V[Metadata Creation]
        W[Audit Logging]
    end
    
    A --> H
    B --> H
    C --> H
    H --> I
    I --> J
    J --> K
    K --> L
    K --> M
    L --> P
    M --> P
    P --> Q
    Q --> R
    R --> S
    S --> T
    T --> U
    U --> V
    V --> W
    
    D --> J
    E --> J
    F --> J
    G --> J
    
    style A fill:#e3f2fd
    style B fill:#e3f2fd
    style C fill:#e3f2fd
    style D fill:#f3e5f5
    style E fill:#f3e5f5
    style F fill:#f3e5f5
    style G fill:#f3e5f5
    style H fill:#fff3e0
    style I fill:#fff3e0
    style J fill:#fff3e0
    style K fill:#fff3e0
    style L fill:#e8f5e8
    style M fill:#fce4ec
    style N fill:#e8f5e8
    style O fill:#e8f5e8
    style P fill:#f1f8e9
    style Q fill:#f1f8e9
    style R fill:#ffebee
    style S fill:#f1f8e9
    style T fill:#e0f2f1
    style U fill:#e0f2f1
    style V fill:#e0f2f1
    style W fill:#e0f2f1
```

## Function Call Hierarchy

### 1. **Main Entry Point**
- `main()` - Entry point for CLI application
- `flag.Parse()` - Parse command line arguments
- `checkOverridePermissions()` - Check for override permissions

### 2. **Configuration Loading**
- `loadPersonalityConfig()` - Load personality.yml
- `loadScaffoldingConfig()` - Load scaffolding.json
- `loadVeniceConfig()` - Load Venice.ai configuration
- `readCelesteConfig()` - Load CelesteAI configuration

### 3. **Prompt Construction**
- `getPersonalityPrompt()` - Get personality-specific prompt
- `getScaffoldPrompt()` - Get content-type specific prompt
- Prompt assembly with context and media

### 4. **API Request Handling**
- `makeVeniceRequest()` - Handle Venice.ai requests (NSFW mode)
- Standard HTTP client for CelesteAI requests
- `createConversationEntry()` - Create conversation metadata

### 5. **Response Processing**
- JSON parsing and content extraction
- Debug mode handling
- Error handling and user feedback

### 6. **Storage Integration**
- `uploadConversationToS3()` - Upload to DigitalOcean Spaces
- `loadS3Config()` - Load S3 configuration
- Conversation metadata creation

### 7. **Utility Functions**
- `determineIntent()` - Determine conversation intent
- `determinePlatform()` - Determine platform context
- `determineSentiment()` - Analyze sentiment
- `extractTopics()` - Extract conversation topics
- `generateTags()` - Generate content tags
- `verifyPGPSignature()` - Verify PGP signatures

## Key Features

### **User Isolation System**
- Per-user conversation tracking via `CELESTE_USER_ID`
- Platform-specific metadata capture
- Channel, guild, and message ID tracking

### **Override Functionality**
- PGP signature verification for override commands
- Environment variable-based permission checking
- Audit logging for all override commands

### **Content Generation**
- 14 different content types supported
- External scaffolding system for easy template updates
- Personality-driven response generation

### **API Integration**
- CelesteAI for standard content generation
- Venice.ai for NSFW content generation
- IGDB for game metadata
- S3/DigitalOcean Spaces for conversation storage

### **Error Handling**
- Comprehensive error handling throughout
- Graceful fallbacks for configuration issues
- User-friendly error messages

## Usage Examples

### **Standard Content Generation**
```bash
./celestecli --type tweet --game "NIKKE" --tone "teasing"
```

### **NSFW Mode**
```bash
./celestecli --nsfw --type tweet --game "NIKKE" --tone "explicit"
```

### **Bot Integration**
```bash
CELESTE_USER_ID="discord_user_123" CELESTE_PLATFORM="discord" ./celestecli --type tweet --game "NIKKE" --tone "teasing" --sync
```

### **Override Commands**
```bash
CELESTE_OVERRIDE_ENABLED="true" CELESTE_PGP_SIGNATURE="kusanagi-abyss-override" ./celestecli --type tweet --game "NIKKE" --tone "explicit"
```

## ASCII Art Flow Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                           CelesteCLI Architecture                              │
└─────────────────────────────────────────────────────────────────────────────────┘

┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   User      │    │   Bot       │    │  Override   │    │   Config    │
│  Input      │    │Integration  │    │  Commands   │    │   Files     │
└─────┬───────┘    └─────┬───────┘    └─────┬───────┘    └─────┬───────┘
      │                  │                  │                  │
      └──────────────────┼──────────────────┼──────────────────┘
                         │                  │
                    ┌────▼──────────────────▼────┐
                    │      Flag Parsing          │
                    │   ┌─────────────────────┐   │
                    │   │ --type, --game,     │   │
                    │   │ --tone, --nsfw,    │   │
                    │   │ --sync, --debug    │   │
                    │   └─────────────────────┘   │
                    └─────────────┬───────────────┘
                                  │
                    ┌─────────────▼───────────────┐
                    │    Configuration Loading    │
                    │  ┌─────────────────────────┐│
                    │  │ ~/.celeste.cfg          ││
                    │  │ ~/.celesteAI            ││
                    │  │ personality.yml         ││
                    │  │ scaffolding.json        ││
                    │  └─────────────────────────┘│
                    └─────────────┬───────────────┘
                                  │
                    ┌─────────────▼───────────────┐
                    │      Override Check         │
                    │  ┌─────────────────────────┐│
                    │  │ CELESTE_OVERRIDE_ENABLED││
                    │  │ CELESTE_PGP_SIGNATURE   ││
                    │  │ 🔓 Abyssal Law Bypass   ││
                    │  └─────────────────────────┘│
                    └─────────────┬───────────────┘
                                  │
                    ┌─────────────▼───────────────┐
                    │      Prompt Building        │
                    │  ┌─────────────────────────┐│
                    │  │ Personality + Scaffold  ││
                    │  │ + Context + Media       ││
                    │  └─────────────────────────┘│
                    └─────────────┬───────────────┘
                                  │
                    ┌─────────────▼───────────────┐
                    │      Mode Selection         │
                    │  ┌─────────────────────────┐│
                    │  │     NSFW Mode?          ││
                    │  │         │               ││
                    │  │    Yes ──┼── No          ││
                    │  │         │               ││
                    │  │  Venice.ai    CelesteAI ││
                    │  └─────────────────────────┘│
                    └─────────────┬───────────────┘
                                  │
                    ┌─────────────▼───────────────┐
                    │       API Request          │
                    │  ┌─────────────────────────┐│
                    │  │ HTTP POST to API         ││
                    │  │ Bearer Token Auth        ││
                    │  │ JSON Payload             ││
                    │  └─────────────────────────┘│
                    └─────────────┬───────────────┘
                                  │
                    ┌─────────────▼───────────────┐
                    │     Response Processing     │
                    │  ┌─────────────────────────┐│
                    │  │ Parse JSON Response     ││
                    │  │ Extract Content         ││
                    │  │ Handle Errors            ││
                    │  └─────────────────────────┘│
                    └─────────────┬───────────────┘
                                  │
                    ┌─────────────▼───────────────┐
                    │       Output Display        │
                    │  ┌─────────────────────────┐│
                    │  │ Show Response to User   ││
                    │  │ Debug Mode (Optional)   ││
                    │  └─────────────────────────┘│
                    └─────────────┬───────────────┘
                                  │
                    ┌─────────────▼───────────────┐
                    │      Sync to S3?            │
                    │  ┌─────────────────────────┐│
                    │  │ Create Conversation     ││
                    │  │ Entry with Metadata      ││
                    │  │ Upload to DigitalOcean  ││
                    │  │ Spaces for OpenSearch    ││
                    │  └─────────────────────────┘│
                    └─────────────┬───────────────┘
                                  │
                    ┌─────────────▼───────────────┐
                    │         Complete            │
                    │  ┌─────────────────────────┐│
                    │  │ ✅ Success              ││
                    │  │ 📊 Metrics Logged       ││
                    │  │ 🔄 Ready for Next       ││
                    │  └─────────────────────────┘│
                    └─────────────────────────────┘
```

This comprehensive documentation provides all the necessary diagrams and flowcharts for understanding the CelesteCLI architecture and can be used in setup documentation, README files, or technical documentation.
