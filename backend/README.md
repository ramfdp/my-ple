# my-ple backend (Go)

```sh
go mod download
copy .env.example .env
# Buat database PostgreSQL `my_ple`, lalu jalankan db/schema.sql memakai client PostgreSQL.
go run .
```

Endpoint publik: `GET /health` dan `POST /auth/login`. Endpoint lain memerlukan cookie sesi HttpOnly yang dibuat saat login.

`POST /reports` menetapkan nomor laporan di database. Nomor diurutkan terpisah untuk setiap kombinasi divisi utama dan divisi pekerjaan; request yang lebih dulu tersimpan mendapat nomor lebih kecil.

Selain health, endpoint memerlukan `DATABASE_URL`. Durasi sesi diatur dengan `SESSION_TTL` (default `8h`). Gunakan `APP_ENV=production` atau `COOKIE_SECURE=true` pada HTTPS; localhost HTTP tetap memakai cookie non-Secure.
