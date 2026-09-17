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

Jalankan dari direktori testing di VPS (`cd /opt/my-ple-testing`).
Docker Compose menjalankan aplikasi, tetapi **ngrok harus dinyalakan terpisah**
agar domain testing dapat diakses.

Jika `ngrok: command not found`, instal sekali di VPS Ubuntu/Debian:

```sh
curl -fsSL https://ngrok-agent.s3.amazonaws.com/ngrok.asc \
  | sudo tee /etc/apt/trusted.gpg.d/ngrok.asc >/dev/null
echo "deb https://ngrok-agent.s3.amazonaws.com bookworm main" \
  | sudo tee /etc/apt/sources.list.d/ngrok.list
sudo apt update
sudo apt install -y ngrok
```

Sebagai user yang akan menjalankan ngrok (misalnya `deploy`), tambahkan authtoken
dari akun pemilik domain. Ganti `TOKEN_ANDA` dengan token dari dashboard ngrok:

```sh
ngrok config add-authtoken TOKEN_ANDA
```

Salin `.env.testing.example` menjadi `.env.testing`. Isi `NGROK_DOMAIN` dengan
hostname milik Anda di dashboard ngrok (tanpa `https://`) dan ganti password
database testing sebelum pertama kali menjalankan stack.

```sh
docker compose --env-file .env.testing -f compose.yaml -f compose.testing.yaml config --quiet
docker compose --env-file .env.testing -f compose.yaml -f compose.testing.yaml up -d --build --wait
curl -I --max-time 10 http://127.0.0.1:8081
ngrok http 8081 --url https://DOMAIN-NGROK-ANDA
```

Gunakan domain yang sama pada perintah ngrok dan `NGROK_DOMAIN`.
Contoh untuk domain testing saat ini:

```sh
ngrok http 8081 --url https://unrecessive-fibular-alyce.ngrok-free.dev
```

Jalankan hanya satu perintah ngrok untuk domain tersebut. Tunggu status `online`,
lalu biarkan terminal ngrok terbuka. `Ctrl+C`, putusnya sesi SSH, atau reboot VPS
dapat menghentikan tunnel; jalankan kembali perintah ngrok di atas jika berhenti.
Cara manual ini belum memasang service untuk otomatis berjalan setelah reboot.

Jika muncul `ERR_NGROK_3200`, periksa dari terminal SSH lain:

```sh
curl -I --max-time 10 http://127.0.0.1:8081
pgrep -a ngrok
```

Jika aplikasi merespons `200 OK` tetapi tidak ada proses ngrok, nyalakan kembali
ngrok. Jika prosesnya ada, periksa status/error pada terminal ngrok dan pastikan
domainnya sesuai. Jika port `8081` tidak merespons, periksa stack testing:

```sh
docker compose --env-file .env.testing -f compose.yaml -f compose.testing.yaml ps
```

Buka URL HTTPS ngrok untuk testing login; cookie testing memakai HTTPS.
Gateway meneruskan `/api/` ke backend (termasuk upload dan WebSocket),
dan path lainnya ke frontend. Setelah mengubah domain, jalankan ulang perintah
`up -d --build --wait` agar URL API dalam bundle frontend diperbarui.

Project `my-ple-testing` mempunyai database, upload, dan arsip sendiri.
Port lokal testing: gateway `8081`, backend `3001`, frontend `5174`.
Email testing dinonaktifkan. File `.env.testing` tidak masuk Git.
Branch Git tidak memilih environment otomatis; selalu gunakan kedua file
Compose dan `--env-file .env.testing` seperti di atas, termasuk saat `logs`,
`ps`, atau `down`. Hindari `down -v` jika data testing masih dibutuhkan.

