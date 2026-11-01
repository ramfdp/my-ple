package main

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"my-ple-backend/controller"
	"my-ple-backend/db"
	"os"
	"os/signal"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type options struct {
	apply       bool
	count       int
	fingerprint string
}

func parseOptions(args []string) (options, error) {
	var option options
	flags := flag.NewFlagSet("weekly-recovery", flag.ContinueOnError)
	flags.BoolVar(&option.apply, "apply", false, "Apply the reviewed recovery; default is read-only preview")
	flags.IntVar(&option.count, "expected-count", -1, "Exact count from preview")
	flags.StringVar(&option.fingerprint, "fingerprint", "", "SHA-256 fingerprint from preview")
	if err := flags.Parse(args); err != nil {
		return option, err
	}
	if flags.NArg() != 0 {
		return option, errors.New("unexpected positional arguments")
	}
	if !option.apply && (option.count != -1 || option.fingerprint != "") {
		return option, errors.New("review arguments require -apply")
	}
	if option.apply {
		digest, err := hex.DecodeString(option.fingerprint)
		if option.count <= 0 || err != nil || len(digest) != 32 {
			return option, errors.New("-apply requires -expected-count and a 64-character SHA-256 -fingerprint from preview")
		}
	}
	return option, nil
}

func run(ctx context.Context, args []string, output io.Writer) error {
	option, err := parseOptions(args)
	if errors.Is(err, flag.ErrHelp) {
		return nil
	}
	if err != nil {
		return err
	}
	dsn := os.Getenv("DATABASE_URL")
	if strings.TrimSpace(dsn) == "" {
		return errors.New("DATABASE_URL is required")
	}
	connection, err := sql.Open("pgx", dsn)
	if err != nil {
		return errors.New("cannot configure PostgreSQL connection")
	}
	defer connection.Close()
	if err = connection.PingContext(ctx); err != nil {
		return errors.New("cannot connect to PostgreSQL; check DATABASE_URL and network")
	}
	database := db.NewDatabase(connection)
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	if !option.apply {
		preview, err := database.PreviewWeeklyRecovery(ctx)
		if err != nil {
			return err
		}
		return encoder.Encode(preview)
	}
	results, recoveryErr := controller.NewWeeklyLifecycle(database).RunRecovery(ctx, option.count, option.fingerprint, time.Now())
	response := struct {
		Archives []controller.WeeklyRecoveryResult `json:"archives"`
		Error    string                            `json:"error,omitempty"`
	}{Archives: results}
	if recoveryErr != nil {
		response.Error = recoveryErr.Error()
	}
	if err = encoder.Encode(response); err != nil {
		return err
	}
	return recoveryErr
}
