# Migration Notes - Database-based Tracking & Webhook Whitelist

## Version 1.3.0 - October 8, 2025

### What Changed

#### 1. **Database-based Transaction Tracking** 
**Previously**: Transaction tracking menggunakan in-memory cache (map) yang hilang saat restart.  
**Now**: Transaction tracking disimpan di SQLite database (`tracking.db`) untuk persistence.

**Why**: 
- Mencegah duplicate transaction bahkan setelah aplikasi restart
- Lebih reliable untuk production environment
- Data tracking tidak hilang saat crash/restart

**Technical Changes**:
- Created `internal/repository/transaction.go` - Database layer untuk CRUD operations
- Updated `internal/service/transaction.go` - Use repository instead of MessageTracker
- New database table: `transactions` dengan auto-cleanup expired records
- Removed in-memory `MessageTracker` (map-based cache)

#### 2. **Webhook Whitelist Filter**
**Previously**: Webhook dikirim untuk semua incoming messages dari chat manapun.  
**Now**: Webhook hanya dikirim jika sender JID/Group ada dalam whitelist (jika configured).

**Why**:
- Security: Hanya process messages dari contact/group yang terdaftar
- Kontrol: Admin bisa atur siapa saja yang boleh trigger webhook ke Otomax
- Flexibility: Bisa disabled dengan leave whitelist kosong (allow all)

**Technical Changes**:
- Added `WEBHOOK_WHITELIST_JIDS` config in `.env`
- Updated `internal/service/whatsapp.go` - Check whitelist before sending webhook
- Added `isWhitelisted()` function untuk validation

---

## Database Schema

### Table: `transactions`

```sql
CREATE TABLE transactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    trx_id TEXT NOT NULL UNIQUE,         -- Transaction ID dari Otomax
    message_id TEXT NOT NULL,             -- WhatsApp message ID
    destination TEXT NOT NULL,            -- JID (phone or group)
    destination_type TEXT NOT NULL,       -- "personal" or "group"
    sent_at DATETIME NOT NULL,            -- Kapan message dikirim
    expires_at DATETIME NOT NULL,         -- TTL (default 24 jam)
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Indexes untuk performance
CREATE INDEX idx_trx_id ON transactions(trx_id);
CREATE INDEX idx_expires_at ON transactions(expires_at);
CREATE INDEX idx_destination ON transactions(destination);
```

**Location**: `./db/tracking.db` (configurable via `TRACKING_DB_PATH`)

---

## Configuration Changes

### New Environment Variables

Add to your `.env` file:

```env
# Transaction Tracking Database
TRACKING_DB_PATH=./db/tracking.db

# Webhook Whitelist (comma-separated JID/Group IDs)
# Leave empty to allow all (default behavior)
# Example: 628123456789@s.whatsapp.net,120363365891642441@g.us
WEBHOOK_WHITELIST_JIDS=
```

### Migration Steps

1. **Update `.env` file**:
   ```bash
   cp .env .env.backup
   # Add new configs (already in .env.example)
   echo "" >> .env
   echo "TRACKING_DB_PATH=./db/tracking.db" >> .env
   echo "WEBHOOK_WHITELIST_JIDS=" >> .env
   ```

2. **Ensure database directory exists**:
   ```bash
   mkdir -p db
   ```

3. **Rebuild application**:
   ```bash
   go build -ldflags="-s -w" -o bin/whatsapp-h2h cmd/server/main.go
   ```

4. **Test duplicate prevention** (see Testing section below)

---

## Breaking Changes

### Code Changes

#### Removed:
- `MessageTracker` struct (in-memory cache)
- `transactionService.GetMessageTracker()` method
- `whatsappService.SetMessageTracker()` method

#### Added:
- `TransactionRepository` struct (database layer)
- `transactionService.GetRepository()` method
- `whatsappService.SetTransactionRepository()` method
- `whatsappService.SetWebhookWhitelist()` method

### API Behavior Changes

