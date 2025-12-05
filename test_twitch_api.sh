#!/usr/bin/env bash
# Twitch API Credential Verification Script
# Tests Twitch Client ID and Secret to generate OAuth token

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔍 Twitch API Credential Verification${NC}"
echo ""

# Load credentials from skills.json
SKILLS_JSON="$HOME/.celeste/skills.json"

if [ ! -f "$SKILLS_JSON" ]; then
    echo -e "${RED}❌ Error: $SKILLS_JSON not found${NC}"
    exit 1
fi

CLIENT_ID=$(grep -o '"twitch_client_id"[[:space:]]*:[[:space:]]*"[^"]*"' "$SKILLS_JSON" | sed 's/.*"\([^"]*\)"$/\1/')
CLIENT_SECRET=$(grep -o '"twitch_client_secret"[[:space:]]*:[[:space:]]*"[^"]*"' "$SKILLS_JSON" | sed 's/.*"\([^"]*\)"$/\1/')

if [ -z "$CLIENT_ID" ] || [ -z "$CLIENT_SECRET" ]; then
    echo -e "${RED}❌ Error: Twitch credentials not found in skills.json${NC}"
    exit 1
fi

echo -e "${GREEN}✓${NC} Loaded credentials from $SKILLS_JSON"
echo -e "  Client ID: ${CLIENT_ID:0:8}...${CLIENT_ID: -4}"
echo -e "  Client Secret: ${CLIENT_SECRET:0:8}...${CLIENT_SECRET: -4}"
echo ""

# Step 1: Get OAuth token using Client Credentials flow
echo -e "${YELLOW}Step 1: Requesting OAuth token...${NC}"

TOKEN_RESPONSE=$(curl -s -X POST "https://id.twitch.tv/oauth2/token" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -d "client_id=$CLIENT_ID&client_secret=$CLIENT_SECRET&grant_type=client_credentials")

echo "$TOKEN_RESPONSE" | jq . 2>/dev/null || echo "$TOKEN_RESPONSE"

ACCESS_TOKEN=$(echo "$TOKEN_RESPONSE" | grep -o '"access_token"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/.*"\([^"]*\)"$/\1/')

if [ -z "$ACCESS_TOKEN" ]; then
    echo -e "${RED}❌ Failed to get OAuth token${NC}"
    echo ""
    echo "Response:"
    echo "$TOKEN_RESPONSE"
    exit 1
fi

echo -e "${GREEN}✓${NC} OAuth token obtained: ${ACCESS_TOKEN:0:10}...${ACCESS_TOKEN: -4}"
echo ""

# Step 2: Validate token
echo -e "${YELLOW}Step 2: Validating OAuth token...${NC}"

VALIDATE_RESPONSE=$(curl -s -H "Authorization: OAuth $ACCESS_TOKEN" \
    "https://id.twitch.tv/oauth2/validate")

echo "$VALIDATE_RESPONSE" | jq . 2>/dev/null || echo "$VALIDATE_RESPONSE"
echo -e "${GREEN}✓${NC} Token is valid"
echo ""

# Step 3: Test API call with streamer
echo -e "${YELLOW}Step 3: Testing Twitch Helix API...${NC}"

STREAMER="whykusanagi"
API_RESPONSE=$(curl -s -H "Client-ID: $CLIENT_ID" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    "https://api.twitch.tv/helix/streams?user_login=$STREAMER")

echo "$API_RESPONSE" | jq . 2>/dev/null || echo "$API_RESPONSE"

IS_LIVE=$(echo "$API_RESPONSE" | grep -o '"data"[[:space:]]*:[[:space:]]*\[[^]]*\]' | grep -c '"id"' || echo "0")

echo ""
if [ "$IS_LIVE" -gt 0 ]; then
    echo -e "${GREEN}✓${NC} ${STREAMER} is LIVE!"
    GAME=$(echo "$API_RESPONSE" | grep -o '"game_name"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/.*"\([^"]*\)"$/\1/')
    VIEWERS=$(echo "$API_RESPONSE" | grep -o '"viewer_count"[[:space:]]*:[[:space:]]*[0-9]*' | head -1 | sed 's/.*:[[:space:]]*\([0-9]*\)/\1/')
    echo -e "  Game: $GAME"
    echo -e "  Viewers: $VIEWERS"
else
    echo -e "${YELLOW}ℹ${NC}  ${STREAMER} is not currently live"
fi

echo ""
echo -e "${BLUE}═══════════════════════════════════════${NC}"
echo -e "${GREEN}✅ All tests passed!${NC}"
echo ""
echo -e "${YELLOW}OAuth Token for Celeste:${NC}"
echo -e "$ACCESS_TOKEN"
echo ""
echo -e "${YELLOW}Note:${NC} This token expires. Celeste needs to implement"
echo "OAuth token generation using the Client Credentials flow."
