// TEST_LAYOUT=1 node tests/weekly-history.browser.test.mjs (Vite on port 5190).
import assert from 'node:assert/strict';
import { setTimeout as delay } from 'node:timers/promises';
import { writeFile } from 'node:fs/promises';

export async function testLayout({ send, evaluate, waitFor, click }) {
 const issues = [];
 for (const path of ['/weekly-progress/history', '/weekly-progress?division=Project%20Engineer', '/weekly-progress/new', '/dashboard', '/my-profile', '/employees', '/booking-zoom']) {
  await send('Page.navigate', { url: `http://127.0.0.1:5190${path}` });
  await waitFor('document.querySelector("main")');
  await delay(800);
  if (path === '/weekly-progress/history') { await click('Detail'); await waitFor('document.body.innerText.includes("Archived Employee")'); }
  if (process.env.CAPTURE_LAYOUT && ['/weekly-progress/history', '/my-profile'].includes(path)) {
   await send('Emulation.setDeviceMetricsOverride', { width: 480, height: 1100, deviceScaleFactor: 1, mobile: false });
   await delay(100);
   const screenshot = await send('Page.captureScreenshot', { format: 'png' });
   await writeFile(`.svelte-kit/layout-${path.split('/').at(-1)}.png`, Buffer.from(screenshot.data, 'base64'));
  }
  if (path === '/employees') await click('Add employee');
  if (path === '/my-profile') await click('Edit Profile');
  if (path === '/booking-zoom') await click('+ Buat Booking');
  if (path === '/dashboard') await evaluate("document.querySelector('.notification-button').click()");
  await evaluate(`document.querySelectorAll('.task-description, [aria-label="Isi arsip"] td, .identity-copy h2, .dashboard-header h1 span:first-child').forEach(e => e.textContent = 'LongEmployeeProjectDescription'.repeat(12))`);
  // Effective CSS viewport sizes for desktop zoom 80%, 100%, 125%, plus a resize sweep.
  const widths = [...new Set([1440, 1280, 1100, 1024, 900, 869, 868, 768, 640, 560, 480, 375, 320, ...[1440,1280,1024,768,480].flatMap(w => [.8,1,1.25].map(z => Math.round(w/z))), ...Array.from({length: 22}, (_,i) => 320+i*53)])];
  for (const width of widths) {
   await send('Emulation.setDeviceMetricsOverride', { width, height: 600, deviceScaleFactor: 1, mobile: false });
   await delay(30);
   const failures = await evaluate(`(() => {
    const failures = [], rect = e => e.getBoundingClientRect();
    if (document.documentElement.scrollWidth > innerWidth + 1) failures.push('page overflow');
    const bar = document.querySelector('.mobile-topbar'), main = document.querySelector('main');
    if (bar && rect(bar).height && rect(main).top < rect(bar).bottom) failures.push('topbar beside content');
    for (const e of document.querySelectorAll('th,td,input,select,.menu-card,.week-actions,.info-content,.info-value,.status-badge')) {
     if (e.clientWidth && e.scrollWidth > e.clientWidth + 2) failures.push('clipped/overflowing ' + e.tagName + '.' + e.className);
    }
    for (const e of document.querySelectorAll('[role="dialog"],.notification-panel')) {
     const r = rect(e);
     if (r.left < 0 || r.right > innerWidth + 1 || r.top < 0 || r.bottom > innerHeight + 1) failures.push('overlay outside viewport');
     const hit = document.elementFromPoint(r.left + r.width/2, Math.min(r.bottom-2, innerHeight-2));
     if (!e.contains(hit)) failures.push('overlay behind content');
    }
    for (const group of document.querySelectorAll('.history-row-actions,.report-actions,.task-list-header,.schedule-toolbar,.modal-actions,.form-actions,.zoom-modal footer')) {
     const children = [...group.children].map(rect);
     for(let i=0;i<children.length;i++) for(let j=i+1;j<children.length;j++) {
      const a=children[i], b=children[j];
      if(Math.min(a.right,b.right)-Math.max(a.left,b.left)>1 && Math.min(a.bottom,b.bottom)-Math.max(a.top,b.top)>1) failures.push('overlapping controls');
     }
    }
    return [...new Set(failures)];
   })()`);
   if (failures.length) issues.push(`${path} @${width}: ${failures.join(', ')}`);
  }
  for (const selector of ['.work-table-wrap', '.table-wrap', '.calendar-scroll']) {
   assert.ok(await evaluate(`(() => { const e=document.querySelector('${selector}'); if(!e || e.scrollWidth<=e.clientWidth)return true; e.scrollLeft=100; return e.scrollLeft>0; })()`), `${path}: table/calendar cannot scroll`);
  }
  if (path === '/my-profile') { await click('Cancel'); await waitFor('!document.querySelector("[role=dialog]")'); }
  if (path === '/booking-zoom') { await click('Batal'); await waitFor('!document.querySelector("[role=dialog]")'); }
  if (path === '/employees') {
   await click('Cancel');
   assert.ok(await evaluate(`(() => {const e=document.querySelector('.table-wrap'); e.scrollLeft=100; return e.scrollLeft>0 && document.documentElement.scrollWidth<=innerWidth+1;})()`), 'Employee table must scroll inside the page');
  }
 }
 await send('Page.navigate', { url: 'http://127.0.0.1:5190/weekly-progress?division=AI%20Dev' });
 await waitFor("document.querySelector('.edit-report')");
 await click('Export PDF');
 await click('Edit'); await delay(300);
 await send('Emulation.setDeviceMetricsOverride', { width: 480, height: 400, deviceScaleFactor: 1, mobile: false });
 assert.ok(await evaluate(`(() => {const e=document.querySelector('[role=dialog]'),r=e.getBoundingClientRect();return r.top>=0&&r.bottom<=innerHeight+1&&e.scrollWidth<=e.clientWidth+1;})()`), 'Report drawer must fit short viewport');
 await click('Batal'); await delay(300); await click('Hapus'); await delay(300);
 assert.ok(await evaluate(`(() => {const e=document.querySelector('.delete-report-confirm');e.scrollIntoView();const r=e.getBoundingClientRect();return document.elementFromPoint(r.x+r.width/2,r.y+r.height/2)===e;})()`), 'Delete confirmation must remain clickable above navigation');
 await click('Batal'); await delay(300);
 await evaluate("document.querySelector('.mobile-menu-btn').click()"); await delay(300);
 assert.ok(await evaluate(`(() => {const e=document.querySelector('.sidebar-container a[href="/dashboard"]'),r=e.getBoundingClientRect();return e.contains(document.elementFromPoint(r.x+r.width/2,r.y+r.height/2));})()`), 'Mobile navigation must be clickable');
 await evaluate("document.querySelector('.sidebar-container a[href=\"/dashboard\"]').click()");
 await waitFor('document.querySelector(".dashboard-page")');
 assert.deepEqual(issues, [], issues.join('\n'));
 console.log('PASS: seven routes, responsive/zoom viewport sweep, long text, forms, overlays, table/calendar scrolling');
}
