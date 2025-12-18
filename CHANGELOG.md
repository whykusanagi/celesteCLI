# Changelog

All notable changes to CelesteCLI will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.3.0] - 2025-12-18

### Added
- **IPFS Integration** - Decentralized content management
  - Upload and download content via IPFS (returns CID)
  - Pin management (pin, unpin, list pins)
  - Multi-provider support (Infura, Pinata, custom nodes)
  - Gateway URL generation for public access
  - Official go-ipfs-http-client library integration
- **Alchemy Blockchain API** - Comprehensive blockchain data access
  - Wallet operations: ETH/token balances, transaction history, asset transfers
  - Token data: Real-time metadata and comprehensive token information
  - NFT APIs: Query NFTs by owner, metadata, collection info
  - Transaction monitoring: Gas prices, transaction receipts, block information
  - Multi-network support: Ethereum, Arbitrum, Optimism, Polygon, Base (mainnet + testnets)
  - JSON-RPC interface with proper error handling
- **Blockchain Monitoring** - Real-time blockchain event tracking
  - Watch addresses for new transactions across multiple blocks
  - Get latest block information with transaction details
  - Query specific blocks by number (hex or decimal)
  - Asset transfer tracking (external, internal, ERC20, ERC721, ERC1155)
  - Network-specific monitoring with configurable poll intervals
- **Modern Crypto Utilities**
  - Ethereum address validation using go-ethereum (EIP-55 checksumming)
  - Wei ↔ Ether ↔ Gwei conversion helpers with big.Int precision
  - Production-ready rate limiting using golang.org/x/time/rate
  - Multi-network URL construction and validation
  - Chain ID support for all major networks
- **Enhanced Configuration System**
  - Network-specific settings for L2 support
  - Environment variable overrides for CI/CD (`CELESTE_IPFS_API_KEY`, `CELESTE_ALCHEMY_API_KEY`)
  - Flexible provider configuration (Infura, Pinata, custom endpoints)
  - Crypto-specific config fields in config.json and skills.json
  - ConfigLoader interface with GetIPFSConfig(), GetAlchemyConfig(), GetBlockmonConfig()

### Changed
- Upgraded to modern production-grade Go crypto libraries:
  - `github.com/ethereum/go-ethereum@v1.16.7` - Official Ethereum Go implementation
  - `github.com/ipfs/go-ipfs-http-client@v0.7.0` - Official IPFS HTTP client
  - `github.com/ipfs/go-cid@v0.6.0` - Content Identifier handling
  - `golang.org/x/time@v0.14.0` - Token bucket rate limiting
- Improved error handling for external API integrations
- Enhanced skills.json structure for crypto service configuration
- Better address normalization with proper checksum validation

### Documentation
- Added `docs/IPFS_SETUP.md` - Infura IPFS configuration guide
- Added `docs/ALCHEMY_SETUP.md` - Alchemy API setup and usage
- Added `docs/BLOCKCHAIN_MONITORING.md` - Real-time monitoring guide

## [1.1.0] - 2025-12-14

### Added
- **One-shot CLI commands** for all features (context, stats, export, session, config, skills)
  - Execute any command without entering TUI: `./celeste context`, `./celeste stats`
  - Direct skill execution: `./celeste skill <name> [--args]`
  - Comprehensive skill testing with `./celeste skill generate_uuid`, etc.
- **Context Management System**
  - Token usage tracking with input/output breakdown
  - Retroactive token calculation for session history
  - Context window monitoring and warnings
  - Auto-summarization when approaching limits
- **Enhanced Session Persistence**
  - Message persistence across sessions
  - Session metadata tracking (token counts, model info)
  - Improved session loading and restoration
- Interactive model selector with arrow key navigation
- Flickering corruption animation for stats dashboard
- GitHub Actions CI/CD pipeline
- Comprehensive test coverage
- Security vulnerability scanning
- Cross-platform build support

### Fixed
- **Token counting** - Now correctly displays input/output token breakdown
- **All 18 skills** - 100% functional from CLI one-shot commands:
  - Type conversion for numeric arguments (length, value, amount)
  - Parameter name corrections (encoded, text, from_timezone, etc.)
  - Weather skill accepts both string and numeric zip codes
- Session persistence and provider detection issues
- Code formatting issues
- Dependency version compatibility

### Changed
- Improved documentation structure
- Enhanced error handling
- Model selector with arrow key navigation
- Stats dashboard with corruption animation effects

