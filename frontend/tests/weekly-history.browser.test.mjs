// Run with Vite on 127.0.0.1:5190; uses installed Edge/Chromium, no browser dependency.
import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { mkdtemp, readFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { setTimeout as delay } from 'node:timers/promises';

const profile = await mkdtemp(join(tmpdir(), 'weekly-browser-'));
const chrome = spawn(process.env.CHROME_PATH || 'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe',
 ['--headless=new', '--remote-debugging-port=0', `--user-data-dir=${profile}`, '--no-first-run', 'about:blank'], { windowsHide: true, stdio: 'ignore' });
let socket;
try {
 let port;
 for (let i = 0; i < 100; i++) {
  try { port = (await readFile(join(profile, 'DevToolsActivePort'), 'utf8')).split('\n')[0]; break; } catch { await delay(100); }
 }
 assert.ok(port, 'Chrome did not start');
 const tabs = await (await fetch(`http://127.0.0.1:${port}/json`)).json();
 socket = new WebSocket(tabs.find(t => t.type === 'page').webSocketDebuggerUrl);
 await new Promise((resolve, reject) => { socket.onopen = resolve; socket.onerror = reject; });
 let sequence = 0;
 const pending = new Map(), exceptions = [], requests = [];
 socket.onclose = event => { for (const p of pending.values()) p.reject(new Error(`Chrome connection closed: ${event.code} ${event.reason}`)); };
 const send = (method, params = {}) => new Promise((resolve, reject) => {
  const id = ++sequence; pending.set(id, { resolve, reject }); socket.send(JSON.stringify({ id, method, params }));
 });
 const archive = { id: 'history-fixture', division: 'Project Engineer', weekStart: '2026-08-30T17:00:00Z', weekEnd: '2026-09-06T17:00:00Z', finalizedAt: '2026-09-06T17:00:00Z', pdfGeneratedAt: '2026-09-06T17:00:00Z', status: 'finalized', filename: 'weekly-report-project-engineer-2026-08-31-to-2026-09-06.pdf' };
 const item = { id: 'snapshot-item', userId: 'test-user', primaryDivision: 'Project Engineer', employeeName: 'Archived Employee', employeeRole: 'Project Engineer', role: 'PE', workDivision: 'MK supervisi', work: 'Archived Project', number: 1, description: 'Archived report text', completedAt: '2026-09-03T00:00:00Z' };
 const masters = {divisions:[{id:1,name:'Project Engineer'},{id:2,name:'AI Dev'}],roles:[{id:1,divisionId:1,name:'PE',roleId:null},{id:2,divisionId:1,name:'QA',roleId:null}],categories:[{id:1,divisionId:1,roleId:1,name:'MK supervisi'},{id:2,divisionId:1,roleId:2,name:'QA1'},{id:3,divisionId:1,roleId:1,name:'Non-MK supervisi'}]};
 let jobs=[{id:1,divisionId:1,roleId:1,jobCategoryId:1,name:'Archived Project',description:'',isActive:true,sortOrder:0,division:'Project Engineer',role:'PE',category:'MK supervisi',referenced:true}];
 jobs.push({...jobs[0],id:3,jobCategoryId:3,category:'Non-MK supervisi',name:'Other category job',referenced:false}, {...jobs[0],id:4,roleId:2,role:'QA',jobCategoryId:2,category:'QA1',name:'QA job',referenced:false}, {...jobs[0],id:5,divisionId:2,division:'AI Dev',roleId:null,role:'',jobCategoryId:null,category:'',name:'Direct job',referenced:false}, {...jobs[0],id:6,name:'Second selectable job',referenced:false});
 const scenario={hold:false,release:null,failNext:false,posts:[],nextNumber:41};
 const picUsers=[{userId:'pic-a',name:'Jarwo',username:'jarwo',isActive:true},{userId:'pic-b',name:'Budi',username:'budi',isActive:true}];
 let jobMode=false;
 const user = { id: 'test-user', name: 'Browser Test', role: 'Management', roleId: 1, username: 'browser', jobTitle: 'Project Engineer', email: 'long.employee.address@example.test', isActive: true, createdAt: '2026-01-01T00:00:00Z', updatedAt: '2026-01-01T00:00:00Z', salary: 0, residence: '', phoneNumber: '', avatarUrl: '', employmentStatus: 'Permanent Employee' };
 socket.onmessage = async ({ data }) => {
  const event = JSON.parse(data);
  if (event.id) { const p = pending.get(event.id); pending.delete(event.id); event.error ? p.reject(event.error) : p.resolve(event.result); return; }
  if (event.method === 'Runtime.exceptionThrown') exceptions.push(event.params.exceptionDetails.exception?.description || event.params.exceptionDetails.text);
  if (event.method !== 'Fetch.requestPaused') return;
  const { requestId, request } = event.params, url = new URL(request.url);
  requests.push(url.pathname + url.search);
  let status=200;
  let payload = {}, contentType = 'application/json', headers = [];
  if (['/auth/me', '/me'].includes(url.pathname)) payload = { user };
  else if(url.pathname==='/jobs/pic-users') payload={users:picUsers.filter(u=>(u.name+' '+u.username).toLowerCase().includes((url.searchParams.get('q')??'').toLowerCase()))};
  else if(/\/jobs\/\d+\/pics/.test(url.pathname)) {
   const parts=url.pathname.split('/'),job=jobs.find(j=>j.id===Number(parts[2]));job.pics??=[];
   if(request.method==='POST'){const {userId}=JSON.parse(request.postData);if(!job.pics.some(p=>p.userId===userId))job.pics.push(picUsers.find(p=>p.userId===userId));}
   if(request.method==='DELETE')job.pics=job.pics.filter(p=>p.userId!==parts[4]);
   payload={pics:job.pics};
  }
  else if(url.pathname==='/jobs/masters') payload=masters;
  else if(url.pathname==='/jobs' && request.method==='POST') {
   const body=JSON.parse(request.postData); const job={...body,id:Math.max(0,...jobs.map(j=>j.id))+1,division:masters.divisions.find(d=>d.id===body.divisionId)?.name ?? '',role:masters.roles.find(r=>r.id===body.roleId)?.name ?? '',category:masters.categories.find(c=>c.id===body.jobCategoryId)?.name ?? '',referenced:false};jobs.push(job);payload={job};
  }
  else if(url.pathname==='/jobs') payload={jobs:jobs.filter(j=>(!url.searchParams.has('division_id')||j.divisionId===Number(url.searchParams.get('division_id'))) && (!url.searchParams.has('role_id')||(j.roleId??0)===Number(url.searchParams.get('role_id'))) && (!url.searchParams.has('job_category_id')||(j.jobCategoryId??0)===Number(url.searchParams.get('job_category_id'))) && (url.searchParams.get('active')!=='true'||j.isActive))};
  else if(url.pathname.startsWith('/jobs/')) {
   const id=Number(url.pathname.split('/')[2]), job=jobs.find(j=>j.id===id);
   if(request.method==='DELETE'){if(job.referenced){status=409;payload={error:'Pekerjaan sudah digunakan. Nonaktifkan pekerjaan ini.'};}else jobs=jobs.filter(j=>j.id!==id);}
   else if(['PATCH','PUT'].includes(request.method)){Object.assign(job,JSON.parse(request.postData));payload={job};}else payload={job};
  }
  else if (url.pathname === '/users') payload = { users: [user] };
  else if (url.pathname === '/roles') payload = { roles: [{ id: 1, name: 'Management' }] };
  else if (url.pathname === '/zoom-bookings') payload = { bookings: [] };
  else if (url.pathname === '/zoom-resources') payload = { resources: [1, 2, 3].map(id => ({ id, name: `Zoom ${id}`, code: `Z${id}`, meetingUrl: 'https://example.test/meeting', meetingId: '123456789', passcode: 'test' })) };
  else if (url.pathname === '/reports/history') payload = { archives: url.searchParams.get('page') === '2' ? [] : [archive], hasMore: url.searchParams.get('page') !== '2' };
  else if (url.pathname.endsWith('/pdf') || url.pathname === '/reports.pdf') { payload = '%PDF-1.4\n%%EOF'; contentType = 'application/pdf'; headers = [{ name: 'Content-Disposition', value: `attachment; filename="${archive.filename}"` }]; }
  else if (url.pathname === '/reports/history/history-fixture') payload = { archive: { ...archive, items: [item] } };
  else if (url.pathname === '/reports' && request.method==='POST') {
   const body=JSON.parse(request.postData);scenario.posts.push(body);
   if(scenario.hold)await new Promise(resolve=>scenario.release=resolve);
   if(scenario.failNext){scenario.failNext=false;status=503;payload={error:'Test save failure'};}
   else {status=201;payload={report:{...body,id:'created-'+scenario.nextNumber,number:scenario.nextNumber++}};}
  }
  else if (url.pathname === '/reports') payload = { reports: [{...item,pics:jobs.find(j=>j.id===1)?.pics??[]}], period: { start: archive.weekStart, end: '2026-09-05T17:00:00Z', label: '31 Agustus - 6 September 2026', readOnly: !jobMode && url.searchParams.get('primaryDivision') !== 'AI Dev' } };
  else if (url.pathname === '/notifications') payload = { notifications: [] };
  else payload = { unreadCount: 0 };
  await send('Fetch.fulfillRequest', { requestId, responseCode: status, responseHeaders: [{ name: 'Content-Type', value: contentType }, { name: 'Access-Control-Allow-Origin', value: 'http://127.0.0.1:5190' }, { name: 'Access-Control-Allow-Headers', value: 'Content-Type' }, { name: 'Access-Control-Allow-Methods', value: 'GET, POST, PUT, PATCH, DELETE, OPTIONS' }, { name: 'Access-Control-Allow-Credentials', value: 'true' }, { name: 'Access-Control-Expose-Headers', value: 'Content-Disposition' }, ...headers], body: Buffer.from(typeof payload === 'string' ? payload : JSON.stringify(payload)).toString('base64') });
 };
 await send('Page.enable'); await send('Runtime.enable');
 await send('Fetch.enable', { patterns: [{ urlPattern: '*://localhost:3000/*' }] });
 await send('Browser.setDownloadBehavior', { behavior: 'allow', downloadPath: profile });
 const evaluate = async expression => (await send('Runtime.evaluate', { expression, returnByValue: true })).result.value;
 const waitFor = async expression => { for (let i = 0; i < 100; i++) { if (await evaluate(expression)) return; await delay(100); } throw new Error(`Timed out: ${expression}\n${await evaluate('document.body.innerText')}\nRequests: ${requests.join(', ')}`); };
 const click = text => evaluate(`[...document.querySelectorAll('button')].find(b => b.textContent.trim() === ${JSON.stringify(text)}).click()`);
 if (process.env.TEST_LAYOUT) {
  const { testLayout } = await import('./responsive-layout.mjs');
  await testLayout({ send, evaluate, waitFor, click });
 }
 await send('Page.navigate', { url: 'http://127.0.0.1:5190/weekly-progress/history' });
 await waitFor("document.body.innerText.includes('31 Agu 2026') && document.body.innerText.includes('Download PDF')");
 await click('Detail'); await waitFor("document.body.innerText.includes('Archived Employee') && document.body.innerText.includes('Archived report text')");
 await click('Download PDF'); await waitFor("!document.body.innerText.includes('Mengunduh…')");
 for (let i = 0; i < 50 && !requests.some(p => p.endsWith('/pdf')); i++) await delay(100);
 assert.ok(requests.some(p => p.endsWith('/pdf')), 'download endpoint not requested');
 await evaluate("const date=document.querySelector('input[type=date]'); date.value='2026-09-03'; date.dispatchEvent(new Event('input',{bubbles:true})); const division=document.querySelector('select'); division.value='Project Engineer'; division.dispatchEvent(new Event('change',{bubbles:true}));");
 await click('Tampilkan'); await waitFor("!document.body.innerText.includes('Memuat riwayat')");
 assert.ok(requests.some(p => p.includes('date=2026-09-03') && p.includes('division=Project+Engineer')), 'filters missing from backend request');
 await click('Berikutnya'); await waitFor("document.body.innerText.includes('Halaman 2') && document.body.innerText.includes('Belum ada arsip')");
 await send('Page.navigate', { url: 'http://127.0.0.1:5190/weekly-progress?division=Project%20Engineer' });
 await waitFor("document.body.innerText.includes('hanya baca') && document.body.innerText.includes('Archived report text')");
 assert.equal(await evaluate("[...document.querySelectorAll('button,a')].some(e => ['Edit','Hapus','+ Input laporan'].includes(e.textContent.trim()))"), false, 'Monday exposes mutation controls');
 await send('Page.navigate', { url: 'http://127.0.0.1:5190/weekly-progress/new' });
 await waitFor("document.body.innerText.includes('Periode ditutup.')");
 assert.equal(await evaluate("document.querySelector('button[type=submit]').disabled"), true);
 jobMode=true;
 const {testWeeklyEntry}=await import('./weekly-entry.browser.mjs');
 await testWeeklyEntry({send,evaluate,waitFor,click,scenario});
 const {testJobSettings}=await import('./job-settings.browser.mjs');
 await testJobSettings({send,evaluate,waitFor,click,user,jobs,requests});
 assert.deepEqual(exceptions, [], 'browser runtime exceptions');
 console.log('PASS: history list, date/division filters, pagination, immutable detail, PDF download, Monday read-only, closed input');
} catch (error) { console.error(error); throw error; } finally {
 socket?.close(); chrome.kill();
 await delay(500);
 await rm(profile, { recursive: true, force: true, maxRetries: 5, retryDelay: 200 });
}
