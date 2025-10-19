# CelesteCLI Function Calling Flow

## Mermaid Diagram

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
    
    %% Configuration Loading
    subgraph Config["Configuration Loading"]
        DD[~/.celeste.cfg] --> EE[Environment Variables]
        EE --> FF[Default Values]
        GG[~/.celesteAI] --> HH[Venice.ai Config]
        II[personality.yml] --> JJ[Personality Rules]
        KK[scaffolding.json] --> LL[Prompt Templates]
    end
    
    %% User Isolation
    subgraph UserIsolation["User Isolation System"]
        MM[CELESTE_USER_ID] --> NN[Per-User Context]
        OO[CELESTE_PLATFORM] --> PP[Platform Metadata]
        QQ[CELESTE_CHANNEL_ID] --> RR[Channel Tracking]
        SS[CELESTE_GUILD_ID] --> TT[Guild Tracking]
        UU[CELESTE_MESSAGE_ID] --> VV[Message Tracking]
    end
    
    %% Override System
    subgraph Override["Override System"]
        WW[CELESTE_OVERRIDE_ENABLED] --> XX[Override Permissions]
        YY[CELESTE_PGP_SIGNATURE] --> ZZ[PGP Verification]
        ZZ --> AAA[Abyssal Law Bypass]
    end
    
    %% Content Types
    subgraph ContentTypes["Content Generation Types"]
        BBB[tweet] --> CCC[Twitter Post]
        DDD[tweet_image] --> EEE[Twitter with Image Credit]
        FFF[tweet_thread] --> GGG[Multi-part Thread]
        HHH[title] --> III[YouTube/Twitch Title]
        JJJ[ytdesc] --> KKK[YouTube Description]
        LLL[tiktok] --> MMM[TikTok Caption]
        NNN[discord] --> OOO[Discord Announcement]
        PPP[goodnight] --> QQQ[Goodnight Tweet]
        RRR[pixivpost] --> SSS[Pixiv Caption]
        TTT[skebreq] --> UUU[Skeb Commission]
        VVV[quote_tweet] --> WWW[Quote Tweet Response]
        XXX[reply_snark] --> YYY[Snarky Reply]
        ZZZ[birthday] --> AAAA[Birthday Message]
        BBBB[alt_text] --> CCCC[Image Alt Text]
    end
    
    %% API Integration
    subgraph APIIntegration["API Integration"]
        DDDD[CelesteAI API] --> EEEE[Standard Content Generation]
        FFFF[Venice.ai API] --> GGGG[NSFW Content Generation]
        HHHH[IGDB API] --> IIII[Game Metadata]
        JJJJ[S3/DigitalOcean] --> KKKK[Conversation Storage]
    end
    
    %% Error Handling
    subgraph ErrorHandling["Error Handling"]
        LLLL[Configuration Error] --> MMMM[Warning + Defaults]
        NNNN[API Error] --> OOOO[Error Message + Exit]
        PPPP[Network Error] --> QQQQ[Retry Logic]
        RRRR[Parse Error] --> SSSS[Error Message + Exit]
    end
    
    %% Styling
    classDef startEnd fill:#e1f5fe,stroke:#01579b,stroke-width:2px
    classDef process fill:#f3e5f5,stroke:#4a148c,stroke-width:2px
    classDef decision fill:#fff3e0,stroke:#e65100,stroke-width:2px
    classDef config fill:#e8f5e8,stroke:#2e7d32,stroke-width:2px
    classDef api fill:#fce4ec,stroke:#c2185b,stroke-width:2px
    classDef error fill:#ffebee,stroke:#c62828,stroke-width:2px
    
    class A,AA startEnd
    class B,E,F,G,H,I,K,L,M,N,O,P,Q,R,S,T,U,V,W,X,Y,Z,BB,CC process
    class C,J,U,Y decision
    class DD,EE,FF,GG,HH,II,JJ,KK,LL config
    class DDDD,EEEE,FFFF,GGGG,HHHH,IIII,JJJJ,KKKK api
    class LLLL,MMMM,NNNN,OOOO,PPPP,QQQQ,RRRR,SSSS error
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

This flowchart provides a comprehensive overview of the CelesteCLI function calling architecture and can be used in documentation to help users understand the system's operation.
