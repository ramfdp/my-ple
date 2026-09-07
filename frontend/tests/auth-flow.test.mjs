// Run with node tests/auth-flow.test.mjs while the frontend dev server is running.
// PLAYWRIGHT_MODULE may point to an existing Playwright installation.
import assert from 'node:assert/strict';
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE || 'playwright');
const browser = await chromium.launch({ channel: 'msedge', headless: true });
const context = await browser.newContext();
const user = { id: 'test-user', name: 'Auth Test', role: 'Admin' };
let valid = false;
const requests = [];
let releaseInitialization;
const initialization = new Promise((resolve) => { releaseInitialization = resolve; });
let initializationStarted;
const started = new Promise((resolve) => { initializationStarted = resolve; });
let firstCheck = true;
await context.route('http://localhost:3000/**', async (route) => {
	const request = route.request();
	const path = new URL(request.url()).pathname;
	if (path === '/auth/me' && firstCheck) {
		firstCheck = false;
		initializationStarted();
		await initialization;
	}
	const cookie = (await request.allHeaders()).cookie || '';
	let status = 200;
	let body = { user };
	const headers = { 'Content-Type': 'application/json' };
	if (path === '/auth/login') {
		valid = true;
		headers['Set-Cookie'] = 'my_ple_session=browser-test; Path=/; HttpOnly; SameSite=Lax';
	} else if (!valid || !cookie.includes('my_ple_session=browser-test')) {
		status = 401; body = { error: 'Autentikasi diperlukan.' };
	} else if (path === '/auth/logout') {
		valid = false;
		headers['Set-Cookie'] = 'my_ple_session=; Path=/; HttpOnly; Max-Age=0';
	} else if (path === '/notifications') body = { notifications: [] };
	else if (path === '/notifications/unread-count') body = { count: 0 };
	requests.push({ path, status, cookieSent: cookie.includes('my_ple_session=browser-test') });
	await route.fulfill({ status, headers, body: JSON.stringify(body) });
});
try {
	const page = await context.newPage();
	const errors = [];
	page.on('pageerror', (error) => errors.push(error.message));
	await page.goto('http://localhost:5173/');
	await started;
	assert.equal(await page.locator('[role=status]').innerText(), 'Memeriksa sesi…');
	assert.equal(await page.locator('#username').count(), 0, 'login must wait for initialization');
	releaseInitialization();
	const login = async () => {
		await page.locator('#username').fill('test-user');
		await page.locator('#password').fill('test-password');
		const offset = requests.length;
		await page.locator('button[type=submit]').click();
		await page.waitForURL('**/dashboard');
		await page.locator('.dashboard-page').waitFor({ timeout: 5000 });
		assert.match(await page.locator('h1').innerText(), /Auth Test/);
		const auth = requests.slice(offset).filter(({ path }) => path.startsWith('/auth/'));
		assert.deepEqual(auth.map(({ path, status }) => [path, status]), [['/auth/login', 200], ['/auth/me', 200]]);
		assert.equal(auth[1].cookieSent, true);
	};
	await login();
	console.log('PASS: login renders dashboard without refresh');
	await page.reload();
	await page.locator('.dashboard-page').waitFor();
	console.log('PASS: refresh preserves the session');
	const tab = await context.newPage();
	await tab.goto('http://localhost:5173/dashboard');
	await tab.locator('.dashboard-page').waitFor();
	await tab.close();
	console.log('PASS: another tab validates the session');
	await page.locator('.logout-button').click();
	await page.waitForURL('http://localhost:5173/');
	await page.goto('http://localhost:5173/dashboard');
	await page.waitForURL('http://localhost:5173/');
	await page.locator('#username').waitFor();
	console.log('PASS: logout blocks direct dashboard access');
	await login();
	console.log('PASS: second login renders without a transient 401');
	valid = false;
	await page.evaluate(async () => {
		const { api } = await import('/src/lib/api.ts');
		await api('/auth/me').catch(() => {});
	});
	await page.waitForURL('http://localhost:5173/');
	await page.locator('#username').waitFor();
	assert.deepEqual(errors, []);
	console.log('PASS: revoked session redirects to login');
} finally {
	console.log(JSON.stringify(requests));
	await browser.close();
}
