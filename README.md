# Celeste CLI

A powerful command-line interface for interacting with CelesteAI, featuring content generation, image processing, and NSFW capabilities through Venice.ai integration.

## 🚀 Features

### Core Functionality
- **Content Generation**: Twitter posts, TikTok captions, YouTube descriptions, Discord announcements
- **Personality System**: YAML-based personality configuration with multiple personas
- **Game Integration**: IGDB API integration for game metadata
- **S3 Sync**: DigitalOcean Spaces integration for conversation storage
- **Bot Integration**: Discord/Twitch bot support with user isolation

### NSFW Mode (Venice.ai Integration)
- **Text Generation**: Uncensored content using `venice-uncensored` model
- **Image Generation**: NSFW image creation with `lustify-sdxl` model
- **Image Upscaling**: High-quality upscaling with fidelity controls
- **Image Editing**: Inpainting and signature removal
- **Smart Workflows**: Optimized 2-step process for small images

## 📦 Installation

### Prerequisites
- Go 1.19+
- Venice.ai API key (for NSFW features)
- DigitalOcean Spaces credentials (for S3 sync)

### Quick Install
```bash
git clone <repository>
cd celesteCLI
go build -o celestecli main.go scaffolding.go
./install.sh
```

### Manual Install
```bash
go build -o celestecli main.go scaffolding.go
cp celestecli ~/.local/bin/
chmod +x ~/.local/bin/celestecli
```

## ⚙️ Configuration

### CelesteAI Configuration (`~/.celesteAI`)
```bash
# CelesteAI API
endpoint=https://your-celeste-api-endpoint
api_key=your-api-key

# IGDB Integration
client_id=your-igdb-client-id
secret=your-igdb-client-secret

# NSFW Mode (Venice.ai)
venice_api_key=your-venice-api-key
venice_base_url=https://api.venice.ai/api/v1
venice_model=venice-uncensored
venice_upscaler=upscaler
```

### S3 Configuration (`~/.celeste.cfg`)
```bash
# DigitalOcean Spaces
endpoint=https://sfo3.digitaloceanspaces.com
bucket_name=whykusanagi
access_key_id=your-access-key
secret_access_key=your-secret-key
```

## 🎯 Usage

### Basic Content Generation
```bash
# Twitter post
celestecli --type tweet --game "NIKKE" --tone "lewd"

# TikTok caption
celestecli --type tiktok --tone "playful"

# YouTube description
celestecli --type ytdesc --game "Streaming" --tone "professional"
```

### NSFW Mode
```bash
# Uncensored text generation
celestecli --nsfw --context "Generate explicit content"

# Image generation
celestecli --nsfw --image --context "Generate NSFW image of Celeste"

# Image upscaling
celestecli --nsfw --upscale --image-path "image.png"

# Image editing (signature removal)
celestecli --nsfw --edit --image-path "image.png" --edit-prompt "remove signature"

# Optimized workflow for small images
celestecli --nsfw --edit --image-path "small_image.png" --edit-prompt "remove watermark" --upscale-first
```

### Advanced Options
```bash
# List available Venice.ai models
celestecli --nsfw --list-models

# Override model
celestecli --nsfw --model "wai-Illustrious" --image --context "Anime style"

# Custom output filename
celestecli --nsfw --image --output "my_image.png" --context "Custom filename"

# Preserve original size
celestecli --nsfw --edit --image-path "large_image.png" --edit-prompt "edit" --preserve-size
```

## 🎨 Image Processing Workflows

### Image Processing Pipeline