**No API changes** - All endpoints remain the same:
- `GET /api/v1/forward` - Same parameters, improved duplicate detection
- `POST /api/v1/webhook/message` - No changes
- `GET /api/v1/groups` - No changes
- `GET /health` - No changes

---

## Testing Guide

### 1. Test Duplicate Prevention (Database)

```bash
# Test 1: Send transaction pertama
curl -X GET "http://localhost:8080/api/v1/forward?destination=628123456789&trxid=DUP_TEST_001&descriptions=Test%20duplicate&instructions=Test" \
  -H "X-API-Key: your-secret-api-key"

# Expected: 200 OK, message sent
# Check logs: "Transaction sent"

# Test 2: Send duplicate (same TrxID, immediately)
curl -X GET "http://localhost:8080/api/v1/forward?destination=628123456789&trxid=DUP_TEST_001&descriptions=Test%20duplicate&instructions=Test" \
  -H "X-API-Key: your-secret-api-key"

# Expected: 409 Conflict, ERR_DUPLICATE_TRANSACTION
# Check logs: "Duplicate transaction detected"
```

**Key Difference from Before**:  
- ✅ Duplicate check tetap berlaku **setelah restart**
- ✅ Data tersimpan di database `./db/tracking.db`

### 2. Test Restart Persistence

```bash
# Step 1: Send transaction
curl -X GET "http://localhost:8080/api/v1/forward?destination=628123456789&trxid=RESTART_TEST&descriptions=Test&instructions=Test" \
  -H "X-API-Key: your-secret-api-key"
# Expected: 200 OK

# Step 2: Restart aplikasi
pkill whatsapp-h2h
./bin/whatsapp-h2h

# Step 3: Try send same TrxID lagi
curl -X GET "http://localhost:8080/api/v1/forward?destination=628123456789&trxid=RESTART_TEST&descriptions=Test&instructions=Test" \
  -H "X-API-Key: your-secret-api-key"
# Expected: 409 Conflict (duplicate detected from database!)
```

### 3. Test Webhook Whitelist

**Scenario A: No Whitelist (Allow All)**

```bash
# .env setting:
WEBHOOK_WHITELIST_JIDS=

# Send message to any contact
# Expected: Webhook dikirim ke Otomax (default behavior)
```

**Scenario B: Whitelist Enabled**

```bash
# .env setting:
WEBHOOK_WHITELIST_JIDS=628123456789@s.whatsapp.net,120363365891642441@g.us

# Test 1: Send to whitelisted number
curl -X GET "http://localhost:8080/api/v1/forward?destination=628123456789&trxid=WHITELIST_OK&descriptions=Test&instructions=Test" \
  -H "X-API-Key: your-secret-api-key"
# Reply from WhatsApp: Webhook WILL be sent

# Test 2: Send to non-whitelisted number
curl -X GET "http://localhost:8080/api/v1/forward?destination=628999999999&trxid=WHITELIST_BLOCK&descriptions=Test&instructions=Test" \
  -H "X-API-Key: your-secret-api-key"
# Reply from WhatsApp: Webhook will NOT be sent
# Check logs: "Message from non-whitelisted JID ignored"
```

### 4. Test Database Auto-Cleanup

```bash
# Check expired cleanup (runs every 1 hour)
# Wait 1 hour or manually trigger by changing MESSAGE_TRACKING_TTL to "1m" and restart

# After TTL expired:
curl -X GET "http://localhost:8080/api/v1/forward?destination=628123456789&trxid=OLD_TRX_ID&descriptions=Test&instructions=Test" \
  -H "X-API-Key: your-secret-api-key"
# Expected: 200 OK (old tracking expired, can reuse TrxID)

# Check logs for cleanup:
# "Cleaned up expired transactions" count=X
```

---

## Database Operations

### Manual Database Inspection

