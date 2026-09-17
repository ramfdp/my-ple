// Run the guarded TestAttendanceBrowserServer fixture and Vite on port 5189 first.
import assert from 'node:assert/strict';
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE ?? '../../graphify-out/responsive-tools/node_modules/playwright/index.mjs');
const browser = await chromium.launch({ executablePath: process.env.CHROME_PATH ?? 'C:/Program Files/Google/Chrome/Application/chrome.exe', headless: true, args: ['--disable-gpu', '--in-process-gpu'] });
const origin = 'http://127.0.0.1:5189', backend = 'http://127.0.0.1:3001';
const context = await browser.newContext({ viewport: { width: 1440, height: 1000 }, geolocation: { latitude: 0, longitude: 0, accuracy: 5 }, permissions: ['geolocation'] });
const page = await context.newPage(), errors = [];
page.on('pageerror', error => errors.push(error.message));
await context.addInitScript(() => {
 const locate = navigator.geolocation.getCurrentPosition.bind(navigator.geolocation);
 navigator.geolocation.getCurrentPosition = (success, failure, options) => {
  if (window.attendanceGeoError) failure({ code: window.attendanceGeoError });
  else locate(success, failure, options);
 };
});
async function login(role) {
 console.log('Attendance browser: ' + role);
 await context.clearCookies(); await page.goto(origin);
 await page.getByLabel('Username', { exact: true }).fill('attendance-browser-' + role);
 await page.getByLabel('Password', { exact: true }).fill('Checklist-test-only-2026');
 await page.getByRole('button', { name: /Masuk|Log in|Login/i }).click();
 await page.waitForURL('**/dashboard', { waitUntil: 'domcontentloaded' });
 await page.locator('a.menu-card[href="/attendance"]').click();
 await page.waitForURL('**/attendance'); await idle();
}
async function idle() { await page.locator('.clock-panel[aria-busy="false"]').waitFor(); }
async function history() {
 const response = await context.request.get(backend + '/attendance');
 assert.equal(response.status(), 200); return response.json();
}
async function clickClock(name) {
 await page.getByRole('button', { name, exact: true }).click(); await idle();
}
try {
 await login('Admin');
 await page.getByRole('button', { name: 'WFO Work From Office' }).focus();
 await page.keyboard.press('Enter');
 assert.equal(await page.getByRole('button', { name: 'WFO Work From Office' }).getAttribute('aria-pressed'), 'true');
 await context.setGeolocation({ latitude: 1, longitude: 0, accuracy: 5 });
 await clickClock('Clock In'); await page.getByRole('alert').filter({ hasText: 'di luar radius' }).waitFor();
 assert.equal((await history()).records.length, 0);
 await context.setGeolocation({ latitude: 0, longitude: 0, accuracy: 101 });
 await clickClock('Clock In'); await page.getByRole('alert').filter({ hasText: 'belum cukup akurat' }).waitFor();
 await context.setGeolocation({ latitude: 0, longitude: 0, accuracy: 5 });
 await clickClock('Clock In'); await page.getByRole('button', { name: 'Clock Out', exact: true }).waitFor();
 let record = (await history()).open;
 assert.equal(record.mode, 'WFO'); assert.equal(record.latitude, null); assert.equal(record.longitude, null);
 await page.reload(); await idle();
 assert.equal(await page.getByRole('button', { name: 'WFA Work From Anywhere' }).count(), 0);
 await page.evaluate(() => { window.attendanceGeoError = 1; });
 await clickClock('Clock Out'); await page.getByText('Absensi hari ini selesai.').waitFor();
 assert.ok((await history()).today.clockOut);
 for (const width of [320, 768, 1024, 1440]) {
  await page.setViewportSize({ width, height: 1000 });
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth), true, 'overflow at ' + width);
  if (width === 320 || width === 1440) await page.screenshot({ path: `../graphify-out/attendance-${width}.png`, fullPage: true, animations: 'disabled' });
 }
 await page.getByLabel('Bulan', { exact: true }).fill('2000-01'); await page.getByLabel('Bulan', { exact: true }).press('Tab'); await idle();
 await page.getByText('Belum ada absensi pada bulan ini.').waitFor();
 await page.getByText('Absensi hari ini selesai.').waitFor();

 await login('Finance');
 await page.getByRole('button', { name: 'WFA Work From Anywhere' }).click();
 for (const [code, text] of [[1, 'Izin lokasi ditolak'], [2, 'Lokasi tidak dapat ditemukan'], [3, 'Pencarian lokasi habis waktu']]) {
  await page.evaluate(code => { window.attendanceGeoError = code; }, code);
  await clickClock('Clock In'); await page.getByRole('alert').filter({ hasText: text }).waitFor();
 }
 assert.equal((await history()).records.length, 0);
 await page.evaluate(() => { window.attendanceGeoError = 0; });
 await context.setGeolocation({ latitude: -6.2, longitude: 106.8, accuracy: 1000 });
 await clickClock('Clock In');
 record = (await history()).open;
 assert.equal(record.latitude, -6.2); assert.equal(record.longitude, 106.8);
 const map = page.getByRole('link', { name: /Lihat lokasi Clock In/ });
 assert.equal(await map.getAttribute('href'), 'https://www.google.com/maps?q=-6.2,106.8');
 await page.evaluate(() => { window.attendanceGeoError = 1; });
 await clickClock('Clock Out'); await page.getByText('Absensi hari ini selesai.').waitFor();

 await login('HR');
 // Verify unconfigured-office UI without changing fixture configuration for other users.
 await page.route('**/attendance?month=*', async route => {
  const response = await route.fetch(); const data = await response.json();
  await route.fulfill({ response, json: { ...data, officeConfigured: false } });
 });
 await page.reload(); await idle();
 await page.getByRole('button', { name: 'WFO Work From Office' }).click();
 assert.equal(await page.getByRole('button', { name: 'Clock In', exact: true }).isDisabled(), true);
 await page.getByText('Lokasi kantor belum dikonfigurasi.', { exact: false }).waitFor();
 await page.unroute('**/attendance?month=*');
 await page.getByRole('button', { name: 'WFA Work From Anywhere' }).click();
 // Commit the request, lose its response, and temporarily fail reconciliation.
 await page.route('**/attendance/clock-in', async route => {
  const response = await route.fetch(); assert.equal(response.status(), 200);
  await page.route('**/attendance?month=*', route => route.abort());
  await route.abort();
 });
 await clickClock('Clock In');
 await page.getByRole('button', { name: 'Muat ulang data' }).waitFor();
 assert.equal(await page.getByRole('button', { name: 'Clock In', exact: true }).isDisabled(), true);
 await page.unroute('**/attendance?month=*'); await page.unroute('**/attendance/clock-in');
 await page.getByRole('button', { name: 'Muat ulang data' }).click(); await idle();
 await page.getByRole('button', { name: 'Clock Out', exact: true }).waitFor();
 assert.equal((await history()).records.length, 1);

 // Overview uses real API data, and the fixture gives Management approved izin.
 for (const role of ['HR', 'Management', 'Admin']) {
  await login(role);
  await page.getByRole('link', { name: 'Absensi Karyawan', exact: true }).click();
  await page.locator('.overview-panel[aria-busy="false"]').waitFor();
  await page.getByRole('heading', { name: 'Absensi Karyawan', exact: true }).waitFor();
  await page.getByLabel('Cari nama').fill('Attendance HR');
  await page.getByRole('cell', { name: 'Belum Clock Out', exact: true }).waitFor();
  await page.getByLabel('Cari nama').fill('Attendance Management');
  await page.getByRole('cell', { name: 'Izin', exact: true }).waitFor();
 }
 await page.getByLabel('Status', { exact: true }).selectOption('MISSING');
 await page.getByText('Tidak ada karyawan yang sesuai dengan pencarian atau filter.').waitFor();
 await page.getByLabel('Cari nama').fill('Attendance IT');
 await page.getByRole('cell', { name: 'Belum Clock In', exact: true }).waitFor();
 await page.getByLabel('Status', { exact: true }).selectOption('PRESENT');
 await page.getByLabel('Cari nama').fill('Attendance Finance');
 await page.getByRole('cell', { name: 'Hadir', exact: true }).waitFor();
 assert.equal(await page.getByRole('cell', { name: 'Belum Clock Out', exact: true }).count(), 0);
 await page.getByLabel('Status', { exact: true }).selectOption('');
 await page.getByLabel('Cari nama').fill('');
 for (const width of [320, 768, 1024, 1440]) {
  await page.setViewportSize({ width, height: 1000 });
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth), true, 'overview overflow at ' + width);
  if (width === 320 || width === 1440) await page.screenshot({ path: `../graphify-out/attendance-overview-${width}.png`, fullPage: true, animations: 'disabled' });
 }
 const date = page.getByLabel('Tanggal', { exact: true });
 const today = await date.inputValue();
 await date.fill('0001-01-01'); await date.press('Tab');
 await page.getByText('Belum ada karyawan aktif pada tanggal ini.').waitFor();
 await date.fill(today); await date.press('Tab');
 await page.locator('.overview-panel[aria-busy="false"]').waitFor();
 await page.route('**/attendance/overview?date=*', route => route.fulfill({ status: 500, contentType: 'application/json', body: JSON.stringify({ error: 'Overview test error' }) }));
 await page.getByRole('button', { name: 'Muat ulang', exact: true }).click();
 await page.getByRole('alert').filter({ hasText: 'Overview test error' }).waitFor();
 assert.equal(await page.locator('tbody tr').count(), 0, 'failed refresh must not show stale attendance');
 await page.unroute('**/attendance/overview?date=*');
 await page.getByRole('button', { name: 'Muat ulang', exact: true }).focus();
 await page.keyboard.press('Enter');
 await page.locator('.overview-panel[aria-busy="false"]').waitFor();
 await page.getByRole('cell', { name: 'Izin', exact: true }).waitFor();
 await login('Finance');
 assert.equal(await page.getByRole('link', { name: 'Absensi Karyawan', exact: true }).count(), 0);
 await page.goto(origin + '/attendance/overview');
 await page.getByRole('alert').filter({ hasText: 'hanya untuk HR, Management, dan Admin' }).waitFor();
 assert.equal((await context.request.get(backend + '/attendance/overview')).status(), 403);
 assert.deepEqual(errors, []);
 console.log('Attendance browser checks passed: personal attendance, overview permissions, approved leave, filters, dates, recovery, keyboard, and 4 viewport sizes.');
} catch (error) {
 console.error('Browser state:', page.url(), await page.locator('body').innerText(), errors);
 await page.screenshot({ path: '../graphify-out/attendance-failure.png', fullPage: true });
 throw error;
} finally { await browser.close(); }
