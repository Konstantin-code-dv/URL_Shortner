package sqlite

import (
	"URL_Shortner/internal/storage"
	"database/sql"
	"errors"
	"fmt"

	"github.com/mattn/go-sqlite3"
)

type Storage struct {
	db *sql.DB
}

func NewStorage(storagePath string) (*Storage, error) {
	const op = "storage.sqlite.New"

	db, err := sql.Open("sqlite3", storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	// Проверка подключения к БД
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("%s: ping database: %w", op, err)
	}

	stmt, err := db.Prepare(`
	CREATE TABLE IF NOT EXISTS url(
		id INTEGER PRIMARY KEY,
		alias TEXT NOT NULL UNIQUE,
		url TEXT NOT NULL);
	CREATE INDEX IF NOT EXISTS idx_alias ON url(alias);
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.Exec()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

// Сохранение ссылки
func (s *Storage) SaveURL(urlToSave, alias string) (int64, error) {
	const op = "storage.sqlite.SaveURL"

	res, err := s.db.Exec(
		"INSERT INTO url(url, alias) VALUES (?, ?)",
		urlToSave, alias,
	)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) &&
			sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrURLExists)
		}

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

// Получение ссылки
func (s *Storage) GetURL(alias string) (string, error) {
	const op = "storage.sqlite.GetURL"
	query := `SELECT url FROM url WHERE alias = ?`

	var result string

	err := s.db.QueryRow(query, alias).Scan(&result)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", storage.ErrURLNotFound
		}
		return "", fmt.Errorf("%s: %w", op, err)
	}
	return result, nil
}

// Удаление ссылки
func (s *Storage) DeleteURL(alias string) error {
	const op = "storage.sqlite.DeleteURL"
	query := `DELETE FROM url WHERE alias = ?`
	_, err := s.db.Exec(query, alias)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ErrURLNotFound
		}
		return fmt.Errorf("%s: %w", op, err)
	}
	fmt.Println("Deleted URL", alias)
	return nil
}