```mermaid
graph TD
    A[Image Input] --> B{Image Size Check}
    
    B -->|< 1024x1024| C[Small Image Workflow]
    B -->|≥ 1024x1024| D[Large Image Workflow]
    
    C --> E[getImageDimensions]
    E --> E1[runCommand file]
    E1 --> E2[Parse Dimensions]
    E2 --> F[makeVeniceUpscaleRequest]
    F --> F1[Read Image File]
    F1 --> F2[Base64 Encode]
    F2 --> F3[POST /image/upscale]
    F3 --> F4[Return Raw Image Data]
    F4 --> G[makeVeniceEditRequest]
    G --> G1[Read Upscaled Image]
    G1 --> G2[Base64 Encode]
    G2 --> G3[POST /image/edit]
    G3 --> G4[Return Edited Image]
    G4 --> H[saveImageData]
    H --> H1[Generate Filename]
    H1 --> H2[Write File to Disk]
    
    D --> I{--upscale-first?}
    I -->|Yes| J[getImageDimensions]
    J --> J1[Check if < 1024x1024]
    J1 -->|Yes| K[makeVeniceUpscaleRequest]
    J1 -->|No| L[makeVeniceEditRequest]
    K --> K1[Upscale to 1024x1024]
    K1 --> L
    L --> L1[Edit at 1024x1024]
    L1 --> M[saveImageData]
    
    I -->|No| N[makeVeniceEditRequest]
    N --> N1[Read Original Image]
    N1 --> N2[Base64 Encode]
    N2 --> N3[POST /image/edit]
    N3 --> N4[Return Edited Image]
    N4 --> O{--preserve-size?}
    O -->|Yes| P[makeVeniceUpscaleRequest]
    O -->|No| Q[saveImageData]
    P --> P1[Upscale Back to Original Size]
    P1 --> Q
    
    H2 --> R[Final Output]
    M --> R
    Q --> R
    
    style A fill:#e1f5fe,stroke:#01579b,stroke-width:3px
    style C fill:#fff3e0,stroke:#e65100,stroke-width:2px
    style D fill:#f3e5f5,stroke:#4a148c,stroke-width:2px
    style R fill:#e8f5e8,stroke:#1b5e20,stroke-width:3px
```

### Standard Upscaling
- **Input**: Any image ≥256x256 pixels
- **Output**: 2x upscaled with quality preservation
- **Parameters**: Conservative settings for fidelity

### Smart Editing Workflow
- **Small Images** (<1024x1024): Uses `--upscale-first` (2 API calls)
  1. Upscale to 1024x1024
  2. Edit at native size
- **Large Images** (≥1024x1024): Uses standard edit workflow
- **Result**: High-quality edited images without distortion

### Quality Controls
- **Enhancement Creativity**: 0.0-1.0 (lower = more faithful)
- **Replication Level**: 0.0-1.0 (higher = more faithful)
- **Enhancement Prompt**: Custom instructions for upscaling

## 🤖 Bot Integration

### Environment Variables
```bash
export CELESTE_USER_ID="user123"
export CELESTE_PLATFORM="discord"
export CELESTE_CHANNEL_ID="channel123"
export CELESTE_GUILD_ID="guild123"
export CELESTE_OVERRIDE_ENABLED="true"
export CELESTE_PGP_SIGNATURE="signature"
```

### User Isolation
- Each user gets separate conversation contexts
- Platform-specific metadata tracking
- PGP signature verification for override commands

## 📊 S3 Integration & RAG

### Data Flow Architecture

```mermaid
graph TD
    A[CLI Request] --> B[createConversationEntry]
    B --> C[determineIntent]
    B --> D[determinePlatform]
    B --> E[determineSentiment]
    B --> F[extractTopics]
    B --> G[generateTags]
    
    C --> C1[Analyze Content Type]
    C1 --> C2[Map to Intent Categories]
    C2 --> H[ConversationEntry]
    
    D --> D1[Check Platform Context]
    D1 --> D2[Set Platform Metadata]
    D2 --> H
    
    E --> E1[Analyze Tone Keywords]
    E1 --> E2[Classify Sentiment]
    E2 --> H
    
    F --> F1[Extract Game Names]
    F1 --> F2[Extract Keywords]
    F2 --> F3[Generate Topic Tags]
    F3 --> H
    
    G --> G1[Combine All Metadata]
    G1 --> G2[Generate Search Tags]
    G2 --> H
    
    H --> I{--sync flag?}
    I -->|Yes| J[loadS3Config]
    J --> J1[Read ~/.celeste.cfg]
    J1 --> J2[Parse S3 Credentials]
    J2 --> K[createS3Session]
    K --> K1[Create AWS S3 Client]
    K1 --> K2[Configure Endpoint]
    K2 --> L[uploadConversationToS3]
    L --> L1[Generate S3 Key]
    L1 --> L2[Upload JSON to S3]
    L2 --> M[DigitalOcean Spaces]
    M --> M1[Store in s3://whykusanagi/celeste/]
    M1 --> N[OpenSearch RAG]
    N --> N1[Index Conversation Data]
    N1 --> N2[Enable Semantic Search]
    
    I -->|No| O[Local Processing]
    O --> O1[Skip S3 Upload]
    O1 --> O2[Return Response Only]
    
    style A fill:#e1f5fe,stroke:#01579b,stroke-width:3px
    style H fill:#fff3e0,stroke:#e65100,stroke-width:2px
    style M fill:#f3e5f5,stroke:#4a148c,stroke-width:2px
    style N fill:#e8f5e8,stroke:#1b5e20,stroke-width:2px
    style O2 fill:#ffebee,stroke:#c62828,stroke-width:2px
```

