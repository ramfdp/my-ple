// Real Svelte -> cookie session -> Go -> isolated PostgreSQL acceptance test.
// Start TestDocumentChecklistBrowserServer and Vite on 5189 (VITE_API_URL=http://127.0.0.1:3001).
import assert from 'node:assert/strict';
const {chromium} = await import(process.env.PLAYWRIGHT_MODULE ?? '../../graphify-out/responsive-tools/node_modules/playwright/index.mjs');
const origin = 'http://127.0.0.1:5189', backend = 'http://127.0.0.1:3001';
const browser = await chromium.launch({executablePath:process.env.CHROME_PATH ?? 'C:/Program Files/Google/Chrome/Application/chrome.exe',headless:true,args:['--disable-gpu','--in-process-gpu']});
const context = await browser.newContext({viewport:{width:1440,height:1000}});
const page = await context.newPage(), errors=[];
page.on('pageerror', e => errors.push(e.message));
page.on('dialog', d => d.accept());
async function login(role) {
 await context.clearCookies(); await page.goto(origin);
 await page.getByLabel('Username',{exact:true}).fill('browser-'+role);
 await page.getByLabel('Password',{exact:true}).fill('Checklist-test-only-2026');
 await page.getByRole('button',{name:/Masuk|Log in|Login/i}).click();
 await page.waitForURL('**/dashboard');
}
async function api(path, method='GET', data) {
 const res = await context.request.fetch(backend+path,{method,data});
 assert.equal(res.ok(),true,await res.text());return res.json();
}
async function saved(action) {
 const response = page.waitForResponse(r => r.url().includes('/document-checklists/') && ['POST','PUT','PATCH','DELETE'].includes(r.request().method()));
 await action(); const result=await response; assert.equal(result.ok(),true,await result.text());
 await page.waitForFunction(() => !document.querySelector('.checklist input:disabled'));
}
async function configure(section, internal, external, digital) {
 await section.getByRole('button',{name:'+ Buat Tabel Checklist',exact:true}).click();
 const dialog=page.getByRole('dialog');
 if(digital)await dialog.getByLabel('Butuh TTD Digital?').check();
 await dialog.getByLabel('Berapa banyak TTD Internal?').fill(String(internal.length));
 await dialog.getByLabel('Berapa banyak TTD External?').fill(String(external.length));
 for(const [i,s] of [...internal,...external].entries()) {
  await dialog.getByLabel('Nama',{exact:true}).nth(i).fill(s[0]);
  await dialog.getByLabel('Sebagai',{exact:true}).nth(i).fill(s[1]);
 }
 await saved(()=>dialog.getByRole('button',{name:'Simpan',exact:true}).click());
 assert.equal(await dialog.count(),0);
}
async function add(section,name) {
 await section.getByLabel('Dokumen baru',{exact:true}).fill(name);
 await saved(()=>section.getByRole('button',{name:'+ Tambah Dokumen',exact:true}).click());
}
try {
 await login('Admin');
 const masters=await api('/jobs/masters');
 const stamp=Date.now();const names=['Checklist Browser A '+stamp,'Checklist Browser B '+stamp];
 const jobs=[];
 for(const name of names){const result=await api('/jobs','POST',{divisionId:masters.divisions[0].id,name,isActive:true});jobs.push(result.job)}
 await page.getByRole('link',{name:'Checklist Dokumen Pekerjaan',exact:true}).click();
 const a=page.locator('section').filter({has:page.getByRole('button',{name:new RegExp(names[0])})});
 const b=page.locator('section').filter({has:page.getByRole('button',{name:new RegExp(names[1])})});
 await a.getByRole('button',{name:new RegExp(names[0])}).click();
 await configure(a,[['Budi','Project Manager'],['Andi','Project Engineer']],[['Hendrik','Client']],true);
 await add(a,'Berita Acara');
 await saved(()=>a.getByLabel('TTD Digital Berita Acara',{exact:true}).check());
 await saved(()=>a.getByLabel('Budi — Berita Acara',{exact:true}).check());
 await saved(()=>a.getByLabel('Andi — Berita Acara',{exact:true}).check());
 for(const [label,value] of [['Posisi Berita Acara','Di Client'],['Tanggal kirim Berita Acara','2026-09-14'],['Catatan Berita Acara','Menunggu tanda tangan client']]) {
  await saved(async()=>{const field=a.getByLabel(label,{exact:true});await field.fill(value);await field.press('Tab')});
 }
 await add(a,'Laporan Mingguan');await add(a,'Laporan Final');
 const before=(await api('/document-checklists/'+jobs[0].id)).checklist.items;
 await saved(()=>a.getByRole('button',{name:'Hapus Laporan Mingguan',exact:true}).click());
 assert.deepEqual(await a.locator('tbody tr td:first-child').allTextContents(),['1','2']);
 const after=(await api('/document-checklists/'+jobs[0].id)).checklist.items;
 assert.deepEqual(after.map(i=>i.id),[before[0].id,before[2].id]);
 await b.getByRole('button',{name:new RegExp(names[1])}).click();
 await configure(b,[['Internal B','PE']],[['External B1','Client'],['External B2','Client']],false);
 await add(b,'Dokumen B');assert.equal(await b.getByText('TTD Digital',{exact:true}).count(),0);
 await page.reload();
 await a.getByRole('button',{name:new RegExp(names[0])}).click();
 await a.getByLabel('TTD Digital Berita Acara',{exact:true}).waitFor();
 for(const label of ['TTD Digital Berita Acara','Budi — Berita Acara','Andi — Berita Acara'])assert.equal(await a.getByLabel(label,{exact:true}).isChecked(),true);
 assert.equal(await a.getByLabel('Hendrik — Berita Acara',{exact:true}).isChecked(),false);
 assert.equal(await a.getByLabel('Posisi Berita Acara',{exact:true}).inputValue(),'Di Client');
 assert.equal(await a.getByLabel('Tanggal kirim Berita Acara',{exact:true}).inputValue(),'2026-09-14');
 assert.equal(await a.getByLabel('Catatan Berita Acara',{exact:true}).inputValue(),'Menunggu tanda tangan client');
 await b.getByRole('button',{name:new RegExp(names[1])}).click();
 await b.getByLabel('External B1 — Dokumen B',{exact:true}).waitFor();
 assert.equal(await b.getByText('Budi',{exact:true}).count(),0);
 await a.getByRole('button',{name:'Edit Struktur Checklist',exact:true}).click();
 await page.getByRole('dialog').getByLabel('Nama',{exact:true}).first().fill('Budi Pengganti');
 await saved(()=>page.getByRole('dialog').getByRole('button',{name:'Simpan',exact:true}).click());
 assert.equal(await a.getByLabel('Budi Pengganti — Berita Acara',{exact:true}).isChecked(),false);
 await a.getByText('Riwayat penandatangan yang diarsipkan (1)',{exact:true}).click();
 assert.match(await a.locator('details').textContent(),/Berita Acara: Sudah ditandatangani/);
 await page.screenshot({path:'../graphify-out/checklist-desktop.png',fullPage:true});
 await page.setViewportSize({width:390,height:844});
 assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true,'page overflows mobile');
 assert.equal(await a.locator('.table-scroll').evaluate(el=>el.scrollWidth>el.clientWidth),true,'wide table must scroll');
 await page.screenshot({path:'../graphify-out/checklist-mobile.png',fullPage:true});
 await page.setViewportSize({width:1440,height:1000});
 for(const role of ['Project Engineer','Finance','Management']) {
  await login(role);await page.getByRole('link',{name:'Checklist Dokumen Pekerjaan',exact:true}).click();
  await a.getByRole('button',{name:new RegExp(names[0])}).waitFor();
 }
 await login('HR');assert.equal(await page.getByRole('link',{name:'Checklist Dokumen Pekerjaan',exact:true}).count(),0);
 await page.goto(origin+'/document-checklists');await page.getByRole('alert').filter({hasText:'Akses ditolak'}).waitFor();
 const denied=await context.request.get(backend+'/document-checklists');assert.equal(denied.status(),403);
 assert.deepEqual(errors,[]);
 console.log('PASS: real login, DB jobs, independent configurations, dynamic headers, document CRUD, persistent cells/signatures, archive, stable IDs/numbering, mobile scrolling, allowed roles and forbidden direct URL/API.');
} finally {await browser.close();}
