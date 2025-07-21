# Active Context: Current Focus and Priorities

## Current Work Focus

### 🚨 **CRITICAL: TRANSPARENT PROXY BROKEN - BLOCKING MCP INTEGRATION**

**Status**: **CRITICAL BUG DISCOVERED** - km monitor not working as MCP server  
**Priority**: **URGENT** - Blocks real-world usage in Cursor MCP configuration  
**Objective**: Fix transparent proxy mode so km works seamlessly as MCP server

#### **NEW CRITICAL ISSUE: KIL-65 - Transparent Proxy Mode Broken** 🚨

**Problem**: When `km monitor` is used as MCP server in Cursor's mcp.json:

```json
"sequential-thinking": {
  "command": "km",
  "args": ["monitor", "--", "npx", "-y", "@modelcontextprotocol/server-sequential-thinking"]
}
```

**Result**: Cursor shows **"0 tools enabled"** because MCP protocol forwarding is broken.

**Root Cause Analysis**:

1. **Message parsing too strict** - `parseEventFromData` fails on valid MCP messages, disrupting forwarding
2. **Error handling breaks proxy** - JSON parsing failures interrupt transparent data flow
3. **Forwarding not truly transparent** - Monitoring logic interferes with MCP protocol flow
4. **Buffer/framing issues** - Message boundaries not handled correctly for MCP handshake

**Impact**:

- ❌ Cannot use km as MCP server in Cursor configuration
- ❌ tools/list and other MCP protocol messages not forwarded properly
- ❌ Breaks the entire value proposition of transparent monitoring
- ❌ Users cannot get immediate monitoring visibility in their existing workflows

#### **Technical Details**

**Expected Flow**:

```
Cursor → km monitor → MCP Server
  ↓ tools/list      ↓ tools/list
  ↑ tools response  ↑ tools response
```

**Current Broken Flow**:

```
Cursor → km monitor → MCP Server
  ↓ tools/list      ✗ parsing error/blocked
  ↑ no response     ✗ forwarding fails
```

**Critical Code Locations**:

- `internal/application/services/monitoring_service.go:477` - stdout forwarding
- `internal/application/services/monitoring_service.go:520` - parseEventFromData
- `internal/interfaces/cli/monitor.go:147` - stdin forwarding
- `internal/infrastructure/monitoring/process_monitor.go` - stream handling

### 🎉 **CORE MESSAGE PROCESSING - PREVIOUSLY COMPLETED ✅**

**Status**: Basic message processing works in isolation  
**Note**: While JSON-RPC parsing works, transparent proxy integration is broken

#### Previously Fixed Issues ✅

**KIL-64: MCP Message Framing** ✅ **COMPLETED**

- ✅ Newline-delimited JSON parsing works in isolation
- ⚠️ BUT: Interferes with transparent proxy mode

**KIL-62: Buffer Size Limitations** ✅ **COMPLETED**

- ✅ 1MB buffer size prevents "token too long" errors
- ⚠️ BUT: May not handle MCP handshake edge cases

**KIL-61: JSON-RPC Parsing** ✅ **COMPLETED**

- ✅ JSON-RPC 2.0 structure parsing works
- ⚠️ BUT: Too strict - breaks on valid MCP protocol variations

### 🚀 **TERMINAL DASHBOARD FEATURE - MVP COMPLETED ✅**

**Status**: MVP working with mock data - ready for real data integration after proxy fix  
**Priority**: High - but blocked by transparent proxy issue

#### Dashboard Status

- **Working**: Full TUI with mock events, keyboard controls, session display
- **Blocked**: Cannot show real MCP events until transparent proxy is fixed
- **Ready**: For Phase 2 enhancements once proxy works

## Strategic Impact

### **Current State Assessment**

- **Architecture**: ✅ Excellent foundation with clean patterns
- **CLI Interface**: ✅ Professional command structure
- **Message Processing**: ⚠️ Works in isolation, broken in transparent mode
- **Dashboard**: ✅ MVP complete, ready for real data
- **Real-World Usage**: ❌ **COMPLETELY BLOCKED** by transparent proxy issue

### **Business Impact of Transparent Proxy Bug**

1. **Cannot fulfill primary value proposition** - Seamless MCP monitoring
2. **Blocks adoption** - Users cannot integrate km into existing Cursor workflows
3. **Breaks toolchain integration** - MCP servers don't work through km proxy
4. **Prevents validation** - Cannot demonstrate value with real MCP usage

## Next Development Phase

### **URGENT Phase 1: Fix Transparent Proxy (Current Sprint)**

**Goal**: Make km monitor work flawlessly as transparent MCP proxy

**Critical Tasks**:

1. **Separate forwarding from monitoring** (Day 1)

   - Ensure ALL data forwarded immediately regardless of parsing success
   - Make event capture completely non-blocking and async
   - Fix `readProcessOutput` to prioritize forwarding over analysis

2. **Fix MCP protocol flow** (Day 1-2)

   - Test tools/list specifically with sequential-thinking server
   - Validate full MCP handshake works through km proxy
   - Ensure initialization/capabilities exchange works correctly

3. **Robust error handling** (Day 2)

   - Make `parseEventFromData` failures non-critical
   - Add debug logging to trace message flow
   - Graceful degradation when monitoring fails

4. **Validation Testing** (Day 2-3)
   - Test with multiple MCP servers (sequential-thinking, github, linear)
   - Verify Cursor shows correct tool counts
   - Confirm km behaves identically to direct MCP server execution

### **Success Criteria for Transparent Proxy Fix**

1. **Cursor Integration**: ✅ km monitor works in mcp.json, shows correct tool counts
2. **Protocol Compliance**: ✅ All MCP messages forwarded without modification
3. **Monitoring Functionality**: ✅ Events captured while maintaining transparency
4. **Error Resilience**: ✅ Parsing failures don't break proxy operation
5. **Performance**: ✅ No noticeable latency vs direct MCP server connection

### **Phase 2: Dashboard Enhancement (After Proxy Fix)**

Once transparent proxy works:

1. Connect dashboard to real MCP event streams
2. Add interactive filtering and search
3. Multiple view modes and session selection

## Development Environment

### **For Transparent Proxy Debugging**

```bash
# Build and install
go build -o km cmd/main.go

# Test transparent proxy with Cursor
# Add to ~/.cursor/mcp.json:
{
  "mcpServers": {
    "test-km": {
      "command": "km",
      "args": ["monitor", "--", "npx", "-y", "@modelcontextprotocol/server-sequential-thinking"]
    }
  }
}

# Test direct vs proxied
npx -y @modelcontextprotocol/server-sequential-thinking  # Direct
km monitor -- npx -y @modelcontextprotocol/server-sequential-thinking  # Proxied

# Debug with verbose logging
km monitor --debug -- npx -y @modelcontextprotocol/server-sequential-thinking
```

This active context reflects the critical reality: we have a fundamental transparent proxy issue that blocks real-world MCP integration, requiring immediate attention! 🚨
