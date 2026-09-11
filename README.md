# My PLE — Docker

Jalankan dari root proyek dengan Docker Compose v2.24 atau lebih baru:

```sh
docker compose up -d --build
docker compose ps
```

Frontend: http://localhost:5173. Backend: http://localhost:3000
(health check: `/health`). Compose menjalankan PostgreSQL baru; skema dibuat
otomatis oleh backend. Data database lokal yang sudah ada tidak diimpor otomatis.

## Deployment production (VPS)

Simpan konfigurasi production di `.env.production` pada root proyek, lalu selalu
sertakan file tersebut pada setiap perintah Compose:

```sh
git pull
docker compose --env-file .env.production build
docker compose --env-file .env.production up -d --scale backend=3 --wait
docker compose --env-file .env.production ps
```

Jangan menjalankan build production tanpa `--env-file .env.production`.
`VITE_API_URL` dimasukkan ke bundle frontend saat proses build; jika argumen
tersebut tidak terbaca, browser akan mencoba mengakses `localhost:3000`.

Setelah mengubah `VITE_API_URL`, paksa rebuild frontend dan recreate hanya
container frontend tanpa mengganggu backend:

```sh
docker compose --env-file .env.production build --no-cache frontend
docker compose --env-file .env.production up -d --no-deps --force-recreate frontend
```

Verifikasi bundle dan endpoint production:

```sh
docker compose --env-file .env.production exec -T frontend \
  grep -Roh 'https://myhis.dpdns.org/api' /app/build/client
curl -I https://myhis.dpdns.org/
curl -I https://myhis.dpdns.org/api/health
```

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
