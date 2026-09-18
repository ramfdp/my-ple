// Run with the frontend dev server on port 5194. All API/email calls are mocked.
import assert from 'node:assert/strict';
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE ?? '../../graphify-out/responsive-tools/node_modules/playwright/index.mjs');
const origin = process.env.RESPONSIVE_ORIGIN ?? 'http://127.0.0.1:5194';
const browser = await chromium.launch({ executablePath: process.env.CHROME_PATH ?? 'C:/Program Files/Google/Chrome/Application/chrome.exe', headless: true });
try {
 const context = await browser.newContext();
 const user = { id: 'test', name: 'Pengguna', email: 'user@gmail.com', role: 'Admin', language: 'id' };
 let status = 502, calls = 0;
 await context.route('**/*', async route => {
  const url = new URL(route.request().url());
  if (url.origin === origin) return route.continue();
  if (!['localhost', '127.0.0.1'].includes(url.hostname)) return route.abort();
  const headers = { 'Access-Control-Allow-Origin': origin, 'Access-Control-Allow-Credentials': 'true', 'Access-Control-Allow-Headers': 'Content-Type', 'Access-Control-Allow-Methods': 'GET,POST,OPTIONS' };
  if (route.request().method() === 'OPTIONS') return route.fulfill({ status: 204, headers });
  if (url.pathname === '/feedback') {
   calls++;
   assert.deepEqual(route.request().postDataJSON(), { message: 'Masukan 👍\nBaris kedua' });
   await new Promise(resolve => setTimeout(resolve, 150));
   return route.fulfill({ status, headers, contentType: 'application/json', body: status === 204 ? '' : JSON.stringify({ error: 'Kritik dan saran gagal dikirim. Silakan coba lagi.' }) });
  }
  return route.fulfill({ headers, contentType: 'application/json', body: JSON.stringify(url.pathname === '/auth/me' ? { user } : { notifications: [], count: 0 }) });
 });
 await context.routeWebSocket(/.*/, socket => socket.close());
 const page = await context.newPage();
 const errors = []; page.on('pageerror', error => errors.push(error.message));
 for (const width of [320, 768, 1024, 1440]) {
  await page.setViewportSize({ width, height: 800 });
  await page.goto(origin + '/dashboard');
  const button = page.getByRole('button', { name: 'Kritik & Saran', exact: true });
  await button.click();
  const field = page.getByLabel('Kritik atau saran Anda');
  assert.equal(await field.evaluate(el => el === document.activeElement), true);
  assert.equal(await field.getAttribute('maxlength'), '4000');
  assert.equal(await page.locator('dialog').evaluate(el => el.scrollWidth <= el.clientWidth), true);
  const bounds = await page.locator('dialog').boundingBox();
  assert.ok(bounds.x >= 0 && bounds.x + bounds.width <= width && bounds.y >= 0);
  await page.keyboard.press('Shift+Tab');
  assert.equal(await page.evaluate(() => !!document.activeElement.closest('dialog')), true);
  await page.keyboard.press('Escape');
  assert.equal(await button.evaluate(el => el === document.activeElement), true);
 }
 await page.getByRole('button', { name: 'Kritik & Saran', exact: true }).click();
 const field = page.getByLabel('Kritik atau saran Anda');
 await field.fill('Masukan 👍\nBaris kedua');
 await page.getByRole('button', { name: 'Kirim', exact: true }).click();
 await page.getByRole('alert').waitFor();
 assert.equal(await field.inputValue(), 'Masukan 👍\nBaris kedua');
 assert.equal(calls, 1);
 status = 204;
 await page.getByRole('button', { name: 'Kirim', exact: true }).click();
 await page.locator('dialog').waitFor({ state: 'hidden' });
 await page.getByText('Kritik dan saran berhasil dikirim.', { exact: true }).waitFor();
 assert.equal(calls, 2);
 await page.getByRole('button', { name: 'Kritik & Saran', exact: true }).click();
 assert.equal(await field.inputValue(), '');
 await page.emulateMedia({ reducedMotion: 'reduce' });
 assert.ok(await page.locator('dialog').evaluate(el => parseFloat(getComputedStyle(el).transitionDuration) <= 0.001));
 await page.setViewportSize({ width: 320, height: 360 });
 assert.equal(await page.locator('dialog').evaluate(el => el.getBoundingClientRect().bottom <= innerHeight), true);
 await page.screenshot({ path: '../graphify-out/feedback-mobile.png' });
 assert.deepEqual(errors, []);
 console.log('Feedback browser checks passed: responsive, focus, reduced motion, failed/successful submission.');
} finally { await browser.close(); }
