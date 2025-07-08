# Rate Limiter

This package implements a sliding window rate limiter for the TFGrid Proxy server.

## Features

- **IP-based Rate Limiting**: Tracks requests per IP address
- **Sliding Window Algorithm**: Uses a sliding window approach for smooth rate limiting
- **Thread-Safe**: Safe for concurrent use across multiple goroutines
- **Memory Efficient**: Automatic cleanup of old entries
- **Configurable**: Rate limit can be set via command-line flag

## Usage

### Command Line Flag

Use the `--rate-limit-rps` flag to set the rate limit:

```bash
# Enable rate limiting at 20 requests per second per IP (default)
./proxy_server --rate-limit-rps 20

# Set custom rate limit of 100 requests per second per IP
./proxy_server --rate-limit-rps 100

# Disable rate limiting
./proxy_server --rate-limit-rps 0
```

### IP Address Detection

The rate limiter automatically extracts the client IP address using the following priority:

1. `X-Real-IP` header
2. `X-Forwarded-For` header (first IP if multiple)
3. `RemoteAddr` from the connection

This ensures proper rate limiting even when the proxy is behind load balancers or CDNs.

### HTTP Response

When rate limit is exceeded, the server returns:

- **Status Code**: 429 (Too Many Requests)
- **Headers**:
  - `Retry-After: 1` - Suggests retrying after 1 second
  - `X-RateLimit-Limit: 20` - Current rate limit
  - `X-RateLimit-Remaining: 0` - Remaining requests (0 when exceeded)

### Algorithm Details

The sliding window algorithm works as follows:

1. **Time Window**: Uses a 1-second sliding window
2. **Request Tracking**: Stores timestamps of requests within the window
3. **Cleanup**: Automatically removes requests older than the window
4. **Memory Management**: Periodically cleans up inactive IP entries

### Performance

- **Memory Usage**: Minimal overhead, only stores active IP addresses
- **CPU Usage**: O(n) where n is the number of requests in the current window
- **Concurrency**: Thread-safe with read-write mutexes for optimal performance

## Configuration

| Flag | Default | Description |
|------|---------|-------------|
| `--rate-limit-rps` | 20 | Requests per second per IP address (0 to disable) |

## Logging

The rate limiter provides debug logging for:

- Rate limit violations (WARN level)
- New IP tracking (DEBUG level)
- Request allowances (DEBUG level)
- Cleanup operations (DEBUG level)

Example log output:
```
{"level":"warn","ip":"192.168.1.100","method":"GET","path":"/nodes","time":"2025-07-07T14:40:43Z","message":"Rate limit exceeded"}
```
