// Run against the dev server with Playwright available, or set PLAYWRIGHT_MODULE.
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE || 'playwright');
import assert from 'node:assert/strict';

const origin=process.env.RESPONSIVE_ORIGIN??'http://127.0.0.1:5173';
const long = 'Koordinasi Pekerjaan Antar Tim Jember';
const now = new Date().toISOString(), day = now.slice(0,10);
const user = {id:'u1',name:long,email:long+'@example.test',username:'pegawai',roleId:1,role:'Admin',language:'id',jobTitle:long,salary:15000000,residence:long,phoneNumber:'+628123456789',employmentStatus:'Permanent',avatarUrl:'',isActive:true,createdAt:now,updatedAt:now};
const member = {id:'u1',name:long,avatarUrl:''};
const job = {id:1,divisionId:1,roleId:1,jobCategoryId:1,name:long,description:long,isActive:true,sortOrder:0,division:'Project Engineer',role:'Engineer',category:'Supervisi',referenced:false,pics:[{userId:'u1',name:long,isActive:true}]};
const workspace = {id:1,name:'Project Engineer',canManage:true,memberCount:3,activeTaskCount:4};
const task = {id:'t1',workspaceId:1,jobId:1,jobName:long,title:long,description:long,relatedUrl:'https://example.test/'+long,startDate:day,dueDate:day,status:'IN_PROGRESS',priority:'HIGH',createdBy:'u1',createdAt:now,updatedAt:now,version:1,assignees:[member],commentCount:1,canEdit:true,canDelete:true};
const note = {id:'n1',workspaceId:1,parentId:null,title:long,content:[{kind:'text',text:long,checked:false}],visibility:'DIVISION',createdBy:'u1',updatedAt:now,version:1,canEdit:true};
const report = {id:'r1',userId:'u1',jobId:1,primaryDivision:'Project Engineer',workDivision:'Supervisi',role:'Engineer',work:long,number:1,description:long,completedAt:now,pics:job.pics};
const archive = {id:'a1',division:'Project Engineer',weekStart:day,weekEnd:day,status:'finalized',finalizedAt:now,pdfGeneratedAt:now,filename:'report.pdf',items:[{...report,employeeName:long,employeeRole:'Engineer'}]};
const resource = {id:1,name:'Zoom 1',code:'Z1',description:long,meetingUrl:'https://example.test/'+long,meetingId:'123456789',hostKey:'123456',createdAt:now,updatedAt:now};
const fixtures = {
 '/auth/me':{user},'/me':{user},'/users':{users:[user]},'/roles':{roles:[{id:1,name:'Admin'}]},
 '/jobs/masters':{divisions:[{id:1,name:'Project Engineer'}],roles:[{id:1,name:'Engineer',divisionId:1}],categories:[{id:1,name:'Supervisi',divisionId:1,roleId:1}]},'/jobs':{jobs:[job]},'/jobs/pic-users':{users:[{userId:'u2',name:long,username:long,isActive:true}]},
 '/notifications':{notifications:[{id:'nt1',type:'workspace_task',actor:{id:'u2',name:long},workspace:{title:long,workspaceId:1,message:'Mengubah pekerjaan'},createdAt:now,readAt:null}]},'/notifications/unread-count':{count:1},
 '/workspaces':{workspaces:[workspace]},'/workspaces/1':{workspace,members:[member]},'/workspaces/1/jobs':{jobs:[job]},'/workspaces/1/tasks':{tasks:[task],total:1},'/workspaces/1/overview':{overview:{active:1,inProgress:1,needRevision:0,done:0,overdue:0}},'/workspaces/1/activity':{activity:[]},
 '/workspaces/1/workload':{workload:[{member,activeTaskCount:1,highPriorityCount:1,dueSoonCount:1,overdueCount:0,overlapCount:0,workloadScore:5,workloadLevel:'NORMAL',tasks:[task]}]},
 '/workspaces/1/pages':{pages:[note]},'/workspace-pages/n1':{page:note},'/work-items/t1':{task},'/work-items/t1/comments':{comments:[{id:'c1',userId:'u1',name:long,content:long,createdAt:now}]},'/work-items/t1/activity':{activity:[]},
 '/reports':{reports:[report],period:{start:day,end:day,label:'Periode aktif',readOnly:false}},'/reports/history':{archives:[archive],hasMore:false},'/reports/history/a1':{archive},'/reports/period':{start:day,end:day,label:'Periode aktif',readOnly:false},
 '/zoom-resources':{resources:[resource]},'/zoom-bookings':{bookings:[{id:'b1',userId:'u1',createdBy:long,title:long,description:long,zoomResource:resource,startAt:day+'T02:00:00Z',endAt:day+'T03:00:00Z',createdAt:now,updatedAt:now}]}
};
const browser = await chromium.launch({ channel: 'chrome', headless: true });
const attendance = { id: 'at1', date: day, mode: 'WFA', clockIn: now, clockOut: now, latitude: null, longitude: null, notes: long };
const leave = { id: 'leave1', employeeName: long, division: 'Project Engineer', requestType: 'CUTI', status: 'PENDING', startDate: day, endDate: day, reason: long, createdAt: now };
Object.assign(fixtures, {
 '/document-checklists': { jobs: [job] },
 '/attendance': { records: [attendance], today: attendance, open: null, todayDate: day, officeConfigured: true },
 '/attendance/overview': { date: day, todayDate: day, employees: [{ ...attendance, userId: user.id, name: long, role: user.role, status: 'PRESENT', leaveTypes: '' }] },
 '/leave-requests': { requests: [leave] }, '/leave-requests/approvals': { requests: [leave] },
 '/document-checklists/1': { checklists: [{ id: 1, jobId: 1, title: 'Kelengkapan Dokumen Pekerjaan', version: 1, requireDigitalSignature: true, signers: [{ id: 1, type: 'external', name: long.repeat(5), position: long.repeat(5), archived: false, displayOrder: 0 }], items: [{ id: 1, documentName: long.repeat(5), documentPosition: 'X'.repeat(255), notes: (long + '\n').repeat(20), signatures: {}, digitalSigned: false, sentDate: day }] }] },
 '/billing-documents': { jobs: [job], stages: [{ id: 1, jobId: 1, name: 'Tahap Pertama', periodLabel: 'Oktober', notes: long, updatedAt: now, version: 1, readiness: 'BELUM SIAP', documents: [], milestones: [{ id: 1, name: 'Pemeriksaan Dokumen', notes: long, readiness: 'BELUM SIAP', documents: [{ id: 1, name: 'Lembar Pengesahan', status: 'BELUM ADA', updatedAt: now }] }] }] }
});
const page = await browser.newPage({ viewport: { width: 320, height: 850 }, hasTouch: true, reducedMotion: 'reduce' });
user.language = process.env.TEST_LANGUAGE || 'id';
await page.addInitScript(language => localStorage.setItem('language', language), user.language);
const errors = [], failures = [];
const chatRequests = [], feedbackRequests = [];
let feedbackStatus = 204;
page.on('pageerror', error => errors.push(error.message));
await page.routeWebSocket(/ws\/notifications/, () => {});
await page.route('**/*', async route => {
 const url = new URL(route.request().url());
 if (/fonts\.(googleapis|gstatic)\.com/.test(url.hostname)) return route.fulfill({ body: '' });
 if (url.port !== '3000') return route.continue();
 if (url.pathname === '/hermes/chat') {
  chatRequests.push(route.request().postDataJSON());
  return route.fulfill({ status: 200, json: { answer: 'Periksa sambungan.', sources: [{ id: 'task:t1', kind: 'task', title: 'Sambungan', excerpt: 'Periksa kabel.', url: '/workspace/1?task=t1' }] }, headers: { 'Access-Control-Allow-Origin': origin, 'Access-Control-Allow-Credentials': 'true' } });
 }
 if (url.pathname === '/feedback') {
  feedbackRequests.push(route.request().postDataJSON());
  const headers = { 'Access-Control-Allow-Origin': origin, 'Access-Control-Allow-Credentials': 'true' };
  return feedbackStatus === 204 ? route.fulfill({ status: 204, headers }) : route.fulfill({ status: 503, json: { error: 'Layanan email belum tersedia. Silakan coba nanti.' }, headers });
 }
 return route.fulfill({ status: url.pathname === '/auth/me' && !fixtures[url.pathname].user ? 401 : 200, json: fixtures[url.pathname] || {}, headers: { 'Access-Control-Allow-Origin': origin, 'Access-Control-Allow-Credentials': 'true' } });
});
async function check(label, prepare) {
 for (const width of [320, 390, 430, 768, 1440]) {
  await page.setViewportSize({ width, height: 850 });
  await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
  await page.waitForTimeout(80);
  if (label === 'Checklist table') {
   const fields = page.locator('.checklist table textarea');
   assert.equal(await fields.count(), 5, 'All checklist text cells remain editable');
   assert(await fields.evaluateAll(elements => elements.every(field => field.scrollHeight <= field.clientHeight + 1 && field.scrollWidth <= field.clientWidth + 1)), width + 'px: full checklist values visible without internal scrolling');
  }
  if (label === 'Hermes panel' || label === 'Feedback tab') {
   const positions = await page.evaluate(() => {
    const panel = document.querySelector('.hermes-panel').getBoundingClientRect();
    const launcher = document.querySelector('.hermes-launcher').getBoundingClientRect();
    return { left: panel.left, right: panel.right, top: panel.top, bottom: panel.bottom, launcherTop: launcher.top, viewport: innerWidth };
   });
   assert(positions.left >= 0 && positions.right <= positions.viewport + 1 && positions.top >= 0 && positions.bottom < positions.launcherTop, `${width}px: panel fits above launcher`);
  }
  if (prepare) await prepare();
  const broken = await page.evaluate(() => {
   const found = [];
   const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
   while (walker.nextNode()) {
    const node = walker.currentNode, element = node.parentElement;
    if (!element.checkVisibility({ visibilityProperty: true }) || element.closest('script, style, [inert], .sr-only')) continue;
    let clipped = false;
    for (let p = element; p; p = p.parentElement) {
     if (getComputedStyle(p).clipPath === 'inset(50%)') clipped = true;
    }
    if (clipped) continue;
    for (const word of node.textContent.matchAll(/[A-Za-z]{3,}/g)) {
     if (word[0].length > 24) continue; // URLs and unbroken identifiers may need emergency wrapping.
     const range = document.createRange();
     range.setStart(node, word.index); range.setEnd(node, word.index + word[0].length);
     const rows = new Set([...range.getClientRects()].filter(r => r.width && r.height).map(r => Math.round(r.top)));
     if (rows.size > 1) found.push(`${element.tagName}.${element.className}: ${word[0]}`);
    }
   }
   if (document.documentElement.scrollWidth > innerWidth + 1) found.push('PAGE OVERFLOW');
   return [...new Set(found)];
  });
  if (broken.length) failures.push({ label, width, broken });
  if (process.env.SCREENSHOT_DIR && width === 320 && ['/attendance/overview', '/dashboard', 'Workspace Table'].includes(label)) {
   await page.screenshot({ path: `${process.env.SCREENSHOT_DIR}/${label.replaceAll('/', '-')}-${user.language}.png`, fullPage: true });
  }
 }
 console.log(`Checked ${label}`);
}
try {
 for (const path of ['/dashboard', '/employees', '/my-profile', '/attendance', '/attendance/overview', '/leave', '/leave/approvals', '/document-checklists', '/billing-documents', '/weekly-progress?division=Project%20Engineer', '/weekly-progress/history', '/weekly-progress/new', '/settings/jobs', '/booking-zoom', '/workspace', '/workspace/1']) {
  await page.goto(origin + path);
  await page.locator('main').waitFor();
  await page.waitForTimeout(250);
  await check(path);
  if (path === '/dashboard') {
   const chatLabel = user.language === 'en' ? 'Hermes chat' : 'Chat Hermes';
   const feedbackLabel = user.language === 'en' ? 'Send to Gmail' : 'Kirim ke Gmail';
   const emailError = user.language === 'en' ? 'Email service is unavailable. Please try later.' : 'Layanan email belum tersedia. Silakan coba nanti.';
   const successMessage = user.language === 'en' ? 'Feedback sent successfully.' : 'Kritik dan saran berhasil dikirim.';
   await check('Notifications', () => page.locator('.notification-button').click());
   await page.keyboard.press('Escape');
   assert.equal(await page.locator('.hermes-launcher').count(), 1);
   assert.equal(await page.locator('.feedback-button').count(), 0);
   await page.locator('.hermes-launcher').click();
   assert.equal(await page.locator('.hermes-panel').evaluate(el => el.getAnimations().some(animation => animation.effect.getTiming().duration > 0)), false, 'Reduced motion disables morph');
   await check('Hermes panel');
   assert.equal(await page.getByRole('tab', { name: chatLabel }).getAttribute('aria-selected'), 'true');
   await page.locator('#hermes-question').fill('Sambungan gagal');
   await page.getByRole('tab', { name: feedbackLabel }).click();
   await check('Feedback tab');
   await page.locator('#feedback-message').fill('Masukan tersimpan');
   await page.getByRole('tab', { name: chatLabel }).click();
   assert.equal(await page.locator('#hermes-question').inputValue(), 'Sambungan gagal');
   await page.locator('.chat-form button[type="submit"]').click();
   await page.getByText('Periksa sambungan.').waitFor();
   assert.equal(chatRequests.at(-1).question, 'Sambungan gagal');
   assert.equal(await page.getByRole('link', { name: 'Sambungan' }).getAttribute('href'), '/workspace/1?task=t1');
   await page.getByRole('tab', { name: chatLabel }).press('ArrowRight');
   assert.equal(await page.locator('#feedback-message').inputValue(), 'Masukan tersimpan');
   feedbackStatus = 503;
   await page.locator('.feedback-content button[type="submit"]').click();
   await page.getByRole('alert').getByText(emailError).waitFor();
   assert.equal(await page.locator('#feedback-message').inputValue(), 'Masukan tersimpan');
   feedbackStatus = 204;
   await page.locator('.feedback-content button[type="submit"]').click();
   await page.getByText(successMessage).waitFor();
   await page.waitForFunction(() => document.querySelector('.hermes-launcher') === document.activeElement);
   assert.equal(feedbackRequests.at(-1).message, 'Masukan tersimpan');
   assert.equal(await page.locator('.hermes-panel').count(), 0);
   await page.locator('.hermes-launcher').click();
   await page.getByRole('tab', { name: feedbackLabel }).click();
   assert.equal(await page.locator('#feedback-message').inputValue(), '');
   await page.keyboard.press('Escape');
   assert.equal(await page.locator('.hermes-launcher').evaluate(el => el === document.activeElement), true);
   await page.locator('.hermes-panel').waitFor({ state: 'detached' });
   await page.emulateMedia({ reducedMotion: 'no-preference' });
   for (const width of [320, 390, 430, 768, 1440]) {
    await page.setViewportSize({ width, height: 850 });
    await page.locator('.hermes-launcher').click();
    const panel = page.locator('.hermes-panel');
    await panel.waitFor();
    assert.equal(await panel.evaluate(el => el.getAnimations()[0].effect.getTiming().duration), 600, 'Morph runs at 0.75x speed');
    assert.equal(await page.locator('.hermes-launcher').evaluate(el => el === document.activeElement), true, 'Focus waits for morph');
    await page.waitForTimeout(100);
    assert.notEqual(await panel.evaluate(el => getComputedStyle(el).clipPath), 'none', 'Morph clips a shape without scaling text');
    await page.locator('.hermes-launcher').click();
    assert.equal(await panel.evaluate(el => el.inert), true, 'Closing panel is inert');
    await page.waitForTimeout(40);
    await page.locator('.hermes-launcher').click();
    await page.waitForFunction(() => document.querySelector('[role="tab"][aria-selected="true"]') === document.activeElement);
    assert.equal(await panel.count(), 1, 'Rapid toggles keep one panel');
    await page.getByRole('tab', { name: chatLabel }).click();
    await page.waitForTimeout(260);
    assert.equal(await page.locator('.tabs').evaluate(el => getComputedStyle(el, '::before').transitionDuration), '0.24s');
    assert.equal(await page.getByText('Periksa sambungan.').isVisible(), true, 'Chat history survives morph');
    await page.getByRole('tab', { name: chatLabel }).press('End');
    assert.equal(await page.getByRole('tab', { name: feedbackLabel }).getAttribute('aria-selected'), 'true');
    await page.waitForTimeout(260);
    const bounds = await panel.boundingBox();
    const button = await page.locator('.hermes-launcher').boundingBox();
    assert(bounds.x >= 0 && bounds.x + bounds.width <= width + 1 && bounds.y >= 0 && bounds.y + bounds.height < button.y, `${width}px: expanded morph fits viewport`);
    const sendButton = await page.locator('.feedback-content button[type="submit"]').boundingBox();
    assert(sendButton.y >= bounds.y && sendButton.y + sendButton.height <= bounds.y + bounds.height, `${width}px: feedback send stays visible`);
    if (process.env.SCREENSHOT_DIR && width === 390) await page.screenshot({ path: `${process.env.SCREENSHOT_DIR}/assistant-${user.language}.png` });
    await page.keyboard.press('Escape');
    assert.equal(await page.locator('.hermes-launcher').evaluate(el => el === document.activeElement), true);
    await panel.waitFor({ state: 'detached' });
   }
   await page.emulateMedia({ reducedMotion: 'reduce' });
  }
  if (path === '/booking-zoom') {
   await page.locator('.booking-header .zoom-primary').click();
   await check('Zoom booking dialog');
   await page.locator('.zoom-close').click();
  }
  if (path === '/document-checklists') {
   await page.locator('.job-heading').click();
   await page.locator('.checklist table').waitFor();
   await check('Checklist table');
  }
 }
 for (const [index, tab] of [[1, 'Table'], [3, 'Board'], [4, 'Timeline'], [5, 'Calendar'], [6, 'Workload'], [7, 'Notes']]) {
  await page.locator('.ws-tabs button').nth(index).click();
  await page.waitForTimeout(150);
  await check(`Workspace ${tab}`);
 }
 fixtures['/auth/me'] = {};
 await page.goto(origin);
 await page.locator('#username').waitFor();
 await check('Login');
 console.log(JSON.stringify(failures, null, 2));
 assert.deepEqual(errors, [], 'No browser runtime errors');
 assert.deepEqual(failures, [], 'Words must not break into vertical fragments');
} finally { await browser.close(); }
