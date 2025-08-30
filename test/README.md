# Quick Test Instructions

## 🚀 Quick Start (5 minutes)

### 1. Start Mock API (Terminal 1)
```bash
cd /projects/kilometers.ai/kilometers-cli/test/mock-api
go run main.go
```

### 2. Build CLI (Terminal 2)
```bash
cd /projects/kilometers.ai/kilometers-cli

# Build free version
go build -o km-free cmd/main.go

# Build premium version (requires access to private plugins repo)
go build -tags premium -o km cmd/main.go
```

### 3. Test Free User
```bash
# No API key = minimal features
./km-free monitor --server -- echo "hello"
```

### 4. Test Pro User
```bash
# Set Pro API key
./km auth login --api-key km_pro_789012

# Run with premium build - API logging activated!
./km monitor --server -- echo "hello pro"
```

### 5. Test Downgrade
```bash
# Special test key
./km auth login --api-key km_downgrade_test

# First run: Pro features work
./km monitor --server -- echo "run 1"

# Second run: Downgraded to free!
./km monitor --server -- echo "run 2"
# See: "⚠️ Some features are no longer available"
```

## 📋 Test API Keys

- `km_free_123456` - Free tier
- `km_pro_789012` - Pro tier  
- `km_ent_345678` - Enterprise tier
- `km_downgrade_test` - Simulates downgrade

## 🔍 What to Look For

1. **Free Build**: Can't use premium features even with Pro key
2. **Premium Build**: Features activate based on API response
3. **API Server Logs**: Shows feature requests and event batches
4. **Graceful Degradation**: No crashes when features disabled

## 🛠️ Detailed Testing

Run the comprehensive test guide:
```bash
bash /projects/kilometers.ai/kilometers-cli/test/test-locally.sh
```
