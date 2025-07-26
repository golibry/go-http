# Session Management

This package provides comprehensive session management functionality for Go HTTP applications, including session attributes, flash messages, encryption, idle timeout, and garbage collection.

## Features

- **Session Attributes**: Store and retrieve key-value pairs with optional encryption
- **Flash Messages**: Temporary messages that persist across requests and are automatically removed after retrieval
- **Session Lifecycle**: Automatic session creation, idle timeout, and expiration handling
- **Encryption**: Optional AES-GCM encryption for sensitive session data
- **Garbage Collection**: Automatic cleanup of expired sessions
- **Pluggable Storage**: Interface-based storage system with in-memory implementation included
- **Middleware Integration**: Easy integration with HTTP middleware chains

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log/slog"
    "net/http"
    "os"
    
    "github.com/golibry/go-http/http/session"
    "github.com/golibry/go-http/http/router/middleware"
)

func main() {
    // Create storage and manager
    storage := session.NewMemoryStorage()
    ctx := context.Background()
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
    
    options := session.DefaultOptions()
    manager := session.NewManager(storage, ctx, logger, options)
    
    // Start garbage collection
    manager.StartGC(ctx)
    defer manager.StopGC()
    
    // Create middleware
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Get or create session
        sess, err := middleware.GetOrCreateSession(r.Context(), w, r, manager)
        if err != nil {
            http.Error(w, "Session error", http.StatusInternalServerError)
            return
        }
        
        // Use session
        sess.Set("user_id", "12345")
        sess.AddFlash("Welcome back!")
        
        w.WriteHeader(http.StatusOK)
    })
    
    sessionMiddleware := middleware.NewSessionMiddleware(handler, ctx, logger, manager)
    
    http.Handle("/", sessionMiddleware)
    http.ListenAndServe(":8080", nil)
}
```

### With Encryption

```go
package main

import (
    "context"
    "crypto/rand"
    "log/slog"
    "net/http"
    "os"
    
    "github.com/golibry/go-http/http/session"
)

func main() {
    // Generate encryption key (32 bytes for AES-256)
    encryptionKey := make([]byte, 32)
    if _, err := rand.Read(encryptionKey); err != nil {
        panic(err)
    }
    
    // Configure options with encryption
    options := session.DefaultOptions()
    options.EncryptionKey = encryptionKey
    options.CookieSecure = true // Use HTTPS in production
    
    storage := session.NewMemoryStorage()
    ctx := context.Background()
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
    
    manager := session.NewManager(storage, ctx, logger, options)
    manager.StartGC(ctx)
    defer manager.StopGC()
    
    // Your handlers here...
}
```

## Configuration Options

```go
type Options struct {
    // Cookie settings
    CookieName     string        // Default: "session_id"
    CookiePath     string        // Default: "/"
    CookieDomain   string        // Default: ""
    CookieSecure   bool          // Default: false
    CookieHTTPOnly bool          // Default: true
    CookieSameSite http.SameSite // Default: http.SameSiteLaxMode
    
    // Session settings
    MaxAge         time.Duration // Default: 24 hours
    IdleTimeout    time.Duration // Default: 30 minutes
    EncryptionKey  []byte        // Default: nil (no encryption)
    
    // Garbage collection
    GCInterval     time.Duration // Default: 5 minutes
    
    // Security
    SecureRandom   bool          // Default: true
}
```

## Session Interface

```go
type Session interface {
    // Basic operations
    ID() string
    
    // Attributes
    Get(key string) (interface{}, bool)
    Set(key string, value interface{})
    Delete(key string)
    Clear()
    
    // Flash messages
    AddFlash(message interface{}, category ...string)
    GetFlashes(category ...string) []interface{}
    
    // Lifecycle
    Touch()
    LastAccess() time.Time
    CreatedAt() time.Time
    IsExpired(maxAge time.Duration) bool
    Save(ctx context.Context) error
    Destroy(ctx context.Context) error
}
```

## Working with Sessions

### Session Attributes

```go
// Set various types of data
session.Set("user_id", 12345)
session.Set("username", "john_doe")
session.Set("preferences", map[string]interface{}{
    "theme": "dark",
    "language": "en",
})

// Retrieve data
if userID, exists := session.Get("user_id"); exists {
    fmt.Printf("User ID: %v\n", userID)
}

// Delete specific attribute
session.Delete("temporary_data")

// Clear all attributes
session.Clear()
```

### Flash Messages

```go
// Add flash messages
session.AddFlash("Operation completed successfully!")
session.AddFlash("Invalid input", "error")
session.AddFlash("Please verify your email", "warning")

