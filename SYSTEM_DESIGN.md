# Collaborative Code Editor - System Design

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         USER BROWSERS                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │   Browser 1  │  │   Browser 2  │  │   Browser 3  │          │
│  │  (Alice)     │  │  (Bob)       │  │  (Charlie)   │          │
│  │              │  │              │  │              │          │
│  │ Monaco       │  │ Monaco       │  │ Monaco       │          │
│  │ Editor       │  │ Editor       │  │ Editor       │          │
│  │   +          │  │   +          │  │   +          │          │
│  │ Yjs Client   │  │ Yjs Client   │  │ Yjs Client   │          │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘          │
│         │                 │                 │                   │
└─────────┼─────────────────┼─────────────────┼───────────────────┘
          │    WebSocket    │   WebSocket     │  WebSocket
          │                 │                 │
┌─────────┼─────────────────┼─────────────────┼───────────────────┐
│         ▼                 ▼                 ▼                   │
│  ┌──────────────────────────────────────────────────┐          │
│  │         Go WebSocket Server (Hub)                │          │
│  │                                                   │          │
│  │  • Route messages between users                  │          │
│  │  • Manage user presence                          │          │
│  │  • Handle document operations                    │          │
│  │  • Broadcast cursor positions                    │          │
│  │  • Authentication & Authorization                │          │
│  └─────────────────┬────────────────────────────────┘          │
│                    │                                            │
│  ┌─────────────────▼────────────────────────────────┐          │
│  │         Yjs WebSocket Provider (Optional)        │          │
│  │                                                   │          │
│  │  • CRDT state synchronization                    │          │
│  │  • Conflict-free merge of edits                  │          │
│  │  • Awareness (cursor positions)                  │          │
│  └─────────────────┬────────────────────────────────┘          │
│                    │                                            │
└────────────────────┼────────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────────┐
│                   PostgreSQL Database                           │
│                                                                  │
│  Tables:                                                         │
│  • documents (id, name, content, language, version)             │
│  • document_operations (history of all changes)                 │
│  • users (authentication)                                       │
│  • document_collaborators (who's editing what)                  │
└──────────────────────────────────────────────────────────────────┘
```

---

## How It Works: Step-by-Step

### 1. User Opens Editor

```
User (Browser)                    Server                    Database
     │                              │                           │
     │──(1) HTTP GET /)────────────▶│                           │
     │                              │                           │
     │◀─(2) HTML + Monaco Editor────│                           │
     │                              │                           │
     │──(3) Login/Register──────────▶│                           │
     │                              │                           │
     │                              │──(4) Validate User───────▶│
     │                              │                           │
     │                              │◀─(5) User Data───────────│
     │                              │                           │
     │◀─(6) JWT Token───────────────│                           │
     │                              │                           │
```

### 2. Establish WebSocket Connection

```
User                              Server                    Yjs
 │                                  │                        │
 │──(7) ws://server/ws?token=xyz───▶│                        │
 │                                  │                        │
 │                                  │──Verify JWT Token      │
 │                                  │                        │
 │◀─(8) WebSocket Connected─────────│                        │
 │                                  │                        │
 │──(9) Request Document List───────▶│                        │
 │                                  │                        │
 │                                  │──Query DB for docs─────▶
 │                                  │                        │
 │◀─(10) Available Documents────────│                        │
 │     [main.go, server.go...]      │                        │
 │                                  │                        │
```

### 3. Open a Document

```
User A                            Server                    User B
  │                                 │                         │
  │──(11) Open "main.go"───────────▶│                         │
  │                                 │                         │
  │                                 │──Load from DB           │
  │                                 │                         │
  │◀─(12) Document Content──────────│                         │
  │     (full text of main.go)      │                         │
  │                                 │                         │
  │──(13) Initialize Yjs Doc────────▶│                         │
  │                                 │                         │
  │                                 │──(14) Notify "User A    │
  │                                 │       joined main.go"───▶│
  │                                 │                         │
  │                                 │◀─(15) User B already    │
  │◀─(16) User B's cursor position──│       editing main.go───│
  │                                 │                         │
```

### 4. Real-Time Editing (The Magic!)

```
User A Types "hello"              Server                    User B
  │                                 │                         │
  │──(17) Yjs Operation:            │                         │
  │     { type: "insert",           │                         │
  │       position: 10,             │                         │
  │       text: "hello" }──────────▶│                         │
  │                                 │                         │
  │                                 │──(18) Transform if       │
  │                                 │       needed (OT/CRDT)  │
  │                                 │                         │
  │                                 │──(19) Broadcast to all  │
  │                                 │       users on main.go──▶│
  │                                 │                         │
  │                                 │                         │◀─ User B sees
  │                                 │                         │   "hello" appear
  │                                 │                         │   in real-time!
  │                                 │                         │
```

#### What Happens When Two Users Edit Simultaneously?

**Scenario**: Both users type at the same time

```
Initial State: "The cat"

