package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type MsgType string

const (
	PublicMessage  MsgType = "public"
	PrivateMessage MsgType = "Private"
	SystemMessage  MsgType = "system"
	DocList        MsgType = "doc-list"
	DocOpen        MsgType = "doc-open"
	DocCreate      MsgType = "doc-create"
	DocContent     MsgType = "doc-content"
	DocUpdate      MsgType = "doc-update"
	UserJoined     MsgType = "user-joined"
	UserLeft       MsgType = "user-left"
)

type Msg struct {
	Type     MsgType   `json:"type"`
	Username string    `json:"username"`
	Content  string    `json:"content"`
	Time     time.Time `json:"time"`
	UserList []string  `json:"user_list"`
	IsSystem bool      `json:"is_system"`
	To       string    `json:"to,omitempty"`
	From     string    `json:"from,omitempty"`

	// Document-related fields
	DocumentID string      `json:"documentID,omitempty"`
	Documents  []Document  `json:"documents,omitempty"`
	Document   *Document   `json:"document,omitempty"`
	Name       string      `json:"name,omitempty"`
	Language   string      `json:"language,omitempty"`
	Color      string      `json:"color,omitempty"`
}

type Client struct {
	Username           string
	Conn               *websocket.Conn
	Send               chan Msg
	CurrentDocumentID  string // Track which document the user is editing
}

type Hub struct {
	Clients         map[*Client]bool
	BroadCast       chan Msg
	Private         chan Msg
	Register        chan *Client
	Unregister      chan *Client

	// Document editing sessions
	DocumentClients map[string]map[*Client]bool // documentID -> set of clients
	DocumentEdits   chan Msg                     // Channel for document edit broadcasts
}

func NewHub() *Hub {
	return &Hub{
		Clients:         make(map[*Client]bool),
		BroadCast:       make(chan Msg, 256),
		Private:         make(chan Msg, 256),
		Register:        make(chan *Client, 256),
		Unregister:      make(chan *Client, 256),
		DocumentClients: make(map[string]map[*Client]bool),
		DocumentEdits:   make(chan Msg, 256),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Clients[client] = true
			log.Printf("Client %s connected. Total Clients %d", client.Username, len(h.Clients))

			// Send recent message history to new client
			history, err := GetRecentMessages(50)
			if err != nil {
				log.Printf("Failed to get message history: %v", err)
			} else {
				for _, msg := range history {
					msg.UserList = h.GetUserNames()
					select {
					case client.Send <- msg:
					default:
						log.Printf("Failed to send history message to %s", client.Username)
					}
				}
			}

			welcomeMsg := Msg{
				Type:     SystemMessage,
				Username: "System",
				Content:  client.Username + " joined the chat",
				Time:     time.Now(),
				IsSystem: true,
				UserList: h.GetUserNames(),
			}
			h.BroadCast <- welcomeMsg

		case client := <-h.Unregister:
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)
				log.Printf("Client %s disconnected. Total Clients %d", client.Username, len(h.Clients))

				// Remove from document editing session
				if client.CurrentDocumentID != "" {
					if clients, ok := h.DocumentClients[client.CurrentDocumentID]; ok {
						delete(clients, client)

						// Notify other users in the document
						leaveMsg := Msg{
							Type:       UserLeft,
							DocumentID: client.CurrentDocumentID,
							Username:   client.Username,
						}
						for c := range clients {
							select {
							case c.Send <- leaveMsg:
							default:
							}
						}
					}
				}

				goodbyeMsg := Msg{
					Type:     SystemMessage,
					Username: "System",
					Content:  client.Username + " left the chat",
					Time:     time.Now(),
					IsSystem: true,
					UserList: h.GetUserNames(),
				}
				h.BroadCast <- goodbyeMsg
			}

		case message := <-h.BroadCast:
			log.Printf("Broadcasting message from %s: %s", message.Username, message.Content)

			// Save message to database
			if err := SaveMessage(message); err != nil {
				log.Printf("Failed to save message: %v", err)
			}

			// Always update user list for all messages
			message.UserList = h.GetUserNames()

			// Send to ALL connected clients
			for client := range h.Clients {
				select {
				case client.Send <- message:
					log.Printf("Message sent to %s", client.Username)
				default:
					log.Printf("Failed to send to %s, closing connection", client.Username)
					close(client.Send)
					delete(h.Clients, client)
				}
			}

		case privateMsg := <-h.Private:
			log.Printf("Sending private messages from %s to %s", privateMsg.From, privateMsg.To)

			// Save private message to database
			if err := SaveMessage(privateMsg); err != nil {
				log.Printf("Failed to save private message: %v", err)
			}

			var sender, recipient *Client
			for client := range h.Clients {
				if client.Username == privateMsg.From {
					sender = client
				}
				if client.Username == privateMsg.To {
					recipient = client
				}
			}
			if sender != nil {
				select {
				case sender.Send <- privateMsg:
					log.Printf("Private message sent to sender %s", sender.Username)
				default:
					log.Printf("Failed to send private message to sender %s", sender.Username)
				}
			}

			if recipient != nil {
				select {
				case recipient.Send <- privateMsg:
					log.Printf("Private message sent to recipient %s", recipient.Username)
				default:
					log.Printf("Failed to send private message to recipient %s", recipient.Username)
				}
			} else {
				// Recipient not found, send error message to sender
				if sender != nil {
					errorMsg := Msg{
						Type:     SystemMessage,
						Username: "System",
						Content:  "User '" + privateMsg.To + "' is not online",
						Time:     time.Now(),
						IsSystem: true,
					}
					select {
					case sender.Send <- errorMsg:
					default:
					}
				}
			}

		case editMsg := <-h.DocumentEdits:
			// Broadcast document edit to all users editing the same document
			log.Printf("Broadcasting edit for document %s from %s", editMsg.DocumentID, editMsg.Username)

			if clients, ok := h.DocumentClients[editMsg.DocumentID]; ok {
				for client := range clients {
					// Don't send back to the sender
					if client.Username != editMsg.Username {
						select {
						case client.Send <- editMsg:
							log.Printf("Edit sent to %s", client.Username)
						default:
							log.Printf("Failed to send edit to %s", client.Username)
						}
					}
				}
			}
		}
	}
}

