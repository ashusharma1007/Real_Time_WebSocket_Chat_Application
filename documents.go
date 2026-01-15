package main

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Document represents a code file/document
type Document struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	Language  string    `json:"language"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// InitDocumentTables creates the documents table
func InitDocumentTables() error {
	createDocumentsTable := `
	CREATE TABLE IF NOT EXISTS documents (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		content TEXT DEFAULT '',
		language TEXT DEFAULT 'plaintext',
		created_by TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);`

	_, err := db.Exec(createDocumentsTable)
	return err
}

// CreateDocument creates a new document
func CreateDocument(name, language, username string) (*Document, error) {
	doc := &Document{
		ID:        uuid.New().String(),
		Name:      name,
		Content:   "",
		Language:  language,
		CreatedBy: username,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	query := `
		INSERT INTO documents (id, name, content, language, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(query, doc.ID, doc.Name, doc.Content, doc.Language, doc.CreatedBy, doc.CreatedAt, doc.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return doc, nil
}

// GetDocument retrieves a document by ID
func GetDocument(docID string) (*Document, error) {
	var doc Document

	query := `
		SELECT id, name, content, language, created_by, created_at, updated_at
		FROM documents
		WHERE id = ?
	`

	err := db.QueryRow(query, docID).Scan(
		&doc.ID,
		&doc.Name,
		&doc.Content,
		&doc.Language,
		&doc.CreatedBy,
		&doc.CreatedAt,
		&doc.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &doc, nil
}

// GetAllDocuments retrieves all documents
func GetAllDocuments() ([]Document, error) {
	query := `
		SELECT id, name, content, language, created_by, created_at, updated_at
		FROM documents
		ORDER BY updated_at DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var documents []Document
	for rows.Next() {
		var doc Document
		err := rows.Scan(
			&doc.ID,
			&doc.Name,
			&doc.Content,
			&doc.Language,
			&doc.CreatedBy,
			&doc.CreatedAt,
			&doc.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		documents = append(documents, doc)
	}

	return documents, nil
}

// UpdateDocument updates document content
func UpdateDocument(docID, content string) error {
	query := `
		UPDATE documents
		SET content = ?, updated_at = ?
		WHERE id = ?
	`

	_, err := db.Exec(query, content, time.Now(), docID)
	return err
}

// DeleteDocument deletes a document
func DeleteDocument(docID string) error {
	query := `DELETE FROM documents WHERE id = ?`
	_, err := db.Exec(query, docID)
	return err
}