### API Endpoint Flow

```mermaid
graph TD
    A[celestecli Command] --> B{Mode Selection}
    
    B -->|Normal Mode| C[DigitalOcean API]
    C --> C1[POST /chat/completions]
    C1 --> C2[Model: celeste-ai]
    C2 --> C3[Response: Text Content]
    
    B -->|NSFW Mode| D[Venice.ai API]
    D --> E{Function Type}
    
    E -->|Text Generation| F[POST /chat/completions]
    F --> F1[Model: venice-uncensored]
    F1 --> F2[Response: Uncensored Text]
    
    E -->|Image Generation| G[POST /image/generate]
    G --> G1[Model: lustify-sdxl]
    G1 --> G2[Response: Image URL/Data]
    G2 --> G3[extractImageFromResponse]
    G3 --> G4[saveImageData]
    
    E -->|Image Upscaling| H[POST /image/upscale]
    H --> H1[Base64 Image Input]
    H1 --> H2[Response: Raw Image Data]
    H2 --> H3[saveImageData]
    
    E -->|Image Editing| I[POST /image/edit]
    I --> I1[Base64 Image + Prompt]
    I1 --> I2[Response: Edited Image]
    I2 --> I3[saveImageData]
    
    E -->|Model Listing| J[GET /models]
    J --> J1[Response: Available Models]
    J1 --> J2[Display Model List]
    
    C3 --> K[createConversationEntry]
    F2 --> K
    G4 --> L[Image Saved to Disk]
    H3 --> L
    I3 --> L
    J2 --> M[Exit]
    
    K --> N{--sync?}
    N -->|Yes| O[S3 Upload]
    N -->|No| P[Local Response]
    
    style C fill:#e3f2fd,stroke:#1976d2,stroke-width:2px
    style D fill:#fff3e0,stroke:#f57c00,stroke-width:2px
    style L fill:#e8f5e8,stroke:#388e3c,stroke-width:2px
    style O fill:#f3e5f5,stroke:#7b1fa2,stroke-width:2px
    style P fill:#ffebee,stroke:#d32f2f,stroke-width:2px
```

### Conversation Storage
- **Format**: Structured JSON with intent, purpose, topics
- **Location**: `s3://whykusanagi/celeste/conversations/`
- **Metadata**: User ID, platform, sentiment, success tracking

### OpenSearch Integration
- **Purpose**: RAG (Retrieval-Augmented Generation)
- **Data Structure**: Intent-based organization
- **Benefits**: Contextual responses based on conversation history

## 🔧 Development

### Function Call Flow

