package db

import (
	"database/sql"
	"time"
)

// SessionStore implements scs.Store using the existing SQLite db.
type SessionStore struct {
	db *sql.DB
}

func NewSessionStore(db *sql.DB) *SessionStore {
	return &SessionStore{db: db}
}

func (s *SessionStore) Find(token string) ([]byte, bool, error) {
	var data []byte
	var expiry float64
	err := s.db.QueryRow(
		"SELECT data, expiry FROM sessions WHERE token = ?", token,
	).Scan(&data, &expiry)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if float64(time.Now().Unix()) > expiry {
		return nil, false, nil
	}
	return data, true, nil
}

func (s *SessionStore) Commit(token string, b []byte, expiry time.Time) error {
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO sessions (token, data, expiry) VALUES (?, ?, ?)",
		token, b, float64(expiry.Unix()),
	)
	return err
}

func (s *SessionStore) Delete(token string) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}

func (s *SessionStore) DeleteExpired() error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE expiry < ?", float64(time.Now().Unix()))
	return err
}
