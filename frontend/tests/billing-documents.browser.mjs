// Uses the existing isolated checklist browser fixture (3001) and Vite (5189).
import assert from 'node:assert/strict';
const {chromium}=await import(process.env.PLAYWRIGHT_MODULE??'../../graphify-out/responsive-tools/node_modules/playwright/index.mjs');
const browser=await chromium.launch({executablePath:process.env.CHROME_PATH??'C:/Program Files/Google/Chrome/Application/chrome.exe',headless:true,args:['--disable-gpu','--in-process-gpu']});
const context=await browser.newContext({viewport:{width:1440,height:1000}}),page=await context.newPage(),errors=[];
page.on('pageerror',e=>errors.push(e.message));
const origin='http://127.0.0.1:5189',backend='http://127.0.0.1:3001';
async function api(path,method='GET',data){const r=await context.request.fetch(backend+path,{method,data});assert.ok(r.ok(),await r.text());return r.json()}
async function login(role){await context.clearCookies();await page.goto(origin);await page.locator('#username').fill('browser-'+role);await page.locator('#password').fill('Checklist-test-only-2026');await page.getByRole('button',{name:/Masuk|Log in|Login/i}).click();await page.waitForURL('**/dashboard')}
async function save(){const response=page.waitForResponse(r=>r.url().includes('/billing-documents/stages')&&r.request().method()!=='GET');await page.getByRole('dialog').getByRole('button',{name:'Simpan',exact:true}).click();assert.ok((await response).ok());await page.getByRole('dialog').waitFor({state:'hidden'})}
try{
 await login('Admin');const masters=await api('/jobs/masters'),role=masters.roles.find(r=>r.name==='Project Engineer');
 const name='Billing browser '+Date.now();const {job}=await api('/jobs','POST',{divisionId:role.divisionId,roleId:role.id,name,isActive:true});
 await page.locator('.sidebar-nav').getByRole('link',{name:'Dokumen Penagihan',exact:true}).click();
 await page.getByRole('button',{name:'Tambah tahap penagihan',exact:true}).click();
 const dialog=page.getByRole('dialog');await dialog.getByRole('button',{name:'Tutup',exact:true}).click();await dialog.waitFor({state:'hidden'});await page.getByRole('button',{name:'Tambah tahap penagihan',exact:true}).click();await dialog.getByLabel('Pekerjaan',{exact:true}).selectOption(String(job.id));await dialog.getByLabel('Nama tahap',{exact:true}).fill('Tahap browser');assert.equal(await dialog.getByLabel('Periode penagihan').count(),0);await dialog.getByLabel('Nama sub-tahap',{exact:true}).fill('Sub-Tahap A');await dialog.getByLabel('Dokumen wajib 1',{exact:true}).fill('BAPP');await dialog.getByRole('button',{name:'Tambah sub-tahap',exact:true}).click();await dialog.getByLabel('Nama sub-tahap',{exact:true}).nth(1).fill('Sub-Tahap B');await dialog.getByLabel('Dokumen wajib 1',{exact:true}).nth(1).fill('Dokumen Khusus');await dialog.getByRole('button',{name:'Tambah dokumen tahap',exact:true}).click();await dialog.getByLabel('Dokumen tahap 1',{exact:true}).fill('BAST');await save();
 await page.getByRole('main').getByLabel('Pekerjaan',{exact:true}).selectOption(String(job.id));
 const stage=page.locator('tbody').filter({hasText:name});
 const status=doc=>stage.getByRole('combobox',{name:new RegExp(' / '+doc+'$')});
 async function setStatus(doc,value,expected=200){
  const response=page.waitForResponse(r=>r.url().includes('/billing-documents/stages/')&&r.request().method()==='PATCH');
  await status(doc).selectOption(value);assert.equal((await response).status(),expected);
  await page.waitForFunction(()=>!document.querySelector('.status-select:disabled'));
 }
 const viewer=await context.newPage();viewer.on('pageerror',e=>errors.push(e.message));
 await viewer.goto(origin+'/billing-documents');await viewer.getByRole('main').getByLabel('Pekerjaan',{exact:true}).selectOption(String(job.id));
 await viewer.locator('tbody .status-select').first().waitFor();
 await viewer.evaluate(()=>{window.billingEvents=0;window.addEventListener('billing-documents.changed',()=>window.billingEvents++)});
 assert.equal(await stage.locator('tr').count(),2);
 assert.equal(await status('BAST').count(),1);
 assert.equal(await status('BAST').locator('..').getAttribute('rowspan'),'2');
 assert.deepEqual(await status('BAST').locator('option').allTextContents(),['BELUM ADA','PEMBUATAN','KURANG LENGKAP','SIRKULIR TTD','OK']);
 assert.equal(await stage.locator('.readiness').textContent(),'BELUM SIAP');
 assert.ok(await stage.locator('.unused').count()>0);
 assert.ok(await page.getByRole('columnheader',{name:'Dokumen Khusus',exact:true}).count());
 for(const doc of ['BAPP','Dokumen Khusus','BAST'])await setStatus(doc,'OK');
 assert.equal(await stage.locator('.readiness').textContent(),'SIAP');
 await viewer.waitForFunction(()=>document.querySelector('tbody .readiness')?.textContent==='SIAP');
 assert.ok(await viewer.evaluate(()=>window.billingEvents>0),'real socket delivered billing event');
 // Edit a prefilled shared document using its full native dropdown, retaining ID/status.
 const savedStage=(await api('/billing-documents')).stages.find(s=>s.jobId===job.id);assert.equal(savedStage.periodLabel,'');const originalShared=savedStage.documents[0];
 await stage.getByRole('button',{name:'Edit requirement',exact:true}).click();
 assert.equal(await dialog.locator('fieldset.requirement > legend').last().textContent(),'Status Dokumen per Tahap');
 assert.match(await dialog.locator('fieldset.requirement > legend').first().textContent(),/^Status Dokumen per Sub-Tahap/);
 assert.deepEqual(await dialog.locator('fieldset > label').evaluateAll(els=>els.slice(0,3).map(el=>el.firstChild.textContent)),['Pekerjaan','Nama tahap','Nama sub-tahap']);
 const choices=dialog.getByLabel('Pilihan dokumen Dokumen tahap 1',{exact:true});
 assert.equal(await choices.locator('option').count(),10);
 await choices.selectOption('RAB Tagihan');await save();
 let renamed=(await api('/billing-documents')).stages.find(s=>s.jobId===job.id).documents[0];
 assert.equal(renamed.id,originalShared.id);assert.equal(renamed.status,'OK');assert.equal(renamed.name,'RAB Tagihan');
 await viewer.getByRole('combobox',{name:/ \/ RAB Tagihan$/}).waitFor();
 await stage.getByRole('button',{name:'Edit requirement',exact:true}).click();
 await choices.selectOption('BAST');await save();
 await viewer.close();
 await setStatus('BAPP','KURANG LENGKAP');
 assert.equal(await stage.locator('.readiness').textContent(),'BELUM SIAP');
 await stage.getByRole('button',{name:'Catatan sub-tahap Sub-Tahap A',exact:true}).click();
 await dialog.getByLabel('Catatan',{exact:true}).fill('Month note');await save();
 await stage.getByRole('button',{name:'Catatan tahap',exact:true}).click();await dialog.getByLabel('Catatan',{exact:true}).fill('Persistent note');await save();
 await page.reload();await page.getByRole('main').getByLabel('Pekerjaan',{exact:true}).selectOption(String(job.id));await stage.getByText('Persistent note',{exact:true}).waitFor();
 // Failed inline writes restore the saved status; retry is an explicit new selection.
 await page.route('**/billing-documents/stages/*',route=>route.fulfill({status:500,contentType:'application/json',body:JSON.stringify({error:'Dokumen penagihan tidak dapat diproses.'})}),{times:1});
 await setStatus('BAPP','PEMBUATAN',500);assert.equal(await status('BAPP').inputValue(),'KURANG LENGKAP');
 await setStatus('BAPP','PEMBUATAN');assert.equal(await status('BAPP').inputValue(),'PEMBUATAN');
 // A stale shared-document dropdown refreshes latest data without overwriting it.
 const before=(await api('/billing-documents')).stages.find(s=>s.jobId===job.id);
 await api('/billing-documents/stages/'+before.id,'PATCH',{version:before.version,documentId:before.documents[0].id,status:'SIRKULIR TTD'});
 await setStatus('BAST','BELUM ADA',409);
 await page.getByRole('alert').filter({hasText:/Periksa status terbaru/}).waitFor();
 assert.equal(await status('BAST').inputValue(),'SIRKULIR TTD');await setStatus('BAST','OK');
 // Conflicting updates preserve the draft and explicitly require comparing latest data.
 await stage.getByRole('button',{name:'Catatan tahap',exact:true}).click();await dialog.getByLabel('Catatan',{exact:true}).fill('Local draft');
 const current=(await api('/billing-documents')).stages.find(s=>s.jobId===job.id);await api('/billing-documents/stages/'+current.id,'PATCH',{version:current.version,notes:'Remote note'});
 await dialog.getByRole('button',{name:'Simpan',exact:true}).click();await dialog.getByText('Remote note',{exact:true}).waitFor();assert.equal(await dialog.getByLabel('Catatan',{exact:true}).inputValue(),'Local draft');await dialog.getByRole('button',{name:'Gunakan versi terbaru untuk draf ini',exact:true}).click();await save();
 await page.locator('.billing-scroll').evaluate(el=>el.scrollLeft=0);await page.screenshot({path:'../graphify-out/billing-table-shared.png',fullPage:true});
 // Removing a non-OK document/milestone recalculates readiness after confirmation.
 await stage.getByRole('button',{name:'Edit requirement',exact:true}).click();await dialog.getByRole('button',{name:'Hapus sub-tahap',exact:true}).first().click();await dialog.getByRole('button',{name:'Simpan',exact:true}).click();await dialog.getByRole('button',{name:'Konfirmasi dan simpan',exact:true}).click();await dialog.waitFor({state:'hidden'});assert.equal(await stage.locator('.readiness').textContent(),'SIAP');
 await page.getByLabel('Cari penagihan').fill('BAST');assert.equal(await stage.count(),1);await page.getByLabel('Kesiapan tahap').selectOption('BELUM SIAP');assert.equal(await stage.count(),0);await page.getByLabel('Kesiapan tahap').selectOption('');
 // Responsive table scroll is contained; controls remain reachable by keyboard.
 await page.getByLabel('Cari penagihan').fill('');
 for(const width of [320,768,1024,1440]){
  await page.setViewportSize({width,height:1000});await page.waitForTimeout(250);
  assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1));
  assert.ok(await page.locator('.billing-scroll').evaluate(el=>el.scrollWidth>el.clientWidth));
  await status('BAST').focus();await page.keyboard.press('ArrowUp');await page.keyboard.press('Escape');
  await page.waitForFunction(()=>!document.querySelector('.status-select:disabled'));
  await page.locator('.billing-scroll').evaluate(el=>el.scrollLeft=0);await page.screenshot({path:'../graphify-out/billing-table-'+width+'.png',fullPage:true});
  await stage.getByRole('button',{name:'Edit requirement',exact:true}).click();
  assert.ok(await dialog.evaluate(el=>el.getBoundingClientRect().right<=innerWidth&&el.scrollWidth<=el.clientWidth));
  await dialog.getByRole('button',{name:'Batal',exact:true}).click();
 }
 await page.emulateMedia({reducedMotion:'reduce'});await page.reload();
 await page.getByRole('main').getByLabel('Pekerjaan',{exact:true}).selectOption(String(job.id));
 assert.ok(await stage.isVisible());
 await login('HR');await page.goto(origin+'/billing-documents');await page.getByRole('alert').filter({hasText:'Akses penagihan ditolak.'}).waitFor();assert.equal((await context.request.get(backend+'/billing-documents')).status(),403);assert.deepEqual(errors,[]);
 console.log('PASS billing table: shared/local documents, readiness, notes/reload, failed save, conflicts, removal, filters, four viewport widths, reduced motion, permissions.');
}catch(e){console.error('Browser errors:',errors);console.error('Visible dialogs:',await page.getByRole('dialog').allTextContents());await page.screenshot({path:'../graphify-out/billing-failure.png',fullPage:true});throw e}finally{await browser.close()}