```mermaid
graph TD
    A[main] --> B[flag.Parse]
    B --> C{--nsfw flag?}
    
    C -->|Yes| D[loadVeniceConfig]
    D --> E{Mode Type?}
    
    E -->|--list-models| F[listVeniceModels]
    F --> F1[GET /models]
    F1 --> F2[Parse VeniceModelsResponse]
    F2 --> F3[Display Available Models]
    
    E -->|--upscale| G[makeVeniceUpscaleRequest]
    G --> G1[Read Image File]
    G1 --> G2[Base64 Encode]
    G2 --> G3[POST /image/upscale]
    G3 --> G4[Return Raw Image Data]
    G4 --> G5[saveImageData]
    
    E -->|--edit| H{--upscale-first?}
    H -->|Yes| I[getImageDimensions]
    I --> J{Image < 1024x1024?}
    J -->|Yes| K[makeVeniceUpscaleRequest]
    K --> L[makeVeniceEditRequest]
    J -->|No| M[makeVeniceEditRequest]
    L --> N[saveImageData]
    M --> N
    
    H -->|No| O[makeVeniceEditRequest]
    O --> P{--preserve-size?}
    P -->|Yes| Q[makeVeniceUpscaleRequest]
    P -->|No| R[saveImageData]
    Q --> R
    
    E -->|--image| S[makeVeniceImageRequest]
    S --> S1[POST /image/generate]
    S1 --> S2[extractImageFromResponse]
    S2 --> S3[saveImageData]
    
    E -->|text| T[makeVeniceRequest]
    T --> T1[POST /chat/completions]
    T1 --> T2[Return Text Response]
    
    C -->|No| U[readCelesteConfig]
    U --> V[loadPersonalityConfig]
    V --> W[getPersonalityPrompt]
    W --> X{fetchIGDBGameInfo?}
    X -->|Yes| Y[IGDB API Call]
    X -->|No| Z[Build Chat Request]
    Y --> Z
    Z --> AA[HTTP POST to DigitalOcean]
    AA --> BB[createConversationEntry]
    BB --> CC[determineIntent]
    BB --> DD[determinePlatform]
    BB --> EE[determineSentiment]
    BB --> FF[extractTopics]
    BB --> GG[generateTags]
    CC --> HH[ConversationEntry]
    DD --> HH
    EE --> HH
    FF --> HH
    GG --> HH
    HH --> II{--sync?}
    II -->|Yes| JJ[loadS3Config]
    JJ --> KK[createS3Session]
    KK --> LL[uploadConversationToS3]
    LL --> MM[DigitalOcean Spaces]
    II -->|No| NN[Output Response]
    
    style A fill:#e1f5fe,stroke:#01579b,stroke-width:3px
    style D fill:#fff3e0,stroke:#e65100,stroke-width:2px
    style U fill:#f3e5f5,stroke:#4a148c,stroke-width:2px
    style G5 fill:#e8f5e8,stroke:#1b5e20,stroke-width:2px
    style S3 fill:#e8f5e8,stroke:#1b5e20,stroke-width:2px
    style N fill:#e8f5e8,stroke:#1b5e20,stroke-width:2px
    style R fill:#e8f5e8,stroke:#1b5e20,stroke-width:2px
    style T2 fill:#e8f5e8,stroke:#1b5e20,stroke-width:2px
    style NN fill:#e8f5e8,stroke:#1b5e20,stroke-width:2px
```

### Technical Architecture

```mermaid
graph TB
    subgraph "CLI Interface Layer"
        A[celestecli main] --> B[flag.Parse]
        B --> C[Configuration Loading]
        C --> D[readCelesteConfig]
        C --> E[loadPersonalityConfig]
        C --> F[loadVeniceConfig]
        C --> G[loadS3Config]
    end
    
    subgraph "Configuration Files"
        H[~/.celesteAI<br/>DigitalOcean API]
        I[~/.celeste.cfg<br/>S3 Credentials]
        J[personality.yml<br/>Celeste Personality]
    end
    
    subgraph "Normal Mode Processing"
        K[getPersonalityPrompt] --> L[fetchIGDBGameInfo]
        L --> M[Build Chat Request]
        M --> N[HTTP POST to DigitalOcean]
        N --> O[createConversationEntry]
        O --> P[determineIntent]
        O --> Q[determinePlatform]
        O --> R[determineSentiment]
        O --> S[extractTopics]
        O --> T[generateTags]
    end
    
    subgraph "NSFW Mode (Venice.ai)"
        U[makeVeniceRequest] --> V[POST /chat/completions]
        W[makeVeniceImageRequest] --> X[POST /image/generate]
        Y[makeVeniceUpscaleRequest] --> Z[POST /image/upscale]
        AA[makeVeniceEditRequest] --> BB[POST /image/edit]
        CC[listVeniceModels] --> DD[GET /models]
    end
    
    subgraph "Image Processing Pipeline"
        EE[getImageDimensions] --> FF{Image Size Check}
        FF -->|Small| GG[makeVeniceUpscaleRequest]
        FF -->|Large| HH[makeVeniceEditRequest]
        GG --> HH
        HH --> II[saveImageData]
        JJ[extractImageFromResponse] --> II
    end
    
    subgraph "Data Persistence Layer"
        KK[uploadConversationToS3] --> LL[createS3Session]
        LL --> MM[DigitalOcean Spaces]
        MM --> NN[OpenSearch RAG]
    end
    
    subgraph "Bot Integration Layer"
        OO[Discord Bot] --> PP[User Isolation]
        QQ[Twitch Bot] --> PP
        PP --> RR[verifyPGPSignature]
        RR --> SS[checkOverridePermissions]
    end
    
    A --> K
    A --> U
    A --> W
    A --> Y
    A --> AA
    A --> CC
    A --> KK
    A --> OO
    A --> QQ
    
    D --> H
    E --> J
    F --> H
    G --> I
    
    style A fill:#e1f5fe,stroke:#01579b,stroke-width:3px
    style U fill:#fff3e0,stroke:#e65100,stroke-width:2px
    style W fill:#fff3e0,stroke:#e65100,stroke-width:2px
    style Y fill:#fff3e0,stroke:#e65100,stroke-width:2px
    style AA fill:#fff3e0,stroke:#e65100,stroke-width:2px
    style CC fill:#fff3e0,stroke:#e65100,stroke-width:2px
    style KK fill:#f3e5f5,stroke:#4a148c,stroke-width:2px
    style OO fill:#e8f5e8,stroke:#1b5e20,stroke-width:2px
    style QQ fill:#e8f5e8,stroke:#1b5e20,stroke-width:2px
```

