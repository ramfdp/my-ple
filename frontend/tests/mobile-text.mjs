// Start the frontend, then run with Playwright available (or set PLAYWRIGHT_MODULE).
import assert from 'node:assert/strict';
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE || 'playwright');
const origin = process.env.RESPONSIVE_ORIGIN || 'http://127.0.0.1:5173';
const browser = await chromium.launch({ headless: true, channel: 'chrome' });
const page = await browser.newPage({ viewport: { width: 320, height: 800 }, hasTouch: true });
const user = { id: 'test', name: 'Layout Test', role: 'Admin', language: 'id', isActive: true };
const job = { id: 1, name: 'Pengawasan pekerjaan proyek', description: 'Pemeriksaan pekerjaan di lapangan', division: 'Project Engineer', role: 'Project Engineer', category: 'MK/Supervisi', divisionId: 1, roleId: 1, jobCategoryId: 1, isActive: true, pics: [] };
const fixtures = {
  '/auth/me': { user },
  '/jobs/masters': { divisions: [{ id: 1, name: job.division }], roles: [{ id: 1, divisionId: 1, name: job.role }], categories: [{ id: 1, divisionId: 1, roleId: 1, name: job.category }] },
  '/jobs': { jobs: [job, { ...job, id: 2, isActive: false }] },
  '/notifications': { notifications: [] }, '/notifications/unread-count': { count: 0 }
};
await page.routeWebSocket(/ws\/notifications/, () => {});
await page.route('**/*', async route => {
  const url = new URL(route.request().url());
  if (url.hostname.endsWith('googleapis.com') || url.hostname.endsWith('gstatic.com')) return route.fulfill({ body: '' });
  if (url.port !== '3000') return route.continue();
  await route.fulfill({ json: fixtures[url.pathname] || {}, headers: { 'Access-Control-Allow-Origin': origin, 'Access-Control-Allow-Credentials': 'true' } });
});
const errors = [];
page.on('pageerror', error => errors.push(error.message));
try {
  await page.goto(`${origin}/settings/jobs`);
  await page.locator('.status-btn').first().waitFor();
  for (const width of [320, 360, 375, 390, 414, 430, 560, 768, 844, 1440]) {
    await page.setViewportSize({ width, height: 800 });
    const broken = await page.locator('.work-table th, .cell-division, .cell-role, .cell-category, .status-btn, .cell-actions button').evaluateAll(elements => elements.flatMap(element => {
      const walker = document.createTreeWalker(element, NodeFilter.SHOW_TEXT);
      const lines = new Set();
      while (walker.nextNode()) {
        const node = walker.currentNode;
        for (let i = 0; i < node.textContent.length; i++) {
          if (!node.textContent[i].trim()) continue;
          const range = document.createRange();
          range.setStart(node, i); range.setEnd(node, i + 1);
          lines.add(Math.round(range.getBoundingClientRect().top));
        }
      }
      return lines.size > 1 ? [element.textContent.trim()] : [];
    }));
    assert.deepEqual(broken, [], `${width}px: labels must stay horizontal`);
    assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), `${width}px: page overflow`);
    console.log(`PASS ${width}px: horizontal labels, no page overflow`);
  }
  assert.deepEqual(errors, []);
  if (process.env.SCREENSHOT) {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.locator('.work-table-wrap').scrollIntoViewIfNeeded();
    await page.locator('.work-table-wrap').evaluate(element => { element.scrollLeft = element.scrollWidth; });
    await page.screenshot({ path: process.env.SCREENSHOT });
  }
} finally {
  await browser.close();
}
