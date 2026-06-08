package db

import (
	"database/sql"
	"log"
)

// EnsureAdmin creates or updates the admin user. Called at startup when
// AdminUsername and AdminPassword are set in config.
func EnsureAdmin(db *sql.DB, username, passwordHash string) error {
	var id int64
	err := db.QueryRow("SELECT id FROM users WHERE username = ? COLLATE NOCASE", username).Scan(&id)
	if err == sql.ErrNoRows {
		// Create admin user.
		_, err = db.Exec(
			"INSERT INTO users (username, password_hash, is_admin) VALUES (?, ?, 1)",
			username, passwordHash,
		)
		if err != nil {
			return err
		}
		log.Printf("setup: admin user %q created", username)
		return nil
	}
	if err != nil {
		return err
	}
	// Update existing user to admin + update password hash.
	_, err = db.Exec(
		"UPDATE users SET password_hash=?, is_admin=1 WHERE id=?",
		passwordHash, id,
	)
	log.Printf("setup: admin user %q updated", username)
	return err
}
