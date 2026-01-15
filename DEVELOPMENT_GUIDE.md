# WebSocket Chat App - Development Guide

## Current Status

The application is currently running and functional with the following features:

### Implemented Features
- WebSocket-based real-time communication
- Public messaging (broadcast to all connected users)
- Private messaging (DM between users)
- User join/leave notifications
- Dynamic user list updates
- Basic HTML/CSS/JavaScript frontend
- Concurrent client handling with goroutines
- Buffered channels for message queuing

### Architecture Overview
- **Backend**: Go with Gorilla WebSocket library
- **Server Port**: 8080
- **Message Types**: Public, Private, System
- **Hub Pattern**: Central hub manages all client connections and message routing

---

## Next Development Steps

### Phase 1: Core Improvements (Essential)

#### 1.1 Add Message Persistence
**Why**: Currently, messages are lost when users disconnect or refresh the page.

**Steps**:
1. Choose a database (SQLite for simplicity, PostgreSQL for production)
2. Create a `messages` table schema:
   ```sql
   CREATE TABLE messages (
       id SERIAL PRIMARY KEY,
       type VARCHAR(20),
       username VARCHAR(255),
       content TEXT,
       timestamp TIMESTAMP,
       to_user VARCHAR(255),
       from_user VARCHAR(255)
   );
   ```
3. Add database connection in `main.go`
4. Create a `SaveMessage()` function to persist messages
5. Create a `GetRecentMessages()` function to retrieve message history
6. Send recent messages to newly connected clients

**Files to modify**:
- `main.go`: Add database connection and queries
- `go.mod`: Add database driver dependency

#### 1.2 Implement Message History Loading
**Why**: New users should see recent conversation context.

**Steps**:
1. Modify the Hub's `Register` case to send last 50 messages to new clients
2. Add a new message type: `HistoryMessage`
3. Update frontend to display historical messages differently
4. Add timestamp formatting in the UI

**Files to modify**:
- `main.go`: Update Hub.Run() Register case
- `index.html`: Add history message handling in JavaScript

#### 1.3 Add User Authentication
**Why**: Prevent username impersonation and add security.

**Steps**:
1. Create a `users` table for storing credentials:
   ```sql
   CREATE TABLE users (
       id SERIAL PRIMARY KEY,
       username VARCHAR(255) UNIQUE,
       password_hash VARCHAR(255),
       created_at TIMESTAMP
   );
   ```
2. Add password hashing (use `golang.org/x/crypto/bcrypt`)
3. Create `/register` and `/login` HTTP endpoints
4. Implement JWT or session-based authentication
5. Protect WebSocket endpoint with authentication check
6. Update frontend with login/register forms

**Files to create**:
- `auth.go`: Authentication handlers and functions

**Files to modify**:
- `main.go`: Add auth middleware to WebSocket handler
- `index.html`: Add login/register UI

---

### Phase 2: Enhanced Features (Recommended)

#### 2.1 Add Typing Indicators
**Why**: Improves user experience by showing when someone is typing.

**Steps**:
1. Add new message type: `TypingIndicator`
2. Send typing events from frontend on input change
3. Show/hide "User is typing..." indicator in UI
4. Add debouncing to prevent excessive events

**Files to modify**:
- `main.go`: Add typing indicator handling in Hub
- `index.html`: Add input event listeners and typing UI

#### 2.2 Implement Chat Rooms/Channels
**Why**: Allow users to organize conversations by topics.

**Steps**:
1. Create `Room` struct similar to `Hub`
2. Modify `Hub` to manage multiple rooms
3. Add `/create-room` and `/join-room` endpoints
4. Update message routing to be room-specific
5. Add room list UI in frontend

**Files to modify**:
- `main.go`: Add Room structure and management
- `index.html`: Add room selection UI

#### 2.3 Add File Sharing
**Why**: Enable users to share images and files.

**Steps**:
1. Create `/upload` HTTP endpoint
2. Store files in `uploads/` directory or cloud storage
3. Generate unique filenames and store metadata
4. Send file links as special messages
5. Add file preview for images in UI

**Files to create**:
- `upload.go`: File upload handler

**Files to modify**:
- `main.go`: Add file upload route
- `index.html`: Add file input and preview

#### 2.4 Add Message Reactions (Emoji)
**Why**: Allow quick responses without typing.

**Steps**:
1. Add `reactions` field to `Msg` struct
2. Create new message type: `ReactionMessage`
3. Store reactions in database linked to message ID
4. Add reaction picker UI
5. Display reaction counts on messages

**Files to modify**:
- `main.go`: Add reaction handling
- `index.html`: Add reaction UI components

---

### Phase 3: Advanced Features (Optional)

#### 3.1 Add Read Receipts
**Why**: Show when messages have been read.

