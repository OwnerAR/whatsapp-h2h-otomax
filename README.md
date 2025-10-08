# WhatsApp H2H Otomax

Aplikasi middleware untuk forward transaksi dari sistem Otomax ke WhatsApp menggunakan protokol Host-to-Host (H2H) dengan library [whatsmeow](https://pkg.go.dev/go.mau.fi/whatsmeow).

## 🎯 Features

- ✅ Forward transaksi dari Otomax ke WhatsApp (personal & group chat)
- ✅ Receive dan forward reply dari WhatsApp ke Otomax webhook
- ✅ Message tracking dengan in-memory cache (TTL 24 jam)
- ✅ API Key authentication
- ✅ Retry mechanism dengan exponential backoff
- ✅ Rate limiting untuk prevent spam
- ✅ Health check endpoint
- ✅ Structured logging dengan slog
- ✅ Graceful shutdown

## 📋 Flow Aplikasi

### Outgoing Flow (Otomax → WhatsApp)
```
Otomax → HTTP GET /api/v1/forward → whatsapp-h2h → WhatsApp (personal/group)
```

### Incoming Flow (WhatsApp → Otomax)
```
WhatsApp Reply → whatsapp-h2h → HTTP POST → Otomax Webhook
```

## 🏗️ Project Structure

```
whatsapp-h2h-otomax/
├── cmd/
│   └── server/
│       └── main.go              # Entry point aplikasi
├── internal/
│   ├── config/
│   │   └── config.go            # Configuration management
│   ├── handler/
│   │   ├── transaction.go       # HTTP request handlers (outgoing)
│   │   ├── webhook.go           # Webhook handlers (incoming)
│   │   └── health.go            # Health check handler
│   ├── service/
│   │   ├── whatsapp.go          # WhatsApp service logic
│   │   ├── transaction.go       # Transaction processing
│   │   └── otomax.go            # Otomax webhook client
│   ├── model/
│   │   ├── transaction.go       # Transaction models
│   │   └── message.go           # Message models
│   └── middleware/
│       └── auth.go              # Authentication middleware
├── pkg/
│   └── logger/
│       └── logger.go            # Custom logger
├── db/
│   └── whatsmeow.db             # WhatsApp session storage
├── go.mod
├── go.sum
├── .env.example
├── .gitignore
└── README.md
```

## 🚀 Getting Started

### Prerequisites

- Go 1.21 or higher
- SQLite3
- WhatsApp account untuk di-pair dengan aplikasi

### Installation

1. Clone repository:
```bash
git clone <repository-url>
cd whatsapp-h2h-otomax
```

2. Install dependencies:
```bash
go mod download
```

3. Copy `.env.example` ke `.env` dan sesuaikan konfigurasi:
```bash
cp .env.example .env
```

4. Edit `.env` file:
```env
# Server Configuration
PORT=8080
HOST=0.0.0.0

# WhatsApp Configuration
WA_DB_PATH=./db/whatsmeow.db
WA_LOG_LEVEL=INFO

# Otomax Webhook Configuration
OTOMAX_WEBHOOK_URL=https://your-otomax-webhook-url.com/api/webhook/whatsapp
OTOMAX_WEBHOOK_TIMEOUT=10s
OTOMAX_WEBHOOK_RETRY_COUNT=3

# Security
API_KEY=your-secret-api-key-here

# Rate Limiting
MAX_MESSAGES_PER_SECOND=5

# Message Tracking
MESSAGE_TRACKING_TTL=24h
```

### First Run - Pairing WhatsApp

Untuk pairing pertama kali, Anda perlu login dengan scan QR code. Jalankan aplikasi dan scan QR code yang muncul:

```bash
go run cmd/server/main.go
```

Setelah berhasil login, session akan disimpan di database (`db/whatsmeow.db`) dan Anda tidak perlu scan QR code lagi.

### Running the Application

```bash
# Run directly
go run cmd/server/main.go

# Or build first
go build -o bin/whatsapp-h2h cmd/server/main.go
./bin/whatsapp-h2h
```

## 📡 API Documentation

### 1. Forward Transaction (Outgoing)

Forward transaksi dari Otomax ke WhatsApp.

**Endpoint**: `GET /api/v1/forward`

**Headers**:
```
X-API-Key: your-secret-api-key
```

**Query Parameters**:
- `destination` (required): Nomor WhatsApp/group tujuan
  - **Personal Chat**: `628123456789` atau `628123456789@s.whatsapp.net`
  - **Group Chat**: `628123456789-1234567890@g.us` (full JID format)
- `trxid` (required): Transaction ID dari Otomax
- `descriptions` (required): Deskripsi transaksi (max 4096 chars)
- `instructions` (required): Instruksi atau detail transaksi (max 4096 chars)

**Example Request**:
```bash
curl -X GET "http://localhost:8080/api/v1/forward?destination=628123456789&trxid=TRX123456&descriptions=Pesanan%20baru&instructions=Mohon%20diproses" \
  -H "X-API-Key: your-secret-api-key"
```

**Success Response** (200):
```json
{
  "status": "success",
  "message": "Transaction forwarded successfully",
  "data": {
    "trxid": "TRX123456",
    "destination": "628123456789@s.whatsapp.net",
    "destination_type": "personal",
    "message_id": "3EB0XXXX",
    "timestamp": "2025-10-08T10:30:00Z"
  }
}
```

**Error Response** (4xx/5xx):
```json
{
  "status": "error",
  "message": "Error description",
  "error": {
    "error_code": "ERR_CODE",
    "message": "Error description"
  }
}
```

### 2. Health Check

Check service health dan connection status.

**Endpoint**: `GET /health`

**Example Request**:
```bash
curl http://localhost:8080/health
```

**Response** (200):
```json
{
  "status": "healthy",
  "whatsapp": {
    "connected": true,
    "phone": "628123456789",
    "device": "whatsapp-h2h-otomax"
  },
  "otomax_webhook": {
    "configured": true,
    "url": "https://otomax.example.com/api/webhook/whatsapp"
  },
  "uptime": "2h30m15s",
  "timestamp": "2025-10-08T10:30:00Z"
}
```

### 3. Webhook Message (Incoming)

Endpoint ini di-handle secara otomatis oleh WhatsApp event listener. Tidak perlu dipanggil manual.

## 🔐 Error Codes

| Code | Description |
|------|-------------|
| `ERR_INVALID_DESTINATION` | Invalid WhatsApp number/group JID format |
| `ERR_MISSING_PARAMETER` | Required parameter missing |
| `ERR_WHATSAPP_NOT_CONNECTED` | WhatsApp client not connected |
| `ERR_MESSAGE_SEND_FAILED` | Failed to send message |
| `ERR_RATE_LIMIT_EXCEEDED` | Rate limit exceeded |
| `ERR_UNAUTHORIZED` | Invalid or missing API key |
| `ERR_INTERNAL_SERVER` | Internal server error |
| `ERR_GROUP_NOT_FOUND` | Group not found or bot not a member |
| `ERR_DESTINATION_NOT_ON_WHATSAPP` | Phone number not registered on WhatsApp |
| `ERR_WEBHOOK_DELIVERY_FAILED` | Failed to deliver webhook to Otomax |
| `ERR_INVALID_MESSAGE_TYPE` | Unsupported message type |

## 🧪 Testing

### Unit Tests
```bash
go test ./...
```

### Test Coverage
```bash
go test -cover ./...
```

### Integration Test
```bash
# Test forward transaction
curl -X GET "http://localhost:8080/api/v1/forward?destination=YOUR_PHONE&trxid=TEST123&descriptions=Test%20Description&instructions=Test%20Instructions" \
  -H "X-API-Key: your-secret-api-key"
```

## 📝 Message Format

Format pesan yang dikirim ke WhatsApp:

```
🔔 TRANSAKSI BARU
━━━━━━━━━━━━━━━━
📋 TRX ID: TRX123456
📝 Deskripsi:
Pesanan baru dari customer

📌 Instruksi:
Mohon segera diproses dan konfirmasi

⏰ Waktu: 2025-10-08 10:30:00
```

## 🔧 Configuration

Semua konfigurasi berada di file `.env`. Berikut penjelasan masing-masing variable:

### Server
- `PORT`: Port HTTP server (default: 8080)
- `HOST`: Host HTTP server (default: 0.0.0.0)

### WhatsApp
- `WA_DB_PATH`: Path ke database session WhatsApp (default: ./db/whatsmeow.db)
- `WA_LOG_LEVEL`: Log level (DEBUG, INFO, WARN, ERROR)

### Otomax
- `OTOMAX_WEBHOOK_URL`: URL webhook Otomax untuk receive reply
- `OTOMAX_WEBHOOK_TIMEOUT`: Timeout untuk webhook request (default: 10s)
- `OTOMAX_WEBHOOK_RETRY_COUNT`: Jumlah retry jika webhook gagal (default: 3)

### Security
- `API_KEY`: API key untuk authentication

### Rate Limiting
- `MAX_MESSAGES_PER_SECOND`: Maximum messages per second (default: 5)

### Message Tracking
- `MESSAGE_TRACKING_TTL`: Time to live untuk message tracking (default: 24h)

## 🐛 Troubleshooting

### WhatsApp tidak connect
1. Pastikan sudah scan QR code
2. Check file `db/whatsmeow.db` ada dan tidak corrupt
3. Restart aplikasi

### Message tidak terkirim
1. Check WhatsApp connection status via `/health` endpoint
2. Verify destination format (personal atau group)
3. Untuk group, pastikan bot sudah join group tersebut
4. Check logs untuk detail error

### Webhook ke Otomax gagal
1. Verify `OTOMAX_WEBHOOK_URL` sudah benar
2. Check network connectivity ke Otomax server
3. Verify Otomax webhook endpoint bisa receive POST request
4. Check logs untuk detail error dan retry attempts

## 📚 References

- [Go Language Specification](https://go.dev/ref/spec)
- [Whatsmeow Documentation](https://pkg.go.dev/go.mau.fi/whatsmeow)
- [Effective Go](https://go.dev/doc/effective_go)

## 📄 License

[Specify your license here]

## 👥 Contributors

[List contributors here]

## 🤝 Support

For support, email [your-email] or create an issue in this repository.