Rujukan: [project Compose](https://docs.docker.com/compose/how-tos/project-name/)
dan [instalasi ngrok Linux](https://ngrok.com/download/linux),
[CLI ngrok](https://ngrok.com/docs/agent/cli), serta
[ERR_NGROK_3200](https://ngrok.com/docs/errors/err_ngrok_3200).

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

Setiap sesi browser memakai satu koneksi `/ws/notifications`. Event perubahan
diteruskan antar-replika melalui PostgreSQL `LISTEN/NOTIFY` pada database bersama;
tidak memerlukan layanan pub/sub tambahan. Workspace, laporan, Zoom, pekerjaan,
karyawan/profil, cuti, absensi, checklist dokumen, dan status baca notifikasi
memuat ulang data terkait melalui API tanpa reload halaman. Penerima event
dibatasi sesuai akses fitur; lokasi dan data pribadi tetap diambil melalui API
yang memeriksa hak akses. Reconnect, tab yang kembali aktif, dan pemeriksaan
setiap 30 detik memulihkan event yang terlewat. Draf formulir tetap dipertahankan.

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

## Absensi pribadi

Dashboard **Absensi** dan menu **Attendance** membuka `/attendance` untuk semua
pengguna yang masuk. WFO memeriksa jarak maksimal 100 meter dan akurasi lokasi
maksimal 100 meter saat Clock In; koordinat WFO tidak disimpan. WFA menyimpan
koordinat Clock In dan menyediakan tautan Google Maps di riwayat pribadi.
Absensi hanya mencatat waktu Clock In dan Clock Out, tanpa perhitungan
keterlambatan atau lembur. Clock Out tidak memerlukan lokasi.

Pin kantor Giri Mekar Permai adalah **-6.896211064380826, 107.69234020127173**,
dengan radius WFO **100 meter**. Docker Compose memakai pin ini sebagai default.
Untuk mengganti pin pada Docker Compose, isi
`ATTENDANCE_OFFICE_LATITUDE` dan `ATTENDANCE_OFFICE_LONGITUDE` di `.env` root
atau file yang diberikan ke `docker compose --env-file`. Untuk backend langsung,
isi variabel tersebut di environment backend. Keduanya harus berupa angka desimal
dengan titik. Untuk menjalankan backend langsung, gunakan:

```dotenv
ATTENDANCE_OFFICE_LATITUDE=-6.896211064380826
ATTENDANCE_OFFICE_LONGITUDE=107.69234020127173
```

Recreate container backend setelah mengubah pin pada Compose, atau restart proses
backend langsung. Tanpa pin yang valid, WFO diblokir dan WFA
tetap tersedia. Tidak diperlukan Google Maps API key atau library peta.

Gunakan HTTPS untuk akses dari perangkat lain; geolocation dapat digunakan di
localhost untuk pengujian. Pengguna harus mengizinkan lokasi di browser dan
mengaktifkan lokasi presisi perangkat. Browser melaporkan lokasi, bukan bukti
GPS perangkat aktif, dan koordinat dari klien tidak menjadi jaminan anti-spoofing.
Tidak ada pelacakan latar belakang atau pencatatan koordinat ke log aplikasi.

Waktu berasal dari server dan tanggal kehadiran menggunakan `Asia/Jakarta`.
Satu pengguna hanya memiliki satu record per hari dan satu record terbuka.
Record lama harus di-Clock Out sebelum Clock In hari berikutnya; tanggal awal
tetap dipertahankan. Riwayat pribadi hanya dapat dibaca pemiliknya. Migrasi tabel absensi
berjalan bersama migrasi database yang sudah ada.

### Absensi Karyawan

Menu **Absensi Karyawan** (`/attendance/overview`) tersedia untuk **HR, Management,
dan Admin**. Halaman menampilkan karyawan aktif per tanggal (default hari ini WIB),
pencarian nama, filter status, ringkasan jumlah, dan waktu Clock In/Out. Data dimuat
ulang melalui tombol atau saat jendela kembali aktif.

Status **Hadir** berlaku jika ada Clock In, termasuk setelah Clock Out. Tanpa Clock
In, pengajuan **Izin/Cuti** yang sudah disetujui ditampilkan terpisah; pengajuan
pending atau ditolak tidak dihitung. Sisanya berstatus **Belum Clock In** (hari ini)
atau **Tidak ada Clock In** (tanggal lampau), tanpa penilaian keterlambatan atau penalti.
Pengajuan izin/cuti yang bertumpuk tidak menggandakan baris karyawan.

API `GET /attendance/overview?date=YYYY-MM-DD` memeriksa role di server dan menolak
tanggal tidak valid atau masa depan. Respons berisi `date`, `todayDate`, dan
`employees` dengan `userId`, `name`, `role`, `status` (`PRESENT`, `LEAVE`, `MISSING`),
`leaveTypes`, `mode`, `clockIn`, `clockOut`, `latitude`, dan `longitude`. Lokasi Clock In
WFA ditampilkan dengan tautan Google Maps; WFO menampilkan verifikasi radius kantor
100 meter tanpa menyimpan koordinat karyawan. Tidak ada gaji atau
alasan izin/cuti dalam respons. Tidak diperlukan migrasi tambahan.

Karyawan mengacu pada akun yang **aktif saat ini** dan sudah dibuat pada tanggal
pilihan; status keaktifan historis tidak direkonstruksi. Halaman ini tidak mengubah
kewenangan persetujuan izin/cuti. Rebuild aplikasi untuk menerapkan halaman baru.

Verifikasi: `go test ./...` dari `backend`, lalu `npm run check` dan
`npm run build` dari `frontend`. Tes integrasi `TestAttendanceIntegration` memakai
fixture PostgreSQL lokal terisolasi yang sama dengan tes checklist:
`CHECKLIST_TEST_DATABASE_URL` harus menunjuk database `my_ple_checklist_test`
di `127.0.0.1`. Untuk browser, jalankan `TestAttendanceBrowserServer` dengan
`ATTENDANCE_BROWSER_SERVER=1`, Vite di port 5189 dengan
`VITE_API_URL=http://127.0.0.1:3001`, lalu `node tests/attendance.browser.mjs`.
Fixture browser mengosongkan absensi akun `attendance-browser-*` miliknya
dan membuat ulang izin Management pada database uji tersebut sebelum setiap run.


### Tabel checklist dokumen

Checklist hanya menampilkan pekerjaan ber-role Project Engineer (termasuk pekerjaan
nonaktif), bukan QA atau CIHPK. Akses tetap untuk Project Engineer, Finance, Management,
dan Admin. Setiap pekerjaan dapat memiliki beberapa tabel berjudul; gunakan panah
sebelumnya/berikutnya untuk memilih tabel, lalu **Edit Tabel Checklist** untuk mengubah
judul dan penandatangan. **Tambah Isi Dokumen** menambah baris; isi sel tetap disimpan
otomatis. Kolom Aksi dan tombol hapus tidak ditampilkan.

Migrasi otomatis mempertahankan tabel, dokumen, dan riwayat tanda tangan yang ada,
memberi judul awal **Checklist Dokumen**, dan menghapus batas satu tabel per pekerjaan.
Data checklist pekerjaan non-PE tetap tersimpan tetapi tidak tersedia pada fitur ini.
Deploy backend dan frontend bersamaan: `GET /document-checklists/{jobId}` sekarang
mengembalikan `{checklists: [...]}`; `POST` pada URL tersebut membuat tabel.
`GET`/`PUT /document-checklists/{jobId}/tables/{tableId}` membaca/mengubah tabel.
Operasi dokumen menggunakan `/document-checklists/{jobId}/tables/{tableId}/items`
dan `.../items/{itemId}`. Konfigurasi wajib menyertakan `title`; mutasi mengembalikan
`{checklist: ...}`. Route mutasi lama tanpa ID tabel tidak lagi digunakan.

Tes checklist: jalankan `go test . -run TestDocumentChecklist -v` dari `backend`
dengan `CHECKLIST_TEST_DATABASE_URL` ke database lokal terisolasi
`my_ple_checklist_test`. Untuk browser, jalankan `TestDocumentChecklistBrowserServer`
dengan `CHECKLIST_BROWSER_SERVER=1`, Vite port 5189 dengan
`VITE_API_URL=http://127.0.0.1:3001`, lalu dari `frontend` jalankan
`node tests/document-checklist.browser.mjs`.


### Bahasa, menu, dan kalender Zoom

Pemilih **Bahasa Indonesia / English** di samping notifikasi dashboard menyimpan
pilihan pada akun. Akun lama maupun baru memakai `id` secara default. Browser
mengingat bahasa terakhir untuk halaman masuk; preferensi akun mengambil alih
setelah login dan tersinkron melalui event pengguna pada sesi lain yang terbuka.
`PATCH /profile/language` menerima `{language: "id" | "en"}` dan mengembalikan
`{user}` dengan properti `language`. Endpoint hanya mengubah akun yang sedang login.
Migrasi kolom bahasa bersifat tambahan; rilis backend sebelum atau bersama frontend.

Terjemahan mencakup antarmuka situs, tanggal, status, notifikasi sistem, serta pesan
kesalahan. Nama, judul, catatan, isi laporan, email, dan file unduhan tetap seperti
semula. Tambahkan kedua bahasa untuk label baru pada `frontend/src/lib/messages.ts`,
lalu gunakan store terjemahan dari `i18n.ts` saat menampilkan label. Nilai formulir,
role untuk otorisasi, dan enum API tidak diterjemahkan. Respons error mempertahankan
field lama dan menambahkan `messageCode`; pesan yang belum dikenal memakai fallback
terjemahan di klien.

Dashboard dan sidebar memakai `navigation.ts` dengan aturan akses yang sama.
Dashboard dan Keluar tetap merupakan navigasi/aksi, bukan kartu layanan.
Kalender Zoom desktop mengikuti tinggi panel informasi melalui CSS; tablet dan
ponsel mempertahankan susunan bertumpuk dan area kalender yang dapat digulir.

Verifikasi: `npm test`, `npm run check`, `npm run build` dari `frontend`, serta
`go test ./...` dari `backend`. `TestLanguagePreference` menggunakan database uji
checklist yang terisolasi. Dengan fixture browser checklist di port 3001 dan Vite
di 5189, jalankan `node tests/language.browser.mjs` untuk memeriksa kedua bahasa,
sinkronisasi antarsesi, kartu/sidebar, formulir terbuka, dan tinggi kalender Zoom.

### Arsip pekerjaan workspace

Tombol **Arsip Pekerjaan / Archived Jobs** menampilkan pekerjaan yang diarsipkan
pada workspace saat ini, dengan pencarian, filter, dan paginasi. Detail, komentar,
dan riwayat tetap dapat dibaca. Pembuat pekerjaan dan pengelola workspace dapat
memulihkan pekerjaan dengan status, penugasan, dan validasi QA sebelumnya.

API daftar `/workspaces/{id}/tasks` serta GET detail, komentar, dan aktivitas
`/work-items/{id}` menerima `archived=true` untuk membaca arsip. Tanpa parameter
ini, perilaku aktif tetap berlaku. `POST /work-items/{id}/restore` menerima
`{ "version": number }`, mengembalikan `{ "task": ... }`, dan memancarkan
`workspace.changed`. Tidak ada migrasi baru. Rilis backend sebelum frontend.

Uji integrasi arsip menggunakan `TestRevisionIntegration/workspace_archive_and_restore`
dan database lokal `TEST_DATABASE_URL` yang sama dengan pengujian revisi.
Uji browser: jalankan fixture `TestDocumentChecklistBrowserServer` dan Vite seperti
pengujian bahasa, lalu `node tests/workspace-archive.browser.mjs` dari `frontend`.

### Automatic weekly rollover

Weekly reports close on Tuesday at 00:00 WIB. The background worker runs at
startup and every minute, snapshots overdue weeks, saves their PDFs, then resets
only the archived working rows. Manual export is not required; PDFs are available
from Weekly Report History. A PDF/storage failure retains working data for retry.

The week-selection SQL applies `AT TIME ZONE` to the complete Tuesday timestamp
expression, not the added interval. Regression coverage is
`go test . -run TestWeeklyLifecycleAutomaticRollover -count=1` from `backend`, with
`CHECKLIST_TEST_DATABASE_URL` set to the isolated local checklist test database.
The test creates and removes a disposable database and disables email delivery.
After rollout, check `weekly reset completed` logs and the recovered History
periods; retained overdue reports are processed automatically without a migration.