User A: Types "quick " at position 4
User B: Types "brown " at position 4

User A sees: "The quick cat"
User B sees: "The brown cat"

Server receives BOTH operations!
```

**Without Conflict Resolution (Bad)**:
```
Result: "The brown quick cat"  ❌ (Wrong! One edit lost)
```

**With Yjs/CRDT (Good)**:
```
Yjs Algorithm:
1. Detect concurrent edits
2. Assign ordering (based on client ID + timestamp)
3. Transform operations:
   - User A's "quick " stays at position 4
   - User B's "brown " transforms to position 10

Result: "The quick brown cat"  ✅ (Correct! Both edits preserved)
```

---

## Key Components Explained

### Component 1: Monaco Editor (Frontend)

**What**: VS Code's editor as a web component

**Why**:
- Syntax highlighting for 50+ languages
- IntelliSense (autocomplete)
- Minimap, find/replace
- Same editor developers use daily

**How to Use**:
```javascript
// Create editor instance
const editor = monaco.editor.create(document.getElementById('container'), {
    value: 'function hello() {\n\tconsole.log("Hello!");\n}',
    language: 'javascript',
    theme: 'vs-dark'
});

// Listen to changes
editor.onDidChangeModelContent((e) => {
    console.log('User edited:', e.changes);
});

// Get current content
const content = editor.getValue();
```

---

### Component 2: Yjs (CRDT Library)

**What**: Conflict-Free Replicated Data Type library

**Why**: Automatically merges concurrent edits without conflicts

**How It Works**:

Every character has a unique ID:
```
Text: "Hello"
IDs:  [A1, A2, A3, A4, A5]

User A inserts "X" at position 2:
Text: "HeXllo"
IDs:  [A1, A2, B1, A3, A4, A5]

