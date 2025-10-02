# Kilometers.ai CLI Proxy API Endpoint Contracts

## Base URL
- **Default**: `https://api.kilometers.ai`
- **Configurable via**:
  - Environment variable: `KM_API_URL`
  - Config file: `km_config.json`
  - CLI argument: `--api-url`

## API Endpoints

### 1. Authentication Exchange

**Endpoint**: `/auth/exchange`
**HTTP Method**: `POST`
**Full URL**: `{base_url}/auth/exchange`

**Purpose**: Exchange API key for a JWT token with user claims

**Request Body**:
```json
{
  "api_key": "string"
}
```

**Response Body**:
```json
{
  "jwt": "string",
  "expires_in": 3600
}
```

**JWT Token Claims** (decoded from JWT):
```json
{
  "sub": "string (optional)",
  "tier": "string (optional) - user tier level",
  "exp": "number (optional) - expiration timestamp",
  "iat": "number (optional) - issued at timestamp",
  "user_id": "string (optional)"
}
```

**Error Handling**:
- Returns non-2xx status code on authentication failure

---

### 2. Telemetry Event Submission

**Endpoint**: `/api/events/telemetry`
**HTTP Method**: `POST`
**Full URL**: `{base_url}/api/events/telemetry`

**Purpose**: Send telemetry events for command execution tracking

**Headers**:
```
Authorization: Bearer {jwt_token}
```

**Request Body**:
```json
{
  "event_type": "command_execution",
  "timestamp": "2024-01-01T00:00:00Z",
  "user_id": "string (optional)",
  "user_tier": "string (default: 'free')",
  "command": "string",
  "args": ["array", "of", "strings"],
  "session_id": "uuid-v4",
  "metadata": {
    "key": "value pairs as strings"
  }
}
```

**Response Body (2xx)**:
```json
{
  "status": "string",
  "message": "string (optional)",
  "events_remaining": 1000
}
```

**Status Codes**:
- `200-299`: Success - event recorded
- `429`: Rate limit exceeded - request continues without telemetry
- Other: Telemetry failed - logged as warning, execution continues

---

### 3. Risk Analysis

**Endpoint**: `/api/risk/analyze`
**HTTP Method**: `POST`
**Full URL**: `{base_url}/api/risk/analyze`

**Purpose**: Analyze command risk for paid tier users

**Headers**:
```
Authorization: Bearer {jwt_token}
```

**Request Body**:
```json
{
  "command": "string",
  "args": ["array", "of", "strings"],
  "metadata": {}
}
```

**Response Body**:
```json
{
  "risk_score": 0.75,
  "risk_level": "high|medium|low",
  "recommendation": "string - human readable recommendation",
  "details": {},
  "suggested_transform": {
    "command": "string (optional) - transformed command",
    "args": ["optional", "transformed", "args"],
    "reason": "string - explanation for transformation"
  }
}
```

**Business Logic**:
- Only available for non-free tier users
- If `risk_score > threshold` (default 0.8), command is blocked
- May suggest command transformations for safer execution

**Error Handling**:
- Non-2xx status codes result in error

---

## Authentication Flow

1. **Initial Authentication**:
   - Client sends API key to `/auth/exchange`
   - Server validates and returns JWT token
   - JWT contains user claims including tier level
   - Token is stored securely in OS keyring (Keychain/Credential Manager/Secret Service)

2. **Token Usage**:
   - JWT token used as Bearer token for all subsequent API calls
   - Token retrieved from OS keyring when needed
   - Token expires after `expires_in` seconds
   - Token checked for expiration with 60-second buffer

3. **Token Renewal**:
   - When token expires, new exchange request made automatically
   - New token stored securely in OS keyring, replacing expired token

---

## Filter Pipeline Order

1. **LocalLoggerFilter**: Always runs - logs to local file
2. **EventSenderFilter**: Sends telemetry (non-blocking)
3. **RiskAnalysisFilter**: Only for paid tiers (can block/transform)

---

## Environment Configuration

The application supports multiple configuration methods in order of precedence:

1. **Environment Variables** (highest priority):
   - `KM_API_KEY`: API key for authentication
   - `KM_API_URL`: Base URL for API
   - `KM_DEFAULT_TIER`: Default user tier

2. **Configuration File** (`km_config.json`) - for settings only:
```json
{
  "api_url": "https://api.kilometers.ai",
  "default_tier": "enterprise"
}
```

3. **OS Keyring** - for secure credential storage:
   - JWT access tokens stored with key `km-access-token`
   - JWT refresh tokens stored with key `km-refresh-token`
   - Service name: `ai.kilometers.km`

4. **Default Values** (lowest priority):
   - API URL defaults to `https://api.kilometers.ai`
   - Tier defaults to `free` if not specified

## Implementation Details

### Source Code Locations

- **Authentication**: `src/auth.rs` - `AuthClient::exchange_for_jwt()`
- **Keyring Storage**: `src/keyring_token_store.rs` - Secure token storage in OS keyring
- **Telemetry**: `src/filters/event_sender.rs` - `EventSenderFilter::send_telemetry_event()`
- **Risk Analysis**: `src/filters/risk_analysis.rs` - `RiskAnalysisFilter::analyze_risk()`
- **Configuration**: `src/config.rs` - Config loading and environment variable handling
- **Filter Pipeline**: `src/main.rs` - Filter setup and execution order

### HTTP Client

All API calls use the `reqwest::Client` with:
- JSON content type for request bodies
- Bearer token authentication (except initial auth exchange)
- Proper error handling and context propagation
