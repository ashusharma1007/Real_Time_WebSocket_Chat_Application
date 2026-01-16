# Real-Time WebSocket Chat & Code Editor

A full-featured real-time communication platform built with Go, featuring both instant messaging and collaborative code editing capabilities. This project demonstrates concurrent programming patterns, WebSocket management, and real-time data synchronization.

## Features

### Chat Application
- **Public & Private Messaging** - Send messages to everyone or have private conversations
- **Live User Tracking** - See who's online in real-time
- **Message History** - Persistent storage with SQLite, never lose your conversations
- **User Presence** - Get notified when users join or leave
- **Beautiful UI** - Clean, modern interface with smooth animations

### Collaborative Code Editor
- **Monaco Editor** - The same editor that powers VS Code
- **Real-Time Collaboration** - Multiple users can edit the same file simultaneously
- **Multi-Language Support** - Syntax highlighting for 50+ programming languages
- **File Management** - Create, edit, and manage multiple documents
- **Auto-Save** - Changes are automatically persisted to the database
- **User Presence** - See who else is editing each document

## Tech Stack

- **Backend**: Go (Golang)
- **Real-Time**: WebSockets with concurrent channel-based architecture
- **Database**: SQLite with WAL mode for concurrent access
- **Authentication**: JWT tokens with bcrypt password hashing
- **Frontend**: Vanilla JavaScript, Monaco Editor
- **Architecture**: Hub pattern for managing WebSocket connections

## Architecture Highlights

The application uses a hub-based architecture where a central hub manages all WebSocket connections through Go channels. This design allows for:
- Efficient concurrent message broadcasting
- Clean separation of concerns
- Scalable connection management
- Thread-safe operations using Go's channel primitives

## Getting Started

### Prerequisites
- Go 1.24 or higher
- Modern web browser

### Installation

1. Clone the repository
```bash
git clone https://github.com/ashusharma1007/Go_chat_app.git
cd Go_chat_app
```

2. Install dependencies
```bash
go mod download
```

3. Run the server
```bash
go run *.go
```

4. Open your browser
- Chat: http://localhost:8080
- Code Editor: http://localhost:8080/editor

## Usage

### Chat Application
1. Open http://localhost:8080
2. Register a new account or login
3. Start chatting in the public chat
4. Click on a username to send private messages

### Collaborative Code Editor
1. Open http://localhost:8080/editor
2. Login with your credentials
3. Click "New File" to create a document
4. Start coding! Changes sync in real-time across all connected users
5. Open multiple browser tabs to see real-time collaboration in action

## Project Structure

```
├── main.go           # WebSocket server and hub implementation
├── auth.go           # JWT authentication and user management
├── database.go       # SQLite database operations
├── documents.go      # Document CRUD operations
├── index.html        # Chat application frontend
├── editor.html       # Code editor frontend
└── go.mod            # Go dependencies
```

## How It Works

### WebSocket Communication
The server maintains persistent WebSocket connections with all clients. Messages are routed through a central hub that manages:
- User registration and deregistration
- Message broadcasting to all users or specific users
- Document edit synchronization
- User presence notifications

### Data Flow
1. Client connects via WebSocket
2. Server authenticates using JWT token
3. Client joins relevant channels (chat rooms, documents)
4. Messages/edits are sent to hub via Go channels
5. Hub broadcasts to appropriate recipients
6. Changes are persisted to SQLite database

### Concurrency
The hub runs in its own goroutine, with each client connection handled by two additional goroutines (one for reading, one for writing). This allows the server to handle hundreds of concurrent connections efficiently.

## Security Features

- **JWT Authentication** - Secure token-based authentication
- **Password Hashing** - bcrypt with configurable cost factor
- **Session Management** - Automatic token expiration and renewal
- **Input Validation** - Server-side validation of all inputs
- **Concurrent Access** - SQLite WAL mode prevents database locks

## Performance

- Handles 100+ concurrent connections
- Sub-100ms message latency
- Buffered channels prevent blocking
- Efficient memory usage with connection pooling

## Future Enhancements

- [ ] Migrate to PostgreSQL + Redis for better scalability
- [ ] Add CRDT-based conflict resolution (Yjs integration)
- [ ] Implement cursor position synchronization
- [ ] Add voice/video chat capabilities
- [ ] Deploy to cloud (AWS/GCP/Azure)
- [ ] Add end-to-end encryption for private messages

## Contributing

This is a personal learning project, but suggestions and feedback are welcome! Feel free to open an issue or submit a pull request.

## License

MIT License - feel free to use this code for learning purposes.

## Acknowledgments

Built as a learning project to explore:
- Concurrent programming patterns in Go
- WebSocket protocol and real-time communication
- Hub-based architecture for managing connections
- Collaborative editing algorithms

---

**Note**: This project is designed for educational purposes and local development. For production use, additional security measures and infrastructure would be recommended.
