# my-ple backend (Go)

```sh
go mod download
copy .env.example .env
# Buat database PostgreSQL `my_ple`, lalu jalankan db/schema.sql memakai client PostgreSQL.
go run .
```

Endpoint awal: `GET /health`, `POST /auth/register`, `POST /auth/login`, `GET /me`, dan `GET /dashboard`.

Selain health, endpoint memerlukan `DATABASE_URL`. Endpoint beridentitas memakai `Authorization: Bearer <token>`.