**Steps**:
1. Track message delivery and read status
2. Add `ReadReceipt` message type
3. Send read receipts when messages are visible
4. Display checkmarks (single/double) in UI

#### 3.2 Implement Message Search
**Why**: Find old messages quickly.

**Steps**:
1. Add full-text search to database
2. Create `/search` endpoint
3. Add search bar in frontend
4. Display search results with context

#### 3.3 Add User Profiles
**Why**: Personalize user experience.

**Steps**:
1. Extend `users` table with profile fields (avatar, bio, status)
2. Create profile update endpoints
3. Add avatar upload functionality
4. Display user info on hover/click

#### 3.4 Implement Voice/Video Chat
**Why**: Enable richer communication.

**Steps**:
1. Integrate WebRTC for peer-to-peer connections
2. Add signaling through WebSocket
3. Create call UI components
4. Handle ICE candidates and SDP offers

---

### Phase 4: Production Readiness

#### 4.1 Add Comprehensive Testing
**Steps**:
1. Write unit tests for Hub logic
2. Add integration tests for WebSocket handlers
3. Create frontend tests with testing library
4. Set up CI/CD pipeline

**Files to create**:
- `main_test.go`: Backend tests
- `test/`: Test directory for integration tests

#### 4.2 Improve Error Handling
**Steps**:
1. Add structured logging (use `logrus` or `zap`)
2. Implement proper error responses
3. Add client-side error handling
4. Create error monitoring (Sentry, etc.)

#### 4.3 Add Security Enhancements
**Steps**:
1. Implement rate limiting
2. Add input validation and sanitization
3. Use HTTPS/WSS in production
4. Add CORS configuration
5. Implement CSP headers
6. Add password strength requirements

#### 4.4 Performance Optimization
**Steps**:
1. Add Redis for caching user lists
2. Implement message pagination
3. Add database indexing
4. Use connection pooling
5. Add load balancing for horizontal scaling
6. Implement message compression

#### 4.5 Add Monitoring and Analytics
**Steps**:
1. Add Prometheus metrics
2. Create Grafana dashboards
3. Track active users, messages/second
4. Add health check endpoint
5. Implement alerting for errors

---

## Recommended Order of Implementation

1. **Start Here**: Message Persistence (1.1) + Message History (1.2)
2. User Authentication (1.3)
3. Typing Indicators (2.1)
4. Chat Rooms (2.2)
5. File Sharing (2.3)
6. Testing (4.1)
7. Security Enhancements (4.3)
8. Additional features based on your needs

---

## Development Workflow

### Before Starting Each Feature

1. Create a new git branch: `git checkout -b feature/feature-name`
2. Read relevant documentation
3. Plan the database schema changes if needed
4. Identify all files that need modification

### During Development

1. Write the backend logic first
2. Test with curl or Postman
3. Update the frontend
4. Test in browser with multiple tabs
5. Handle edge cases and errors

### After Completing Each Feature

1. Test thoroughly with multiple clients
2. Update README.md with new features
3. Commit changes: `git commit -m "Add feature-name"`
4. Merge to main branch

---

## Useful Resources

- Gorilla WebSocket docs: https://pkg.go.dev/github.com/gorilla/websocket
- Go database/sql tutorial: https://go.dev/doc/database/
- WebSocket API (MDN): https://developer.mozilla.org/en-US/docs/Web/API/WebSocket
- JWT authentication in Go: https://github.com/golang-jwt/jwt

---

## Quick Commands

Start server: `go run main.go`

Run tests: `go test ./...`

Build binary: `go build -o chat-server`

Format code: `go fmt ./...`

Check dependencies: `go mod tidy`

---

## Current File Structure
```
websocket-chat/
├── main.go           # Server code with Hub and Client logic
├── index.html        # Frontend HTML/CSS/JavaScript
├── go.mod           # Go module dependencies
├── go.sum           # Dependency checksums
└── README.md        # Project overview
```

## Recommended Future Structure
```
websocket-chat/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── hub/
│   │   ├── hub.go
│   │   └── client.go
│   ├── auth/
│   │   └── auth.go
│   ├── database/
│   │   └── db.go
│   └── handlers/
│       ├── websocket.go
│       └── http.go
├── static/
│   ├── index.html
│   ├── css/
│   └── js/
├── migrations/
│   └── 001_initial.sql
├── tests/
│   └── integration_test.go
├── go.mod
├── go.sum
└── README.md
```

---

## Notes

- The current implementation uses in-memory storage, so all data is lost on server restart
- No authentication means anyone can use any username
- No message size limits or rate limiting currently implemented
- Consider adding graceful shutdown handling
- WebSocket connections are not encrypted in development (use WSS in production)
