package sql

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/frantjc/daggerverse/dogger/internal/dogger/internal/storage"
)

// Storage is a SQL implementation of the storage.Storage interface
type Storage struct {
	db *sql.DB
}

// New creates a new SQL storage instance and ensures tables are created
func New(db *sql.DB) (*Storage, error) {
	s := &Storage{db: db}

	if err := s.createTables(); err != nil {
		return nil, err
	}

	return s, nil
}

// createTables creates the necessary database tables
func (s *Storage) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS containers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			data TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS images (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			data TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS container_names (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			container_id INTEGER NOT NULL,
			name TEXT NOT NULL UNIQUE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (container_id) REFERENCES containers(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS image_tags (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			image_id INTEGER NOT NULL,
			tag TEXT NOT NULL UNIQUE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (image_id) REFERENCES images(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS container_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			container_id INTEGER NOT NULL,
			stream INTEGER NOT NULL,
			line TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (container_id) REFERENCES containers(id) ON DELETE CASCADE
		)`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return err
		}
	}

	return nil
}

// CreateContainer creates a container in the database
func (s *Storage) CreateContainer(ctx context.Context, id string, container *storage.Container) error {
	data, err := json.Marshal(container)
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert the container and get the internal ID
	result, err := tx.ExecContext(ctx, `INSERT INTO containers (data) VALUES (?)`, data)
	if err != nil {
		return err
	}

	internalID, err := result.LastInsertId()
	if err != nil {
		return err
	}

	// Insert the external "id" as a name
	_, err = tx.ExecContext(ctx, `INSERT INTO container_names (container_id, name) VALUES (?, ?)`, internalID, id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// NameContainer assigns a name to a container
func (s *Storage) NameContainer(ctx context.Context, identifier, name string) error {
	// Find the internal container ID by the identifier (which is also a name)
	query := `INSERT INTO container_names (container_id, name) 
			  SELECT container_id, ? FROM container_names WHERE name = ? LIMIT 1`
	_, err := s.db.ExecContext(ctx, query, name, identifier)
	return err
}

// CreateImage creates an image in the database
func (s *Storage) CreateImage(ctx context.Context, id string, image *storage.Image) error {
	data, err := json.Marshal(image)
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert the image and get the internal ID
	result, err := tx.ExecContext(ctx, `INSERT INTO images (data) VALUES (?)`, data)
	if err != nil {
		return err
	}

	internalID, err := result.LastInsertId()
	if err != nil {
		return err
	}

	// Insert the external "id" as a tag
	_, err = tx.ExecContext(ctx, `INSERT INTO image_tags (image_id, tag) VALUES (?, ?)`, internalID, id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// TagImage assigns a tag to an image
func (s *Storage) TagImage(ctx context.Context, identifier, tag string) error {
	// Find the internal image ID by the identifier (which is also a tag)
	query := `INSERT INTO image_tags (image_id, tag) 
			  SELECT image_id, ? FROM image_tags WHERE tag = ? LIMIT 1`
	_, err := s.db.ExecContext(ctx, query, tag, identifier)
	return err
}

// GetContainer retrieves a container by name
func (s *Storage) GetContainer(ctx context.Context, identifier string) (*storage.Container, error) {
	var data []byte

	// Find container by name (identifier is always a name now)
	query := `SELECT c.data FROM containers c 
			  JOIN container_names cn ON c.id = cn.container_id 
			  WHERE cn.name = ? LIMIT 1`
	err := s.db.QueryRowContext(ctx, query, identifier).Scan(&data)
	if err != nil {
		return nil, err
	}

	var container storage.Container
	if err := json.Unmarshal(data, &container); err != nil {
		return nil, err
	}

	return &container, nil
}

// UpdateContainer updates a container's data by name
func (s *Storage) UpdateContainer(ctx context.Context, identifier string, container *storage.Container) error {
	data, err := json.Marshal(container)
	if err != nil {
		return err
	}

	// Update container by finding it through the name (identifier is always a name now)
	query := `UPDATE containers SET data = ? WHERE id IN (
			    SELECT container_id FROM container_names WHERE name = ? LIMIT 1
			  )`
	result, err := s.db.ExecContext(ctx, query, data, identifier)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// DeleteContainer deletes a container by name
func (s *Storage) DeleteContainer(ctx context.Context, identifier string) error {
	// Delete container by finding it through the name (identifier is always a name now)
	query := `DELETE FROM containers WHERE id IN (
			    SELECT container_id FROM container_names WHERE name = ?
			  )`
	_, err := s.db.ExecContext(ctx, query, identifier)
	return err
}

// ListContainers retrieves all containers
func (s *Storage) ListContainers(ctx context.Context) ([]storage.Container, error) {
	query := `SELECT DISTINCT c.data FROM containers c`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var containers []storage.Container
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}

		var container storage.Container
		if err := json.Unmarshal(data, &container); err != nil {
			return nil, err
		}

		containers = append(containers, container)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return containers, nil
}

// GetImage retrieves an image by tag
func (s *Storage) GetImage(ctx context.Context, identifier string) (*storage.Image, error) {
	var data []byte

	// Find image by tag (identifier is always a tag now)
	query := `SELECT i.data FROM images i 
			  JOIN image_tags it ON i.id = it.image_id 
			  WHERE it.tag = ? LIMIT 1`
	err := s.db.QueryRowContext(ctx, query, identifier).Scan(&data)
	if err != nil {
		return nil, err
	}

	var image storage.Image
	if err := json.Unmarshal(data, &image); err != nil {
		return nil, err
	}

	return &image, nil
}

// DeleteImage deletes an image by tag
func (s *Storage) DeleteImage(ctx context.Context, identifier string) error {
	// Delete image by finding it through the tag (identifier is always a tag now)
	query := `DELETE FROM images WHERE id IN (
			    SELECT image_id FROM image_tags WHERE tag = ?
			  )`
	_, err := s.db.ExecContext(ctx, query, identifier)
	return err
}
