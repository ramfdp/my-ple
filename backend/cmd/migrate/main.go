package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"my-ple-backend/db"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if len(os.Args) > 1 && os.Args[1] != "" {
		dbURL = os.Args[1]
	}
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/my_ple"
	}

	fmt.Println("Connecting to configured database...")
	conn, err := sql.Open("pgx", dbURL)
	if err != nil {
		fmt.Printf("Connection error: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	if err := conn.Ping(); err != nil {
		fmt.Printf("Database ping failed: %v\n", err)
		fmt.Println("\nTip: Pastikan PostgreSQL aktif dan DATABASE_URL sesuai dengan password pgAdmin Anda.")
		fmt.Println("Contoh: go run cmd/migrate/main.go \"postgres://postgres:PASSWORD_ANDA@localhost:5432/my_ple\"")
		os.Exit(1)
	}

	database := db.NewDatabase(conn)
	fmt.Println("Running schema migration...")
	if err := database.Migrate(context.Background()); err != nil {
		fmt.Printf("Migration failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("SUCCESS: Migration completed successfully!")
	fmt.Println("\nTables created/verified:")
	rows, err := conn.Query("SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' ORDER BY table_name;")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err == nil {
				fmt.Printf("  - %s\n", name)
			}
		}
	}
}
