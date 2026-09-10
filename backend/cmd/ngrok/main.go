package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.ngrok.com/ngrok/v2"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	if os.Getenv("NGROK_AUTHTOKEN") == "" {
		return fmt.Errorf("NGROK_AUTHTOKEN wajib diisi")
	}
	publicURL := os.Getenv("NGROK_URL")
	u, err := url.Parse(publicURL)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("NGROK_URL harus berupa https://domain tanpa path atau trailing slash")
	}
	fwd, err := ngrok.Forward(ctx, ngrok.WithUpstream("http://nginx:8080"), ngrok.WithURL(publicURL))
	if err != nil {
		return fmt.Errorf("membuka tunnel ngrok: %w", err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := fwd.CloseWithContext(closeCtx); err != nil {
			log.Printf("menutup tunnel: %v", err)
		}
	}()
	log.Printf("Aplikasi tersedia di %s", fwd.URL())
	select {
	case <-ctx.Done():
		return nil
	case <-fwd.Done():
		return fmt.Errorf("tunnel ngrok berhenti")
	}
}
