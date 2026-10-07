# My PLE — Docker

Jalankan dari root proyek dengan Docker Compose v2.24 atau lebih baru:

```sh
docker compose up -d --build
docker compose ps
```

Frontend: http://localhost:5173. Backend: http://localhost:3000
(health check: `/health`). Compose menjalankan PostgreSQL baru; skema dibuat
otomatis oleh backend. Data database lokal yang sudah ada tidak diimpor otomatis.

## Testing dengan domain ngrok

Environment testing dijalankan dari VPS pada direktori:

```sh
cd /opt/my-ple-testing
```

Testing menggunakan konfigurasi berikut:

- branch Git: `testing`
- environment file: `.env.testing`
- Compose utama: `compose.yaml`
- Compose tambahan testing: `compose.testing.yaml`
- frontend lokal: `127.0.0.1:5174`
- backend/Nginx lokal: `127.0.0.1:3001`
- gateway testing: `127.0.0.1:8081`
- domain testing: `https://unrecessive-fibular-alyce.ngrok-free.dev`

Project `my-ple-testing` mempunyai database PostgreSQL, upload, dan arsip laporan
sendiri. Email testing dinonaktifkan.

> **Penting:** seluruh operasi Docker Compose untuk environment testing harus
> menggunakan `.env.testing`, `compose.yaml`, dan `compose.testing.yaml`.
> Jangan menggunakan `docker compose up`, `docker compose down`, atau command
> Compose lain tanpa konfigurasi tersebut karena service `gateway` berada di
> `compose.testing.yaml`.

### Konfigurasi environment testing

Salin `.env.testing.example` menjadi `.env.testing` jika file belum tersedia:

```sh
cp .env.testing.example .env.testing
```

Isi `NGROK_DOMAIN` dengan hostname dari dashboard ngrok tanpa `https://` dan tanpa
trailing slash. Gunakan password PostgreSQL khusus testing.

Contoh:

```dotenv
COMPOSE_PROJECT_NAME=my-ple-testing
BACKEND_ENV_FILE=./.env.testing

BACKEND_HOST_PORT=3001
FRONTEND_HOST_PORT=5174

POSTGRES_PASSWORD=ganti-dengan-password-testing

NGROK_DOMAIN=unrecessive-fibular-alyce.ngrok-free.dev
FRONTEND_ORIGIN=https://${NGROK_DOMAIN:?Isi NGROK_DOMAIN di .env.testing}
VITE_API_URL=${FRONTEND_ORIGIN}/api

APP_ENV=testing
COOKIE_SECURE=true
SESSION_TTL=8h

EMAIL_ENABLED=false
ZOOM_EMAIL_ENABLED=false
```

File `.env.testing` tidak masuk Git.

`POSTGRES_PASSWORD` hanya digunakan untuk menginisialisasi password saat volume
PostgreSQL pertama kali dibuat. Mengubah nilainya pada `.env.testing` tidak otomatis
mengubah password role PostgreSQL yang sudah tersimpan pada volume lama.

### Update source code testing dari Git

Sebelum deployment, pastikan VPS berada pada branch `testing` dan working tree bersih:

```sh
cd /opt/my-ple-testing

git checkout testing
git status
git pull --ff-only origin testing
```

`git pull --ff-only` digunakan agar VPS tidak membuat merge commit deployment secara
tidak sengaja. Perubahan kode sebaiknya dibuat dan di-commit dari development machine,
kemudian di-push ke branch `testing`.

### Validasi konfigurasi Docker Compose

Sebelum menjalankan stack:

```sh
docker compose \
  --env-file .env.testing \
  -f compose.yaml \
  -f compose.testing.yaml \
  config --quiet
```

Jika command selesai tanpa error, konfigurasi Compose valid.

Untuk melihat konfigurasi hasil gabungan:

```sh
docker compose \
  --env-file .env.testing \
  -f compose.yaml \
  -f compose.testing.yaml \
  config
```

### Build dan menjalankan testing

Untuk build image terbaru dan menjalankan seluruh stack:

```sh
docker compose \
  --env-file .env.testing \
  -f compose.yaml \
  -f compose.testing.yaml \
  up -d --build --wait
```