func (h *Hub) GetUserNames() []string {
	var usernames []string
	for client := range h.Clients {
		usernames = append(usernames, client.Username)
	}
	return usernames
}

// Generate a consistent color for each user based on their username
func generateUserColor(username string) string {
	colors := []string{
		"#FF6B6B", "#4ECDC4", "#45B7D1", "#FFA07A",
		"#98D8C8", "#F7DC6F", "#BB8FCE", "#85C1E2",
		"#F8B739", "#52B788", "#E76F51", "#8E44AD",
	}

	// Simple hash to pick a color consistently for each username
	hash := 0
	for _, char := range username {
		hash += int(char)
	}
	return colors[hash%len(colors)]
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handleWebSocket(hub *Hub, w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		log.Printf("WebSocket connection attempt without username")
		http.Error(w, "Username required", http.StatusBadRequest)
		return
	}

	log.Printf("WebSocket upgrade request from %s", username)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error for %s: %s", username, err)
		return
	}

	log.Printf("WebSocket connection established for %s", username)

	client := &Client{
		Username: username,
		Conn:     conn,
		Send:     make(chan Msg, 256),
	}

	log.Printf("Starting goroutines for %s", username)
	go client.readMessages(hub)
	go client.writeMessages()

	log.Printf("Registering client %s", username)
	// Move registration after starting goroutines to prevent blocking
	hub.Register <- client
}

func (c *Client) readMessages(hub *Hub) {
	defer func() {
		log.Printf("readMessages defer called for %s", c.Username)
		hub.Unregister <- c
		c.Conn.Close()
	}()

	log.Printf("Starting to read messages for %s", c.Username)

	for {
		var msg Msg
		err := c.Conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("Read error for %s: %v", c.Username, err)
			break
		}
		log.Printf("Received message from %s, type: %s", c.Username, msg.Type)
		msg.Username = c.Username
		msg.Time = time.Now()

		// Handle different message types
		switch msg.Type {
		case DocList:
			// Client requests list of documents
			c.handleDocumentList()

		case DocOpen:
			// Client wants to open a document
			c.handleDocumentOpen(msg.DocumentID, hub)

		case DocCreate:
			// Client wants to create a new document
			c.handleDocumentCreate(msg.Name, msg.Language, hub)

		case DocUpdate:
			// Client updated document content - broadcast to other users
			msg.Username = c.Username
			hub.DocumentEdits <- msg

		case PrivateMessage:
			if msg.To != "" {
				msg.From = c.Username
				log.Printf("Received private message from %s to %s: %s", c.Username, msg.To, msg.Content)
				hub.Private <- msg
			}

		default:
			// Public message
			msg.Type = PublicMessage
			msg.IsSystem = false
			log.Printf("Received public message from %s: %s", c.Username, msg.Content)
			hub.BroadCast <- msg
		}
	}
}

