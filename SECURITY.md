# Security Policy

## Supported Versions

We release patches for security vulnerabilities. Currently supported versions:

| Version | Supported          |
| ------- | ------------------ |
| 3.0.x   | :white_check_mark: |
| < 3.0   | :x:                |

## Reporting a Vulnerability

We take the security of CelesteCLI seriously. If you believe you have found a security vulnerability, please report it to us as described below.

### Where to Report

**Please do NOT report security vulnerabilities through public GitHub issues.**

Instead, please report them via one of the following methods:

1. **Email**: Send details to the repository owner via GitHub
2. **GitHub Security Advisory**: Use the [Security Advisory](../../security/advisories/new) feature
3. **Private Disclosure**: Contact @whykusanagi directly on GitHub

### What to Include

Please include the following information in your report:

- Type of issue (e.g., buffer overflow, SQL injection, cross-site scripting, etc.)
- Full paths of source file(s) related to the manifestation of the issue
- The location of the affected source code (tag/branch/commit or direct URL)
- Any special configuration required to reproduce the issue
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue, including how an attacker might exploit it

### Response Timeline

- **Initial Response**: Within 48 hours of receipt
- **Triage**: Within 1 week
- **Fix Development**: Depends on severity and complexity
- **Public Disclosure**: After patch is released (coordinated disclosure)

## Security Best Practices

### For Users

1. **API Keys**: Never commit API keys to version control
   - Use environment variables: `CELESTE_API_KEY`
   - Or store in `~/.celeste/secrets.json` (ensure file permissions are `0600`)

2. **Configuration Files**: Protect your config files
   ```bash
   chmod 600 ~/.celeste/config.json
   chmod 600 ~/.celeste/secrets.json
   ```

3. **Update Regularly**: Keep CelesteCLI up to date
   ```bash
   git pull origin main
   make install
   ```

4. **Named Configs**: Use separate configs for different API providers
   ```bash
   celeste -config openai chat    # OpenAI key
   celeste -config grok chat       # xAI key
   ```

### For Developers

1. **Secret Management**
   - Use the `ConfigLoader` interface for accessing secrets
   - Never hardcode API keys or tokens
   - Use environment variables or config files only

2. **Dependency Updates**
   - Run `go mod tidy` regularly
   - Check for known vulnerabilities with `govulncheck`
   ```bash
   go install golang.org/x/vuln/cmd/govulncheck@latest
   govulncheck ./...
   ```

3. **Input Validation**
   - Validate all user inputs before processing
   - Sanitize data before logging or displaying
   - Use prepared statements for any database queries

4. **Error Handling**
   - Don't expose sensitive information in error messages
   - Log errors securely (avoid logging secrets)
   - Return generic error messages to users

## Known Security Considerations

### API Key Exposure

CelesteCLI handles multiple API keys:
- OpenAI API key
- Venice.ai API key
- Tarot function auth token
- Twitter/YouTube API credentials

**Mitigation**:
- Keys are stored in separate `secrets.json`
- Keys are masked in `config --show` output
- `.gitignore` excludes all config files

### LLM Prompt Injection

As an LLM-based tool, CelesteCLI may be vulnerable to prompt injection attacks.

**Mitigation**:
- System prompts are isolated from user input
- Skills have defined schemas
- No arbitrary code execution from LLM responses

### Third-Party Dependencies

CelesteCLI relies on several third-party libraries.

**Mitigation**:
- Dependencies are pinned in `go.mod`
- Regular security audits with `govulncheck`
- Minimal dependency surface (6 direct dependencies)

## Security Update Process

1. Vulnerability is reported and confirmed
2. Severity is assessed (Critical/High/Medium/Low)
3. Fix is developed and tested
4. Security advisory is drafted
5. Patch is released with security notes
6. Public disclosure after users have time to update

## Disclosure Policy

We follow **coordinated disclosure**:

- Security researchers are credited (with permission)
- We request 90 days before public disclosure
- Critical vulnerabilities may be patched faster
- CVE IDs will be requested for significant issues

## Scope

### In Scope

- Authentication/authorization bypasses
- API key exposure or theft
- Code execution vulnerabilities
- Prompt injection leading to data exfiltration
- Dependency vulnerabilities

### Out of Scope

- Social engineering attacks
- Physical attacks
- DDoS attacks
- Issues in third-party dependencies (report to upstream)
- Browser/client-side issues (this is a CLI tool)

## Recognition

We appreciate security researchers who help keep CelesteCLI safe. Contributors will be:

- Credited in security advisories (with permission)
- Acknowledged in release notes
- Listed in a Security Hall of Fame (if desired)

Thank you for helping keep CelesteCLI and its users safe!
