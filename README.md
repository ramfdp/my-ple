# My PLE — Docker

Jalankan dari root proyek dengan Docker Compose v2.24 atau lebih baru:

```sh
docker compose up -d --build
docker compose ps
```

Frontend: http://localhost:5173. Backend: http://localhost:3000
(health check: `/health`). Compose menjalankan PostgreSQL baru; skema dibuat
otomatis oleh backend. Data database lokal yang sudah ada tidak diimpor otomatis.

Nginx menerima trafik backend pada port `3000` dan membaginya secara round-robin.
Frontend tetap satu container pada port `5173`. Untuk menjalankan tiga backend:

```sh
docker compose up -d --build --scale backend=3
```

Ubah jumlahnya kapan saja, misalnya menjadi lima atau kembali satu:

```sh
docker compose up -d --scale backend=5
docker compose up -d --scale backend=1
docker compose ps backend
docker compose exec nginx nginx -t
```

Ganti angka sesuai kebutuhan. Tidak perlu mengedit konfigurasi atau reload Nginx:
[upstream resolve](https://nginx.org/en/docs/http/ngx_http_upstream_module.html#server)
memperbarui alamat backend melalui DNS Docker dengan cache 5 detik.
Semua replika berbagi database, upload, dan arsip laporan yang sama.
Upload diteruskan ke backend; [WebSocket](https://nginx.org/en/docs/http/websocket.html)
juga diteruskan melalui Nginx. Koneksi WebSocket pada replika yang dihentikan
akan tersambung ulang melalui mekanisme reconnect frontend.

Batasan: hub event WebSocket aplikasi masih berada di memori masing-masing
backend. Saat memakai beberapa replika, notifikasi realtime dan invalidasi
workspace/cuti belum tersebar antar-replika; data tetap dapat dibaca melalui API
atau refresh halaman. Sinkronisasi event lintas replika memerlukan pub/sub bersama.

`backend/.env` opsional untuk pengaturan aplikasi/SMTP. Compose mengganti
`DATABASE_URL`, port, origin, dan direktori penyimpanan agar sesuai container.
Konfigurasi default menggunakan HTTP lokal; pastikan `APP_ENV` bukan `production`
dan `COOKIE_SECURE` bukan `true` untuk login melalui HTTP.

Untuk akses dari komputer lain, buat `.env` di root (jangan commit):

```dotenv
FRONTEND_ORIGIN=http://192.168.1.10:5173
VITE_API_URL=http://192.168.1.10:3000
POSTGRES_PASSWORD=ganti-password-lokal
```

Ganti IP sesuai host Docker, lalu jalankan ulang `docker compose up -d --build`.
`VITE_API_URL` digunakan saat build dan harus bisa dijangkau browser.
Password bawaan hanya untuk penggunaan lokal. Untuk password kustom, gunakan
karakter tanpa spasi, kutip tunggal, atau backslash karena dipakai dalam DSN PostgreSQL.
Mengubah password di `.env` tidak mengganti password database yang sudah terbentuk.
Untuk HTTPS, sesuaikan kedua URL dan aktifkan `COOKIE_SECURE=true` di `backend/.env`.

Database, upload, dan arsip laporan disimpan dalam named volume. File lama di
`backend/upload` dan `backend/storage` tidak disalin otomatis ke volume baru.

```sh
docker compose logs -f
docker compose down
```

`down` mempertahankan data; `down -v` menghapus seluruh volume beserta datanya.

Frontend memakai [adapter Node SvelteKit](https://svelte.dev/docs/kit/adapter-node).
Urutan startup mengikuti [health check Compose](https://docs.docker.com/compose/how-tos/startup-order/).

## ngrok dengan SDK Go (Docker di WSL)

### Impor backup Custom PostgreSQL 18

Compose memakai PostgreSQL 18 dan volume baru `postgres18_data`, dengan mount
`/var/lib/postgresql` sesuai image PostgreSQL 18. Volume `postgres_data` versi 17
yang sudah ada tetap tersimpan, tetapi tidak digunakan oleh konfigurasi ini.
Backup versi 18 tidak dijamin kompatibel dengan server versi 17.

Untuk restore pertama ke database Docker baru, jalankan dari root proyek di WSL.
Hentikan aplikasi terlebih dahulu agar migrasi backend tidak membuat tabel sebelum
restore. Perintah restore menggunakan satu transaksi; jika gagal, jangan lanjutkan
ke startup aplikasi. Tidak perlu menjalankan seed akun.

```sh
docker compose -f compose.yaml -f compose.ngrok.yaml stop ngrok frontend nginx backend
docker compose -f compose.yaml -f compose.ngrok.yaml up -d --wait db
docker compose -f compose.yaml -f compose.ngrok.yaml exec -T db pg_restore --username=my_ple --dbname=my_ple --no-owner --no-privileges --single-transaction --exit-on-error < /mnt/c/Users/LAPI-PMO5/Downloads/DB_My_HIS.sql
```

Perintah ini untuk database tujuan kosong, tanpa menghapus tabel yang sudah ada.
Jika berhasil, verifikasi akun kemudian jalankan aplikasi:

```sh
docker compose -f compose.yaml -f compose.ngrok.yaml exec db psql -U my_ple -d my_ple -c "SELECT username, is_active FROM users;"
docker compose -f compose.yaml -f compose.ngrok.yaml up -d --build
```

Restore membawa skema dan data yang ada dalam backup, termasuk hash password.
File upload dan arsip PDF di filesystem harus disalin terpisah ke volume aplikasi.

### Menjalankan tunnel

Service `ngrok` menjalankan `backend/cmd/ngrok` menggunakan
[SDK Go ngrok v2](https://pkg.go.dev/golang.ngrok.com/ngrok/v2).
Satu URL HTTPS melayani frontend, API `/api/`, upload `/api/uploads/`, dan
WebSocket `/api/ws/notifications` melalui Nginx. Tidak perlu memasang CLI ngrok.

Di WSL, pastikan `docker compose version` berhasil (aktifkan integrasi distro WSL
di Docker Desktop jika memakai Docker Desktop). Dari root proyek, tambahkan ke
file `.env` yang sudah ada atau buat jika belum ada:

```dotenv
NGROK_AUTHTOKEN=isi-authtoken-anda
NGROK_URL=https://domain-anda.ngrok-free.app
COMPOSE_PATH_SEPARATOR=;
COMPOSE_FILE=compose.yaml;compose.ngrok.yaml
```

Ambil token dari [dashboard ngrok](https://dashboard.ngrok.com/get-started/your-authtoken)
dan gunakan domain milik akun Anda dari [Domains](https://dashboard.ngrok.com/domains).
Isi URL lengkap dengan `https://`, tanpa slash di akhir, path, atau query.
Token dibaca saat container berjalan, tidak dimasukkan ke image; `.env` diabaikan Git.
`COMPOSE_FILE` membuat perintah `docker compose` tanpa `-f` memakai mode ngrok
secara otomatis, termasuk saat rebuild. Pemisah eksplisit berlaku di Windows dan WSL.

```sh
docker compose -f compose.yaml -f compose.ngrok.yaml config --quiet
docker compose -f compose.yaml -f compose.ngrok.yaml up -d --build
docker compose -f compose.yaml -f compose.ngrok.yaml logs -f ngrok
```

Buka `NGROK_URL` di browser. Mode ini mengatur origin frontend/backend dan cookie
Secure otomatis; gunakan URL HTTPS tersebut untuk login. Frontend dibangun dengan
API relatif `/api`, sehingga browser tidak mengakses localhost milik pengunjung.
Backend tetap dapat diskalakan dengan menambahkan `--scale backend=3` pada perintah
`up`; batasan realtime antar-replika di atas tetap berlaku.

Verifikasi setelah startup: buka halaman login, login, unggah avatar, dan periksa
koneksi WebSocket di browser. Health API dapat diperiksa dari WSL:

```sh
curl -fsS -H 'ngrok-skip-browser-warning: 1' https://domain-anda.ngrok-free.app/api/health
```

Jika tunnel gagal, periksa log service `ngrok` untuk kesalahan token/domain.
Untuk menghentikan mode ini tanpa menghapus data:

```sh
docker compose -f compose.yaml -f compose.ngrok.yaml down
```

Untuk kembali ke mode HTTP lokal, jalankan perintah `down` di atas lalu
`docker compose -f compose.yaml up -d --build`; pastikan pengaturan `backend/.env` sesuai HTTP
seperti dijelaskan di atas.