func (c *Client) writeMessages() {
	defer func() {
		log.Printf("writeMessages defer called for %s", c.Username)
		c.Conn.Close()
	}()

	log.Printf("Starting to write messages for %s", c.Username)

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				log.Printf("Send channel closed for %s", c.Username)
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			log.Printf("Writing message to %s: %s", c.Username, message.Content)
			if err := c.Conn.WriteJSON(message); err != nil {
				log.Printf("Write error for %s: %v", c.Username, err)
				return
			}
		}
	}
}

// Document operation handlers

func (c *Client) handleDocumentList() {
	documents, err := GetAllDocuments()
	if err != nil {
		log.Printf("Error getting documents: %v", err)
		return
	}

	response := Msg{
		Type:      DocList,
		Documents: documents,
	}

	c.Conn.WriteJSON(response)
}

func (c *Client) handleDocumentOpen(docID string, hub *Hub) {
	doc, err := GetDocument(docID)
	if err != nil {
		log.Printf("Error getting document %s: %v", docID, err)
		return
	}

	if doc == nil {
		log.Printf("Document %s not found", docID)
		return
	}

	// Update client's current document
	c.CurrentDocumentID = docID

	// Add client to document's editing session
	if hub.DocumentClients[docID] == nil {
		hub.DocumentClients[docID] = make(map[*Client]bool)
	}
	hub.DocumentClients[docID][c] = true

	log.Printf("%s opened document %s", c.Username, doc.Name)

	// Send document content to the client
	response := Msg{
		Type:       DocContent,
		DocumentID: doc.ID,
		Name:       doc.Name,
		Content:    doc.Content,
		Language:   doc.Language,
	}
	c.Conn.WriteJSON(response)

	// Notify other users editing this document
	joinMsg := Msg{
		Type:       UserJoined,
		DocumentID: docID,
		Username:   c.Username,
		Color:      generateUserColor(c.Username),
	}

	for client := range hub.DocumentClients[docID] {
		if client != c {
			client.Send <- joinMsg
		}
	}
}

func (c *Client) handleDocumentCreate(name, language string, hub *Hub) {
	doc, err := CreateDocument(name, language, c.Username)
	if err != nil {
		log.Printf("Error creating document: %v", err)
		return
	}

	log.Printf("Document created: %s by %s", doc.Name, c.Username)

	// Send the new document back to the creator
	response := Msg{
		Type:       DocContent,
		DocumentID: doc.ID,
		Name:       doc.Name,
		Content:    doc.Content,
		Language:   doc.Language,
	}
	c.Conn.WriteJSON(response)

	// Notify all clients about the new document
	listMsg := Msg{
		Type: DocList,
	}
	hub.BroadCast <- listMsg
}

func (c *Client) handleDocumentUpdate(docID, content string, hub *Hub) {
	err := UpdateDocument(docID, content)
	if err != nil {
		log.Printf("Error updating document %s: %v", docID, err)
		return
	}

	log.Printf("Document %s updated by %s", docID, c.Username)

	// Broadcast the update to other users editing the same document
	// TODO: We'll implement proper real-time sync with Yjs in next step
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "index.html")
}

func serveEditor(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "editor.html")
}

func main() {
	// Initialize database
	if err := InitDB(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	hub := NewHub()
	go hub.Run()

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/editor", serveEditor)
	http.HandleFunc("/register", HandleRegister)
	http.HandleFunc("/login", HandleLogin)
	http.HandleFunc("/ws", AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(hub, w, r)
	}))

	log.Println("Server starting on :8080")
	log.Println("Chat: http://localhost:8080")
	log.Println("Editor: http://localhost:8080/editor")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