User B (hasn't seen X yet) inserts "Y" at position 2:
Text: "HeYllo" (their view)
IDs:  [A1, A2, C1, A3, A4, A5]

When B receives A's operation:
- B knows "X" (B1) comes from User A
- B knows "Y" (C1) comes from User B
- Yjs sorts by (UserID, Timestamp)
- Result: "HeXYllo" or "HeYXllo" (consistent for all users!)
```

**Basic Yjs Code**:
```javascript
// Create a shared document
const ydoc = new Y.Doc();

// Create a shared text type
const ytext = ydoc.getText('monaco');

// Listen to remote changes
ytext.observe(event => {
    console.log('Remote user changed:', event.delta);
});

// Make local changes
ytext.insert(0, 'Hello World');
```

---

### Component 3: WebSocket Provider

**What**: Connects Yjs to your server via WebSocket

**Two Options**:

#### Option A: Use y-websocket (JavaScript Server)
```javascript
// Frontend
import { WebsocketProvider } from 'y-websocket'

const provider = new WebsocketProvider(
    'ws://localhost:8080',
    'my-document',
    ydoc
);
```

Requires: Node.js server (separate from Go)

#### Option B: Integrate with Go Server (What We'll Do)
```javascript
// Frontend: Connect to your existing Go server
const ws = new WebSocket('ws://localhost:8080/ws?token=' + authToken);

// Bridge Yjs to WebSocket manually
ydoc.on('update', update => {
    ws.send(update); // Send Yjs updates through your Go server
});

ws.onmessage = (msg) => {
    Y.applyUpdate(ydoc, msg.data); // Apply updates from other users
};
```

Advantage: Keep everything in Go!

---

### Component 4: Go Server (Backend)

**Role**:
1. Authenticate users
2. Route messages between clients
3. Store documents in database
4. Manage user presence

**Modified Hub**:
```go
type Hub struct {
    // Existing
    Clients    map[*Client]bool
    Register   chan *Client
    Unregister chan *Client

    // NEW: Document-specific routing
    Documents       map[string]*Document        // docID -> Document
    DocumentClients map[string]map[*Client]bool // docID -> clients

    // NEW: Operation channels
    YjsUpdates chan YjsUpdate // Forward Yjs updates
    Cursors    chan CursorPos  // Broadcast cursor positions
}

type YjsUpdate struct {
    DocumentID string
    Update     []byte  // Yjs binary update
    FromClient *Client
}

type CursorPos struct {
    DocumentID string
    Username   string
    Line       int
    Column     int
    Color      string
}
```

**How Server Routes Messages**:
```go
func (h *Hub) Run() {
    for {
        select {
        case update := <-h.YjsUpdates:
            // Send this Yjs update to all clients editing the same document
            for client := range h.DocumentClients[update.DocumentID] {
                if client != update.FromClient { // Don't send back to sender
                    client.Send <- update.Update
                }
            }

        case cursor := <-h.Cursors:
            // Broadcast cursor position
            for client := range h.DocumentClients[cursor.DocumentID] {
                client.Send <- cursor
            }
        }
    }
}
```

---

### Component 5: PostgreSQL Database

**Schema**:

```sql
-- Store documents
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    content TEXT,
    language VARCHAR(50) DEFAULT 'plaintext',
    created_by VARCHAR(255) REFERENCES users(username),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Store who's editing what (active sessions)
CREATE TABLE active_sessions (
    document_id UUID REFERENCES documents(id),
    username VARCHAR(255) REFERENCES users(username),
    cursor_line INT DEFAULT 0,
    cursor_column INT DEFAULT 0,
    color VARCHAR(7),
    last_seen TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (document_id, username)
);

-- Optional: Store operation history for replay/undo
CREATE TABLE document_history (
    id SERIAL PRIMARY KEY,
    document_id UUID REFERENCES documents(id),
    username VARCHAR(255),
    operation JSONB,  -- Store Yjs update or operation
    created_at TIMESTAMP DEFAULT NOW()
);
```

---

## Data Flow Examples

### Example 1: User Types a Character

```
Step 1: User presses 'A' in Monaco Editor
  ↓
Step 2: Monaco fires onDidChangeModelContent event
  ↓
Step 3: Yjs captures change, creates update
  {
    type: "insert",
    position: 42,
    content: "A"
  }
  ↓
Step 4: Update sent to Go server via WebSocket
  ↓
Step 5: Go server broadcasts to other users editing same doc
  ↓
Step 6: Other users' browsers receive update
  ↓
Step 7: Yjs applies update to their document
  ↓
Step 8: Monaco editor updates to show 'A'
```

**Time**: ~50-100ms total (feels instant!)

---

### Example 2: User Moves Cursor

```
Step 1: User clicks or moves cursor
  ↓
Step 2: Monaco fires onDidChangeCursorPosition
  ↓
Step 3: Send cursor position to server
  {
    type: "cursor",
    documentID: "main.go",
    line: 15,
    column: 8,
    username: "Alice",
    color: "#FF0000"
  }
  ↓
Step 4: Server broadcasts to other users
  ↓
Step 5: Other users show Alice's cursor at line 15, column 8
```

---

## Message Types

### 1. Document Operations (Yjs Updates)
```javascript
{
    type: "yjs-update",
    documentID: "abc-123",
    update: <binary data>  // Yjs internal format
}
```

### 2. Cursor Positions
```javascript
{
    type: "cursor",
    documentID: "abc-123",
    username: "Alice",
    line: 10,
    column: 5,
    color: "#FF6B6B"
}
```

### 3. User Presence
```javascript
{
    type: "user-joined",
    documentID: "abc-123",
    username: "Bob",
    color: "#4ECDC4"
}
```

### 4. Document Management
```javascript
// List documents
{
    type: "doc-list",
}

// Open document
{
    type: "doc-open",
    documentID: "abc-123"
}

// Save document
{
    type: "doc-save",
    documentID: "abc-123"
}
```

---

## Why This Architecture Works

### Scalability
- Each document has its own set of clients
- Updates only sent to users editing that document
- Can handle 100s of documents with 1000s of users

### Performance
- Binary Yjs updates are small (~10-100 bytes per keystroke)
- WebSocket keeps connection open (no HTTP overhead)
- Cursor updates throttled (send max once per 100ms)

### Reliability
- Yjs guarantees eventual consistency
- If user disconnects and reconnects, Yjs syncs state
- Database persists documents (can reload on server restart)

### Correctness
- CRDT algorithm mathematically proven to converge
- No race conditions or lost updates
- Users can't "corrupt" each other's work

---

## Comparison: With vs Without Yjs

### Without Yjs (Manual Implementation):
```javascript
// User A types "X" at position 5
ws.send({ type: "insert", pos: 5, text: "X" });

// User B types "Y" at position 5 (at the same time!)
ws.send({ type: "insert", pos: 5, text: "Y" });

// Server receives both. What to do?
// Option 1: First one wins → User B's edit lost ❌
// Option 2: Last one wins → User A's edit lost ❌
// Option 3: Put both → But which comes first? Need complex logic!

// You'd need to implement:
// - Operation transformation (100+ edge cases)
// - Conflict detection
// - Version vectors
// - Undo/redo that works across users
// = 1000s of lines of complex code!
```

### With Yjs:
```javascript
// Yjs handles ALL of this automatically!
const ytext = ydoc.getText('monaco');
ytext.insert(5, 'X');  // That's it! Yjs does the rest.

// Yjs will:
// - Assign unique IDs to every character
// - Transform concurrent operations automatically
// - Guarantee all users see the same final state
// - Handle undo/redo correctly
// = 10 lines of code!
```

---

## Next Steps

Now that you understand the architecture, let's implement it step by step:

**Phase 1**: Replace chat UI with Monaco Editor (no sync yet)
**Phase 2**: Add Yjs and test with single document
**Phase 3**: Add multi-document support
**Phase 4**: Add cursor positions and presence

Ready to start coding? 🚀