// Retrieve flash messages (they are automatically removed)
messages := session.GetFlashes() // Default category
errorMessages := session.GetFlashes("error")
warningMessages := session.GetFlashes("warning")

// Display in template
for _, msg := range messages {
    fmt.Printf("Message: %v\n", msg)
}
```

### Session Lifecycle

```go
// Check if session is expired
if session.IsExpired(24 * time.Hour) {
    // Handle expired session
}

// Update last access time
session.Touch()

// Get session timestamps
createdAt := session.CreatedAt()
lastAccess := session.LastAccess()

// Manually save session (usually done automatically by middleware)
if err := session.Save(ctx); err != nil {
    // Handle save error
}

// Destroy session
if err := session.Destroy(ctx); err != nil {
    // Handle destroy error
}
```

## Custom Storage Implementation

You can implement your own storage backend by implementing the `Storage` interface:

```go
type Storage interface {
    Get(ctx context.Context, sessionID string) ([]byte, error)
    Set(ctx context.Context, sessionID string, data []byte, expiration time.Duration) error
    Delete(ctx context.Context, sessionID string) error
    Cleanup(ctx context.Context) error
    Exists(ctx context.Context, sessionID string) bool
}
```

### Redis Storage Example

```go
type RedisStorage struct {
    client *redis.Client
}

func NewRedisStorage(client *redis.Client) *RedisStorage {
    return &RedisStorage{client: client}
}

func (rs *RedisStorage) Get(ctx context.Context, sessionID string) ([]byte, error) {
    result, err := rs.client.Get(ctx, "session:"+sessionID).Result()
    if err == redis.Nil {
        return nil, session.ErrSessionNotFound
    }
    if err != nil {
        return nil, err
    }
    return []byte(result), nil
}

func (rs *RedisStorage) Set(ctx context.Context, sessionID string, data []byte, expiration time.Duration) error {
    return rs.client.Set(ctx, "session:"+sessionID, data, expiration).Err()
}

func (rs *RedisStorage) Delete(ctx context.Context, sessionID string) error {
    return rs.client.Del(ctx, "session:"+sessionID).Err()
}

func (rs *RedisStorage) Cleanup(ctx context.Context) error {
    // Redis handles expiration automatically
    return nil
}

func (rs *RedisStorage) Exists(ctx context.Context, sessionID string) bool {
    result, _ := rs.client.Exists(ctx, "session:"+sessionID).Result()
    return result > 0
}
```

## Middleware Integration

The session middleware automatically handles session lifecycle:

```go
import (
    "github.com/golibry/go-http/http/router/middleware"
    "github.com/golibry/go-http/http/session"
)

// Create your handler
handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // Session is automatically available in context
    if sess, ok := middleware.GetSessionFromContext(r.Context()); ok {
        // Use session
        sess.Set("page_views", getPageViews(sess) + 1)
    }
    
    // Response handling...
})

// Wrap with session middleware
sessionMiddleware := middleware.NewSessionMiddleware(handler, ctx, logger, manager)

// Use in your HTTP server
http.Handle("/", sessionMiddleware)
```

## Security Considerations

1. **Use HTTPS in Production**: Set `CookieSecure: true` when using HTTPS
2. **Enable Encryption**: Use `EncryptionKey` for sensitive data
3. **Secure Cookie Settings**: Use `CookieHTTPOnly: true` and appropriate `CookieSameSite` settings
4. **Regular Cleanup**: Ensure garbage collection is running to prevent storage bloat
5. **Key Management**: Store encryption keys securely and rotate them regularly

## Error Handling

The package defines several error types:

```go
var (
    ErrSessionNotFound  = errors.New("session not found")
    ErrInvalidSession   = errors.New("invalid session")
    ErrEncryptionFailed = errors.New("encryption failed")
    ErrDecryptionFailed = errors.New("decryption failed")
)
```

Always handle these errors appropriately in your application:

```go
session, err := manager.GetSession(ctx, r)
if err == session.ErrSessionNotFound {
    // Create new session or redirect to login
} else if err != nil {
    // Handle other errors
    log.Printf("Session error: %v", err)
}
```

## Testing

The package includes comprehensive tests. Run them with:

```bash
go test ./http/session -v
```

For testing your own code that uses sessions, you can use the in-memory storage:

```go
func TestMyHandler(t *testing.T) {
    storage := session.NewMemoryStorage()
    ctx := context.Background()
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
    
    options := session.DefaultOptions()
    manager := session.NewManager(storage, ctx, logger, options)
    
    // Test your handlers...
}
```