Service yang harus berjalan:

```text
db
backend
nginx
frontend
gateway
```

Periksa status:

```sh
docker compose \
  --env-file .env.testing \
  -f compose.yaml \
  -f compose.testing.yaml \
  ps
```

Kondisi normal:

```text
db         healthy
backend    healthy
nginx      healthy
frontend   running
gateway    running
```

Gateway harus memiliki mapping:

```text
127.0.0.1:8081->80/tcp
```

### Build ulang tanpa menjalankan container

Untuk build seluruh image:

```sh
docker compose \
  --env-file .env.testing \
  -f compose.yaml \
  -f compose.testing.yaml \
  build
```

Kemudian jalankan:

```sh
docker compose \
  --env-file .env.testing \
  -f compose.yaml \
  -f compose.testing.yaml \
  up -d --wait
```

Jika perlu memastikan Docker tidak memakai build cache:

```sh
docker compose \
  --env-file .env.testing \
  -f compose.yaml \
  -f compose.testing.yaml \
  build --no-cache
```

### Build backend saja

```sh
docker compose \
  --env-file .env.testing \
  -f compose.yaml \
  -f compose.testing.yaml \
  build backend
```

Kemudian:

```sh
docker compose \
  --env-file .env.testing \
  -f compose.yaml \
  -f compose.testing.yaml \
  up -d backend
```

### Build frontend saja

Untuk memperbarui frontend pada stack testing yang sudah berjalan, build ulang
dan ganti container frontend dalam satu perintah:

```sh
docker compose \
  --env-file .env.testing \
  -f compose.yaml \
  -f compose.testing.yaml \
  up -d --build --no-deps --force-recreate --wait frontend
```

Karena `VITE_API_URL` dimasukkan ke bundle frontend saat proses build, frontend harus
dibuild ulang setelah `NGROK_DOMAIN`, `FRONTEND_ORIGIN`, atau `VITE_API_URL` berubah.

Untuk memaksa build frontend dari nol:

```sh
docker compose \
  --env-file .env.testing \
  -f compose.yaml \
  -f compose.testing.yaml \
  build --no-cache frontend

docker compose \
  --env-file .env.testing \
  -f compose.yaml \
  -f compose.testing.yaml \
  up -d --no-deps --force-recreate frontend
```

### Verifikasi frontend dan API testing

Sebelum menjalankan ngrok, pastikan gateway dapat diakses langsung dari VPS:

```sh
curl -I --max-time 10 http://127.0.0.1:8081/
```

Respons normal:

```text
HTTP/1.1 200 OK
```

Periksa backend melalui gateway:

```sh
curl --fail --show-error --max-time 10 \
  http://127.0.0.1:8081/api/health
```

Respons normal:

```json
{"database":"configured","status":"ok"}
```

Gateway meneruskan:

```text
/api/*  -> nginx -> backend
/*      -> frontend
```

Dengan demikian browser hanya perlu mengakses domain HTTPS ngrok.

### Log Docker testing

Semua log:

```sh
docker compose \
  --env-file .env.testing \
  -f compose.yaml \
  -f compose.testing.yaml \
  logs -f
```

100 log backend terakhir:

```sh
docker compose \
  --env-file .env.testing \
  -f compose.yaml \
  -f compose.testing.yaml \
  logs --tail=100 backend
```

Log backend secara realtime:

```sh
docker compose \
  --env-file .env.testing \
  -f compose.yaml \
  -f compose.testing.yaml \
  logs -f backend
```

Log frontend:

```sh
docker compose \
  --env-file .env.testing \
  -f compose.yaml \
  -f compose.testing.yaml \
  logs -f frontend
```

Log gateway:

```sh
docker compose \
  --env-file .env.testing \
  -f compose.yaml \
  -f compose.testing.yaml \
  logs -f gateway
```

Log database:

```sh
docker compose \
  --env-file .env.testing \
  -f compose.yaml \
  -f compose.testing.yaml \
  logs -f db
```

Gunakan `Ctrl+C` untuk keluar dari tampilan log. Container tetap berjalan.

### Restart service

Restart backend:

```sh
docker compose \
  --env-file .env.testing \
  -f compose.yaml \
  -f compose.testing.yaml \
  restart backend
```

Restart frontend:

```sh
docker compose \
  --env-file .env.testing \
  -f compose.yaml \
  -f compose.testing.yaml \
  restart frontend
```

Restart gateway:

```sh
docker compose \
  --env-file .env.testing \
  -f compose.yaml \
  -f compose.testing.yaml \
  restart gateway
```

### Stop testing

Untuk menghentikan container tanpa menghapusnya:

```sh
docker compose \
  --env-file .env.testing \
  -f compose.yaml \
  -f compose.testing.yaml \
  stop
```

Untuk menjalankannya kembali:

```sh
docker compose \
  --env-file .env.testing \
  -f compose.yaml \
  -f compose.testing.yaml \
  start
```

### Mematikan dan menghapus container testing

Gunakan:

```sh
docker compose \
  --env-file .env.testing \
  -f compose.yaml \
  -f compose.testing.yaml \
  down
```

Command `down` mempertahankan named volume database, upload, dan storage.

Jangan gunakan untuk deployment normal:

```sh
docker compose down -v
```

Opsi `-v` menghapus named volume beserta data testing di dalamnya.

Jangan pula menjalankan hanya:

```sh
docker compose down
```

karena command tersebut tidak membaca `compose.testing.yaml` sehingga container
`gateway` dapat tertinggal dan membuat network Docker tetap digunakan.

### Masuk ke PostgreSQL testing

```sh
docker compose \
  --env-file .env.testing \
  -f compose.yaml \
  -f compose.testing.yaml \
  exec db \
  psql -U my_ple -d my_ple
```

Keluar dari PostgreSQL:

```text
\q
```

### Instalasi ngrok

Docker Compose hanya menjalankan aplikasi. Tunnel ngrok dijalankan sebagai proses
terpisah pada VPS.

Jika `ngrok: command not found`, instal sekali pada Ubuntu/Debian:

```sh
curl -fsSL https://ngrok-agent.s3.amazonaws.com/ngrok.asc \
  | sudo tee /etc/apt/trusted.gpg.d/ngrok.asc >/dev/null

echo "deb https://ngrok-agent.s3.amazonaws.com bookworm main" \
  | sudo tee /etc/apt/sources.list.d/ngrok.list

sudo apt update
sudo apt install -y ngrok
```

Tambahkan authtoken dari dashboard ngrok:

```sh
ngrok config add-authtoken TOKEN_ANDA
```

### Menjalankan tunnel testing

Pastikan terlebih dahulu:

```sh
curl -I http://127.0.0.1:8081/
curl http://127.0.0.1:8081/api/health
```

Kemudian jalankan:

```sh
ngrok http 127.0.0.1:8081 \
  --url https://unrecessive-fibular-alyce.ngrok-free.dev
```

Gunakan `127.0.0.1:8081`, **bukan** `localhost:8081`.

Pada server tertentu `localhost` dapat diterjemahkan menjadi IPv6 `::1`, sedangkan
gateway testing dipublikasikan hanya pada IPv4 `127.0.0.1:8081`. Kondisi tersebut
menghasilkan error seperti:

```text
ERR_NGROK_8012
dial tcp [::1]:8081: connect: connection refused
```

Jalankan hanya satu proses ngrok untuk domain testing yang sama.

Periksa proses yang sedang berjalan:

```sh
pgrep -a ngrok
```

Untuk menghentikan tunnel lama:

```sh
pkill -f ngrok
```

Kemudian jalankan kembali command ngrok dengan `127.0.0.1:8081`.

Tunnel manual akan berhenti jika terminal ngrok ditutup, `Ctrl+C` ditekan, sesi SSH
terputus dan proses ikut berakhir, atau VPS direboot. Konfigurasi ini belum memasang
ngrok sebagai system service.

### Troubleshooting ERR_NGROK_8012

Jika muncul:

```text
ERR_NGROK_8012
```

periksa service testing:

```sh
docker compose \
  --env-file .env.testing \
  -f compose.yaml \
  -f compose.testing.yaml \
  ps
```

Pastikan `gateway` berjalan dan memiliki port:

```text
127.0.0.1:8081->80/tcp
```

Tes gateway:

```sh
curl -v http://127.0.0.1:8081/
```

Tes API:

```sh
curl -v http://127.0.0.1:8081/api/health
```

Jika keduanya menghasilkan `200 OK`, jalankan ngrok dengan:

```sh
ngrok http 127.0.0.1:8081 \
  --url https://unrecessive-fibular-alyce.ngrok-free.dev
```

Jika `gateway` tidak muncul pada `docker compose ps`, kemungkinan stack dijalankan
tanpa `compose.testing.yaml`. Jalankan kembali:

```sh
docker compose \
  --env-file .env.testing \
  -f compose.yaml \
  -f compose.testing.yaml \
  up -d --wait
```

### Troubleshooting ERR_NGROK_3200

Jika domain menampilkan:

```text
ERR_NGROK_3200
```

periksa apakah aplikasi dan proses ngrok masih aktif:

```sh
curl -I --max-time 10 http://127.0.0.1:8081
pgrep -a ngrok
```

Jika aplikasi merespons tetapi `pgrep` tidak menampilkan proses ngrok, jalankan kembali:

```sh
ngrok http 127.0.0.1:8081 \
  --url https://unrecessive-fibular-alyce.ngrok-free.dev
```

Jika proses ngrok sudah ada, lihat terminal tempat ngrok dijalankan untuk mengetahui
error dan pastikan tidak ada dua agent yang menggunakan domain yang sama.

### Workflow deployment testing

Workflow normal setelah perubahan pada branch `testing`:

```sh
cd /opt/my-ple-testing

git checkout testing
git status
git pull --ff-only origin testing

docker compose \
  --env-file .env.testing \
  -f compose.yaml \
  -f compose.testing.yaml \
  config --quiet

docker compose \
  --env-file .env.testing \
  -f compose.yaml \
  -f compose.testing.yaml \
  up -d --build --wait

docker compose \
  --env-file .env.testing \
  -f compose.yaml \
  -f compose.testing.yaml \
  ps

curl --fail --show-error http://127.0.0.1:8081/api/health
```

Setelah aplikasi sehat, jalankan atau restart tunnel:

```sh
ngrok http 127.0.0.1:8081 \
  --url https://unrecessive-fibular-alyce.ngrok-free.dev
```

Buka domain berikut untuk pengujian:

```text
https://unrecessive-fibular-alyce.ngrok-free.dev
```

Karena login testing memakai secure cookie, pengujian dari browser dilakukan melalui
domain HTTPS ngrok.

### Ringkasan command penting testing

Menjalankan:

```sh
docker compose --env-file .env.testing -f compose.yaml -f compose.testing.yaml up -d --build --wait
```

Status:

```sh
docker compose --env-file .env.testing -f compose.yaml -f compose.testing.yaml ps
```

Log:

```sh
docker compose --env-file .env.testing -f compose.yaml -f compose.testing.yaml logs -f
```

Mematikan:

```sh
docker compose --env-file .env.testing -f compose.yaml -f compose.testing.yaml down
```

Health check:

```sh
curl http://127.0.0.1:8081/api/health
```

Ngrok:

```sh
ngrok http 127.0.0.1:8081 \
  --url https://unrecessive-fibular-alyce.ngrok-free.dev
```

Rujukan:

- [Docker Compose project](https://docs.docker.com/compose/how-tos/project-name/)
- [Docker Compose multiple files](https://docs.docker.com/compose/how-tos/multiple-compose-files/)
- [Docker Compose environment variables](https://docs.docker.com/compose/how-tos/environment-variables/)
- [Instalasi ngrok Linux](https://ngrok.com/download/linux)
- [ngrok CLI](https://ngrok.com/docs/agent/cli)
- [ERR_NGROK_8012](https://ngrok.com/docs/errors/err_ngrok_8012/)
- [ERR_NGROK_3200](https://ngrok.com/docs/errors/err_ngrok_3200/)

## Deployment production (VPS)

Simpan konfigurasi production di `.env.production` pada root proyek, lalu selalu
sertakan file tersebut pada setiap perintah Compose. Jalankan dari checkout
production dengan working tree bersih:

```sh
git switch production
git status --short
git pull --ff-only origin production
git log -1 --oneline
docker compose --env-file .env.production config --quiet
docker compose --env-file .env.production up -d --build --scale backend=3 --wait
docker compose --env-file .env.production ps
```

`git pull` hanya memperbarui source code. Container frontend menjalankan bundle
yang dibuat saat build image; `restart` saja tidak memperbarui bundle tersebut.
Pastikan commit dari `git log` sesuai dengan perubahan yang ingin dideploy.

Jangan menjalankan build production tanpa `--env-file .env.production`.
`VITE_API_URL` dimasukkan ke bundle frontend saat proses build; jika argumen
tersebut tidak terbaca, browser akan mencoba mengakses `localhost:3000`.

Untuk perubahan frontend saja (termasuk CSS/layout), setelah `git pull` jalankan
perintah berikut pada stack production yang sudah berjalan. Backend dan database
tetap berjalan:

```sh
docker compose --env-file .env.production up -d --build --no-deps --force-recreate --wait frontend
docker compose --env-file .env.production ps frontend
docker compose --env-file .env.production logs --tail=50 frontend
```

Perubahan `VITE_API_URL` juga memerlukan build ulang. Jika perlu membangun dari
nol untuk diagnosis, gunakan `--no-cache`; ini tidak diperlukan untuk update normal:

```sh
docker compose --env-file .env.production build --no-cache frontend
docker compose --env-file .env.production up -d --no-deps --force-recreate --wait frontend
```

Verifikasi bundle dan endpoint production:

```sh
docker compose --env-file .env.production exec -T frontend \
  grep -Roh 'https://myhis.dpdns.org/api' /app/build/client
curl -I https://myhis.dpdns.org/
curl -I https://myhis.dpdns.org/api/health
```

Setelah deployment, muat ulang halaman di browser. Jika tampilan masih lama,
periksa file CSS pada tab Network dan pastikan bundle yang dilayani berisi
perubahan terbaru. Peringatan preload CSS yang belum dipakai tidak dengan
sendirinya menandakan build gagal.

Nginx menerima trafik backend pada port `3000` dan membaginya secara round-robin.
Frontend tetap satu container pada port `5173`. Untuk menjalankan tiga backend:

```sh
docker compose --env-file .env.production up -d --build --scale backend=3 --wait
```

Ubah jumlahnya kapan saja, misalnya menjadi lima atau kembali satu:

```sh
docker compose --env-file .env.production up -d --scale backend=5 --wait
docker compose --env-file .env.production up -d --scale backend=1 --wait
docker compose --env-file .env.production ps backend
docker compose --env-file .env.production exec nginx nginx -t
```

Ganti angka sesuai kebutuhan. Tidak perlu mengedit konfigurasi atau reload Nginx:
[upstream resolve](https://nginx.org/en/docs/http/ngx_http_upstream_module.html#server)
memperbarui alamat backend melalui DNS Docker dengan cache 5 detik.
Semua replika berbagi database, upload, dan arsip laporan yang sama.
Upload diteruskan ke backend; [WebSocket](https://nginx.org/en/docs/http/websocket.html)
juga diteruskan melalui Nginx. Koneksi WebSocket pada replika yang dihentikan
akan tersambung ulang melalui mekanisme reconnect frontend.

Setiap sesi browser memakai satu koneksi `/ws/notifications`. Event perubahan
diteruskan antar-replika melalui PostgreSQL `LISTEN/NOTIFY` pada database bersama;
tidak memerlukan layanan pub/sub tambahan. Workspace, laporan, Zoom, pekerjaan,
karyawan/profil, cuti, absensi, checklist dokumen, dan status baca notifikasi
memuat ulang data terkait melalui API tanpa reload halaman. Penerima event
dibatasi sesuai akses fitur; lokasi dan data pribadi tetap diambil melalui API
yang memeriksa hak akses. Reconnect, tab yang kembali aktif, dan pemeriksaan
setiap 30 detik memulihkan event yang terlewat. Draf formulir tetap dipertahankan.

## Konfigurasi lokal dan penyimpanan

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
