package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/crypto/bcrypt"

	"my-ple-backend/db"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if len(os.Args) > 1 {
		dbURL = os.Args[1]
	}
	if dbURL == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL is required. Example: go -C backend run ./cmd/seed 'postgres://postgres:PASSWORD@localhost:5432/my_ple?sslmode=disable'")
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

	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	result, err := conn.ExecContext(context.Background(), `
		INSERT INTO users (id, name, email, role_id, username, password_hash)
		SELECT $1, 'Administrator', 'elramaferdian@gmail.com', id, 'admin', $2
		FROM roles WHERE LOWER(name) = 'admin'
		ON CONFLICT (username) DO UPDATE SET
			email = EXCLUDED.email, role_id = EXCLUDED.role_id, password_hash = EXCLUDED.password_hash, updated_at = NOW()`,
		db.NewID(), string(hash))
	if err != nil {
		panic(err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		panic("Admin role not found; run the migration first")
	}
	fmt.Println("Admin account seeded.")
}