```bash
# Install sqlite3 if needed
sudo apt install sqlite3

# Open database
sqlite3 ./db/tracking.db

# Check all active transactions
SELECT * FROM transactions WHERE expires_at > datetime('now');

# Check expired transactions
SELECT * FROM transactions WHERE expires_at <= datetime('now');

# Count total records
SELECT COUNT(*) FROM transactions;

# Check specific TrxID
SELECT * FROM transactions WHERE trx_id = 'TEST123';

# Exit
.exit
```

### Manual Cleanup

```bash
# Clean expired entries manually
sqlite3 ./db/tracking.db "DELETE FROM transactions WHERE expires_at <= datetime('now');"

# Drop all tracking (CAUTION: This will allow duplicate TrxIDs!)
sqlite3 ./db/tracking.db "DELETE FROM transactions;"

# Reset database (delete and recreate)
rm ./db/tracking.db
# App will auto-create on next start
```

---

## Performance Considerations

### Database Size
- **Estimated size**: ~100 bytes per transaction
- **With 1000 transactions**: ~100 KB
- **With 100,000 transactions**: ~10 MB
- **Auto cleanup**: Runs every 1 hour, removes expired entries

### Database Indexes
- Optimized queries with 3 indexes:
  - `idx_trx_id` - Fast duplicate check by TrxID
  - `idx_expires_at` - Fast cleanup of expired records
  - `idx_destination` - Fast lookup for incoming webhooks

### Recommendations
- **Production**: Keep `MESSAGE_TRACKING_TTL=24h` (default)
- **High volume**: Consider reducing TTL to `12h` or `6h`
- **Monitor**: Check database size periodically
- **Backup**: Include `db/tracking.db` in backup strategy

---

## Troubleshooting

### Issue: "Failed to initialize transaction repository"

**Solution**:
```bash
# Ensure db directory exists
mkdir -p db

# Check permissions
chmod 755 db

# Check if database is locked
lsof | grep tracking.db
# Kill process if needed
```

### Issue: "Duplicate transaction" but it shouldn't be

**Solution**:
```bash
# Check if transaction still in database
sqlite3 ./db/tracking.db "SELECT * FROM transactions WHERE trx_id = 'YOUR_TRX_ID';"

# If expired but still exists, manually cleanup
sqlite3 ./db/tracking.db "DELETE FROM transactions WHERE expires_at <= datetime('now');"

# Or restart app to force cleanup
pkill whatsapp-h2h && ./bin/whatsapp-h2h
```

### Issue: "Message from non-whitelisted JID ignored"

**Solution**:
```bash
# Check whitelist config
cat .env | grep WEBHOOK_WHITELIST_JIDS

# Get JID dari groups endpoint
curl -H "X-API-Key: your-api-key" http://localhost:8080/api/v1/groups

# Update whitelist
nano .env
# Add JID to WEBHOOK_WHITELIST_JIDS (comma-separated)
# Restart app
```

### Issue: Database locked

**Solution**:
```bash
# Check if multiple instances running
ps aux | grep whatsapp-h2h

# Kill all instances
pkill whatsapp-h2h

# Start fresh
./bin/whatsapp-h2h
```

---

## Rollback Plan

If you need to rollback to in-memory tracking:

```bash
# 1. Checkout previous version
git checkout v1.2.0

# 2. Rebuild
go build -o bin/whatsapp-h2h cmd/server/main.go

# 3. Remove new config from .env (optional)
# Comment out TRACKING_DB_PATH and WEBHOOK_WHITELIST_JIDS

# 4. Restart
pkill whatsapp-h2h && ./bin/whatsapp-h2h
```

---

## Summary

✅ **More Reliable**: Database persistence untuk duplicate prevention  
✅ **More Secure**: Webhook whitelist untuk kontrol akses  
✅ **More Scalable**: Auto-cleanup expired records  
✅ **Backward Compatible**: No API changes, seamless upgrade  
✅ **Production Ready**: Tested dengan build success  

**Next Steps**:
1. Update `.env` dengan new configs
2. Rebuild aplikasi
3. Test duplicate prevention
4. Configure whitelist (if needed)
5. Monitor logs dan database size

