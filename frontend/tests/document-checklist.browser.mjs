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
 await page.locator('#username').fill('browser-'+role);
 await page.locator('#password').fill('Checklist-test-only-2026');
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
async function configure(section, internal, external, digital, title = 'Table One') {
 await section.getByRole('button',{name:'+ Buat Tabel Checklist',exact:true}).click();
 const dialog=page.getByRole('dialog');
 await dialog.getByLabel('Judul Tabel',{exact:true}).fill(title);
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
 await saved(()=>section.getByRole('button',{name:'+ Tambah Isi Dokumen',exact:true}).click());
}
try {
 await login('Admin');
 const masters=await api('/jobs/masters');
 const stamp=Date.now();const names=['Checklist Browser A '+stamp,'Checklist Browser B '+stamp];
 const jobs=[];
 const pe=masters.roles.find(r=>r.name==='Project Engineer');
 assert.ok(pe);
 for(const role of masters.roles.filter(r=>['Quality Assurance','CIHPK'].includes(r.name))) {
  await api('/jobs','POST',{divisionId:role.divisionId,roleId:role.id,name:'Excluded '+role.name+' '+stamp,isActive:true});
 }
 for(const name of names){const result=await api('/jobs','POST',{divisionId:pe.divisionId,roleId:pe.id,name,isActive:true});jobs.push(result.job)}
 await page.locator('.sidebar-nav').getByRole('link',{name:'Checklist Dokumen Pekerjaan',exact:true}).click();
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
 assert.equal(await a.getByRole('columnheader',{name:'TTD Hardcopy',exact:true}).count(),1);
 assert.equal(await a.getByRole('columnheader',{name:'Aksi',exact:true}).count(),0);
 assert.equal(await a.getByRole('button',{name:/Hapus/}).count(),0);
 assert.equal(await a.getByRole('button',{name:'Tabel sebelumnya',exact:true}).isDisabled(),true);
 assert.equal(await a.getByRole('button',{name:'Tabel berikutnya',exact:true}).isDisabled(),true);
 await configure(a,[],[],false,'Table Two');
 await add(a,'Second table document');
 assert.equal(await a.locator('.table-title').textContent(),'Table Two');
 assert.equal(await a.getByRole('button',{name:'Tabel berikutnya',exact:true}).isDisabled(),true);
 await a.getByRole('button',{name:'Tabel sebelumnya',exact:true}).click();
 await a.getByLabel('Dokumen 1',{exact:true}).waitFor();
 assert.equal(await a.getByLabel('Dokumen 1',{exact:true}).inputValue(),'Berita Acara');
 // Clicking navigation saves the focused cell before changing tables.
 await a.getByLabel('Catatan Berita Acara',{exact:true}).fill('Saved before navigation');
 await saved(()=>a.getByRole('button',{name:'Tabel berikutnya',exact:true}).click());
 await page.waitForFunction(()=>document.querySelector('.table-title')?.textContent==='Table Two');
 await a.getByRole('button',{name:'Tabel sebelumnya',exact:true}).click();
 assert.equal(await a.getByLabel('Catatan Berita Acara',{exact:true}).inputValue(),'Saved before navigation');
 // A failed save must not navigate to another table.
 await page.route('**/document-checklists/*/tables/*/items/*', route => route.fulfill({status:500,contentType:'application/json',body:JSON.stringify({error:'Terjadi kesalahan pada server.'})}), {times:1});
 await a.getByLabel('Catatan Berita Acara',{exact:true}).fill('Must not switch');
 const failure=page.waitForResponse(r=>r.status()===500 && r.url().includes('/document-checklists/'));
 await a.getByRole('button',{name:'Tabel berikutnya',exact:true}).click();await failure;
 await a.getByRole('alert').filter({hasText:'Terjadi kesalahan pada server.'}).waitFor();
 assert.equal(await a.locator('.table-title').textContent(),'Table One');
 await saved(async()=>{const field=a.getByLabel('Catatan Berita Acara',{exact:true});await field.fill('Menunggu tanda tangan client');await field.press('Tab')});
 const tables=(await api('/document-checklists/'+jobs[0].id)).checklists;
 assert.equal(tables.length,2);assert.equal(tables[0].items.length,3);assert.equal(tables[1].items.length,1);
 const downloading=page.waitForEvent('download');
 await a.getByRole('button',{name:'Ekspor PDF pekerjaan',exact:true}).click();
 const pdf=await downloading;
 assert.equal(pdf.suggestedFilename(),'Checklist_Job_'+jobs[0].id+'.pdf');
 assert.equal(await pdf.failure(),null);
 for(const job of (await api('/document-checklists')).jobs)assert.equal(job.role,'Project Engineer');
 assert.equal(await a.locator('.table-title').getAttribute('colspan'),'9');
 await a.getByRole('button',{name:'Tabel berikutnya',exact:true}).click();
 assert.equal(await a.locator('.table-title').getAttribute('colspan'),'5');
 const draft=a.getByLabel('Catatan Second table document',{exact:true});
 await draft.fill('Typing while another table changes');
 await api('/document-checklists/'+jobs[0].id,'POST',{title:'Remote table',version:0,internalCount:0,externalCount:0,signers:[]});
 assert.equal(await draft.inputValue(),'Typing while another table changes');
 await saved(()=>draft.press('Tab'));
 await a.getByText('Table Two (2 / 3)',{exact:true}).waitFor();
 assert.equal(await draft.inputValue(),'Typing while another table changes');
 await a.getByRole('button',{name:'Tabel sebelumnya',exact:true}).click();
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
 await a.getByRole('button',{name:'Edit Tabel Checklist',exact:true}).click();
 await page.getByRole('dialog').getByLabel('Nama',{exact:true}).first().fill('Budi Pengganti');
 await page.getByRole('dialog').getByLabel('Judul Tabel',{exact:true}).fill('Renamed table');
 await saved(()=>page.getByRole('dialog').getByRole('button',{name:'Simpan',exact:true}).click());
 assert.equal(await a.locator('.table-title').textContent(),'Renamed table');
 assert.equal(await a.getByLabel('Budi Pengganti — Berita Acara',{exact:true}).isChecked(),false);
 await a.getByText('Riwayat penandatangan yang diarsipkan (1)',{exact:true}).click();
 assert.match(await a.locator('details').textContent(),/Berita Acara: Sudah ditandatangani/);
 for(const width of [320,768,1024,1440]){
  await page.setViewportSize({width,height:1000});
  const positions=()=>a.locator('.table-navigation').evaluate(el=>{const r=el.getBoundingClientRect(),p=el.parentElement.getBoundingClientRect();return {center:r.x+r.width/2,parent:p.x+p.width/2,arrows:[...el.querySelectorAll('button')].map(b=>b.getBoundingClientRect().x)}});
  const before=await positions();assert.ok(Math.abs(before.center-before.parent)<1);
  await a.getByRole('button',{name:'Tabel berikutnya',exact:true}).click();
  await a.locator('.table-title').filter({hasText:'Table Two'}).waitFor();
  assert.deepEqual((await positions()).arrows,before.arrows);
  await a.getByRole('button',{name:'Tabel sebelumnya',exact:true}).click();
  await a.locator('.table-title').filter({hasText:'Renamed table'}).waitFor();
  assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1));
  await page.screenshot({path:'../graphify-out/checklist-centered-'+width+'.png',fullPage:true});
 }
 for(const role of ['Project Engineer','Finance','Management']) {
  await login(role);await page.locator('.sidebar-nav').getByRole('link',{name:'Checklist Dokumen Pekerjaan',exact:true}).click();
  await a.getByRole('button',{name:new RegExp(names[0])}).waitFor();
 }
 await page.locator('.sidebar-nav').getByRole('link',{name:'Booking Zoom',exact:true}).click();
 await page.locator('.calendar-lane').first().waitFor();
 assert.equal(await page.locator('.calendar-day-head').count(),3);
 assert.equal(await page.locator('.calendar-lane').count(),9);
 const firstPeriod=await page.locator('#week-title').textContent();
 await page.getByRole('button',{name:'3 hari berikutnya',exact:true}).click();
 await page.waitForFunction(previous=>document.querySelector('#week-title')?.textContent!==previous,firstPeriod);
 await page.getByRole('button',{name:'3 hari sebelumnya',exact:true}).click();
 await page.waitForFunction(previous=>document.querySelector('#week-title')?.textContent===previous,firstPeriod);
 await page.getByRole('link',{name:/Kembali ke dashboard/i}).click();
 await page.waitForURL('**/dashboard');
 await page.reload();
 assert.equal(await page.locator('.page-back').count(),0);
 await login('HR');assert.equal(await page.locator('.sidebar-nav').getByRole('link',{name:'Checklist Dokumen Pekerjaan',exact:true}).count(),0);
 await page.goto(origin+'/document-checklists');await page.getByRole('alert').filter({hasText:'Akses ditolak'}).waitFor();
 const denied=await context.request.get(backend+'/document-checklists');assert.equal(denied.status(),403);
 assert.deepEqual(errors,[]);
 console.log('PASS: real login, DB jobs, independent configurations, dynamic headers, multiple named tables, navigation, save failure, inline editing, persistent cells/signatures, archive, stable IDs/numbering, mobile scrolling, allowed roles and forbidden direct URL/API.');
} finally {await browser.close();}