### Project Structure
```
celesteCLI/
├── main.go              # Core CLI application
├── scaffolding.go       # Prompt template loader
├── scaffolding.json     # Prompt templates
├── personality.yml      # Celeste personality configuration
├── go.mod              # Go dependencies
├── install.sh          # Installation script
└── README.md           # This file
```

### Dependencies
- `github.com/aws/aws-sdk-go` - S3 integration
- `github.com/sashabaranov/go-openai` - Venice.ai integration
- `gopkg.in/yaml.v3` - YAML configuration parsing

### Building
```bash
go mod tidy
go build -o celestecli main.go scaffolding.go
```

## 🎭 Personality System

### Configuration (`personality.yml`)
- **Personas**: Multiple character modes (stream, moderation, etc.)
- **Content Types**: Specialized templates for different platforms
- **Voice Rules**: Tone and style guidelines
- **Safety Modes**: Content filtering and guardrails

### Available Personas
- `celeste_stream` - Default streaming persona
- `celeste_ad_read` - Advertisement reading
- `celeste_moderation_warning` - Discord moderation

## 🚨 NSFW Mode Details

### Venice.ai Models
- **Text**: `venice-uncensored` - Uncensored text generation
- **Images**: `lustify-sdxl` - NSFW image generation
- **Anime**: `wai-Illustrious` - Anime-style generation
- **Upscaling**: `upscaler` - High-quality upscaling

### API Endpoints
- `/image/generate` - Image generation
- `/image/upscale` - Image upscaling
- `/image/edit` - Image editing/inpainting
- `/models` - List available models

### Quality Controls
- **Conservative Settings**: 0.05 creativity, 0.9 replication
- **Fidelity Prompts**: "preserve original details exactly"
- **Smart Workflows**: Automatic optimization based on image size

## 🔍 Troubleshooting

### Common Issues
1. **Venice.ai API errors**: Check API key and endpoint
2. **S3 sync failures**: Verify DigitalOcean Spaces credentials
3. **Image dimension errors**: Ensure images meet minimum requirements (256x256)
4. **Permission errors**: Check file permissions and PATH configuration

### Debug Mode
```bash
celestecli --debug --type tweet --context "Debug output"
```

## 📈 Performance

### API Call Optimization
- **Standard Edit**: 1 API call
- **Upscale-First**: 2 API calls (optimized)
- **Previous Workflow**: 3 API calls (deprecated)

### Timing Examples
- **Text Generation**: ~2-5 seconds
- **Image Generation**: ~10-15 seconds
- **Image Upscaling**: ~8-12 seconds
- **Smart Editing**: ~14-20 seconds

## 🔒 Security

### PGP Signature Verification
- Override commands require PGP signatures
- Keybase integration for signature verification
- Environment variable configuration

### Content Safety
- Platform-specific content filtering
- Age-gated content handling
- Moderation capabilities for Discord/Twitch

## 📝 License

This project is part of the CelesteAI ecosystem. See individual component licenses for details.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## 📞 Support

For issues and questions:
- Check the troubleshooting section
- Review configuration examples
- Test with debug mode enabled
