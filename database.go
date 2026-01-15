package main

import (
	"database/sql"
	"log"
	"time"

	_ "modernc.org/sqlite"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB

// InitDB initializes the database connection and creates tables
func InitDB() error {
	var err error
	db, err = sql.Open("sqlite", "./chat.db")
	if err != nil {
		return err
	}

	// Test the connection
	if err = db.Ping(); err != nil {
		return err
	}

	// Enable WAL mode for better concurrency
	_, err = db.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		return err
	}

	// Set busy timeout to 5 seconds
	_, err = db.Exec("PRAGMA busy_timeout=5000;")
	if err != nil {
		return err
	}

	// Create messages table
	createMessagesTable := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		type TEXT NOT NULL,
		username TEXT NOT NULL,
		content TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		to_user TEXT,
		from_user TEXT,
		is_system BOOLEAN DEFAULT 0
	);`

	if _, err = db.Exec(createMessagesTable); err != nil {
		return err
	}

	// Create users table
	createUsersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at DATETIME NOT NULL
	);`

	if _, err = db.Exec(createUsersTable); err != nil {
		return err
	}

	// Create documents table
	if err = InitDocumentTables(); err != nil {
		return err
	}

	log.Println("Database initialized successfully")
	return nil
}

// SaveMessage saves a message to the database
func SaveMessage(msg Msg) error {
	query := `
		INSERT INTO messages (type, username, content, timestamp, to_user, from_user, is_system)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := db.Exec(query, msg.Type, msg.Username, msg.Content, msg.Time, msg.To, msg.From, msg.IsSystem)
	return err
}

// GetRecentMessages retrieves the last N messages from the database
func GetRecentMessages(limit int) ([]Msg, error) {
	query := `
		SELECT type, username, content, timestamp, to_user, from_user, is_system
		FROM messages
		ORDER BY id DESC
		LIMIT ?
	`

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Msg
	for rows.Next() {
		var msg Msg
		var toUser, fromUser sql.NullString

		err := rows.Scan(&msg.Type, &msg.Username, &msg.Content, &msg.Time, &toUser, &fromUser, &msg.IsSystem)
		if err != nil {
			return nil, err
		}

		if toUser.Valid {
			msg.To = toUser.String
		}
		if fromUser.Valid {
			msg.From = fromUser.String
		}

		messages = append(messages, msg)
	}

	// Reverse the slice to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// CreateUser creates a new user with hashed password
func CreateUser(username, password string) error {
	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	query := `INSERT INTO users (username, password_hash, created_at) VALUES (?, ?, ?)`
	_, err = db.Exec(query, username, string(hashedPassword), time.Now())
	return err
}

// ValidateUser checks if username and password are correct
func ValidateUser(username, password string) (bool, error) {
	var hashedPassword string
	query := `SELECT password_hash FROM users WHERE username = ?`

	err := db.QueryRow(query, username).Scan(&hashedPassword)
	if err == sql.ErrNoRows {
		return false, nil // User not found
	}
	if err != nil {
		return false, err
	}

	// Compare the password with the hash
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return false, nil // Password doesn't match
	}

	return true, nil
}

// UserExists checks if a username already exists
func UserExists(username string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)`
	err := db.QueryRow(query, username).Scan(&exists)
	return exists, err
}