### Documentation
- Added `ONESHOT_COMMANDS.md` - Complete CLI command reference
- Added `docs/TEST_RESULTS.md` - Test verification results for all skills
- Added corruption aesthetic validation guides
- Added brand system documentation (migrated to corrupted-theme package)

## [1.0.2] - 2025-12-03

### Added
- **Bubble Tea TUI**: Complete rewrite with flicker-free terminal UI
  - Scrollable chat viewport with PgUp/PgDown navigation
  - Input history with arrow key navigation
  - Real-time skills panel showing execution status
  - Corrupted theme styling (pink/purple aesthetic)
- **Named Configurations**: Multi-profile config support
  - `celeste -config openai chat` for OpenAI
  - `celeste -config grok chat` for xAI/Grok
  - Template system for quick config creation
- **Skills System**: OpenAI function calling support
  - Tarot reading (3-card and Celtic Cross)
  - NSFW mode (Venice.ai integration)
  - Content generation (Twitter, TikTok, YouTube, Discord)
  - Image generation (Venice.ai)
  - Weather lookup
  - Unit/timezone/currency converters
  - Hash/Base64/UUID/Password generators
  - QR code generation
  - Twitch live status checking
  - YouTube video lookup
  - Reminders and notes
- **Session Management**: Conversation persistence
  - Auto-save and resume sessions
  - Session listing and loading
  - Message history with timestamps
- **Simulated Typing**: Smooth text rendering
  - Configurable typing speed
  - Corruption effects during typing
  - Better UX for streamed responses

### Changed
- **Architecture**: Modular package structure
  - `cmd/Celeste/tui/` - Bubble Tea components
  - `cmd/Celeste/llm/` - LLM client
  - `cmd/Celeste/config/` - Configuration management
  - `cmd/Celeste/skills/` - Skills registry and execution
  - `cmd/Celeste/prompts/` - System prompts
- **Configuration**: JSON-based config system
  - Migrated from `.celesteAI` to `~/.celeste/config.json`
  - Separate `secrets.json` for sensitive data
  - Environment variable override support
- **Binary Name**: Renamed from `celestecli` to `Celeste`

### Removed
- Legacy main_old.go (3,481 lines)
- Old configuration format
- Deprecated Python utilities

### Fixed
- API key exposure in error messages
- Config file permission issues
- Session not saving in some scenarios
- Weather skill error handling

### Security
- Added SECURITY.md with vulnerability reporting process
- Implemented secret masking in config display
- Improved API key storage with separate secrets file
- Added .gitignore protection for sensitive files

## [2.0.0] - Previous Release

### Added
- Initial CLI implementation
- Basic LLM integration
- Configuration file support

## [1.0.0] - Initial Release

### Added
- Basic functionality
- Simple command-line interface

---

## Release Links

- [Unreleased](https://github.com/whykusanagi/celesteCLI/compare/v1.3.0...HEAD)
- [1.3.0](https://github.com/whykusanagi/celesteCLI/compare/v1.1.0...v1.3.0)
- [1.1.0](https://github.com/whykusanagi/celesteCLI/compare/v1.0.2...v1.1.0)
- [1.0.2](https://github.com/whykusanagi/celesteCLI/releases/tag/v1.0.2)
- [1.0.0](https://github.com/whykusanagi/celesteCLI/releases/tag/v1.0.0)

## How to Update

### From 0.x to 1.0+

The configuration format has changed:

**Old format** (`.celesteAI`):
```
api_key=sk-xxx
base_url=https://api.openai.com/v1
```

**New format** (`~/.celeste/config.json`):
```json
{
  "api_key": "",
  "base_url": "https://api.openai.com/v1",
  "model": "gpt-4o-mini",
  "timeout": 60,
  "skip_persona_prompt": false,
  "simulate_typing": true,
  "typing_speed": 40
}
```

**Migration steps**:
1. Backup your old config: `cp ~/.celesteAI ~/.celesteAI.backup`
2. Install new version: `make install`
3. Run config migration: `celeste config --show` (auto-migrates)
4. Verify settings: `celeste config --show`
5. Test: `celeste chat`

### Breaking Changes in 1.0+

- Command name changed from `celestecli` to `Celeste`
- Config file location changed to `~/.celeste/`
- Session format incompatible with 2.x (will create new sessions)
- Some command flags renamed for consistency

---

## Support

- **Issues**: [GitHub Issues](https://github.com/whykusanagi/celesteCLI/issues)
- **Security**: See [SECURITY.md](SECURITY.md)
- **Contributing**: See [CONTRIBUTING.md](CONTRIBUTING.md)
