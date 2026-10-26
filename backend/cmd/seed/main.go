package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"unicode/utf8"

	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/crypto/bcrypt"

	"my-ple-backend/db"
)

func databaseURL() (string, error) {
	value := os.Getenv("DATABASE_URL")
	if value == "" {
		return "", errors.New("DATABASE_URL is required")
	}
	return value, nil
}

func initialAdminPassword() (string, error) {
	password := os.Getenv("INITIAL_ADMIN_PASSWORD")
	if utf8.RuneCountInString(password) < 12 {
		return "", errors.New("INITIAL_ADMIN_PASSWORD must contain at least 12 characters")
	}
	return password, nil
}

func main() {
	dbURL, err := databaseURL()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	conn, err := sql.Open("pgx", dbURL)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	if err := conn.Ping(); err != nil {
		fmt.Fprintln(os.Stderr, "Database connection failed:", err)
		os.Exit(1)
	}
	var adminRoleID int
	if err := conn.QueryRowContext(context.Background(), `SELECT id FROM roles WHERE LOWER(name) = 'admin'`).Scan(&adminRoleID); err != nil {
		panic("Admin role not found; run the migration first")
	}

	password, err := initialAdminPassword()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	result, err := conn.ExecContext(context.Background(), `
		INSERT INTO users (id, name, email, role_id, username, password_hash)
		VALUES ($1, 'Administrator', 'elramaferdian@gmail.com', $2, 'admin', $3)
		ON CONFLICT (username) DO NOTHING`,
		db.NewID(), adminRoleID, string(hash))
	if err != nil {
		panic(err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		fmt.Println("Admin account already exists; left unchanged.")
		return
	}
	fmt.Println("Admin account seeded.")
}
