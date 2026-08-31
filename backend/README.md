# my-ple backend (Go)

```sh
go mod download
copy .env.example .env
# Buat database PostgreSQL `my_ple`, lalu jalankan db/schema.sql memakai client PostgreSQL.
go run .
```

Endpoint: `GET /health`, `POST /auth/register`, `POST /auth/login`, `GET /me`, `GET /dashboard`, `GET /reports?primaryDivision=...`, dan `POST /reports`.

`POST /reports` menetapkan nomor laporan di database. Nomor diurutkan terpisah untuk setiap kombinasi divisi utama dan divisi pekerjaan; request yang lebih dulu tersimpan mendapat nomor lebih kecil.

Selain health, endpoint memerlukan `DATABASE_URL`. Endpoint beridentitas memakai `Authorization: Bearer <token>`.
