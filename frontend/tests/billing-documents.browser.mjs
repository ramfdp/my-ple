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
 const dialog=page.getByRole('dialog');await dialog.getByLabel('Pekerjaan',{exact:true}).selectOption(String(job.id));await dialog.getByLabel('Nama tahap',{exact:true}).fill('Tahap browser');await dialog.getByLabel('Periode penagihan').fill('TW-1');await dialog.getByLabel('Nama milestone',{exact:true}).fill('Milestone A');await dialog.getByLabel('Dokumen wajib 1',{exact:true}).fill('BAPP');await dialog.getByRole('button',{name:'Tambah milestone',exact:true}).click();await dialog.getByLabel('Nama milestone',{exact:true}).nth(1).fill('Milestone B');await dialog.getByLabel('Dokumen wajib 1',{exact:true}).nth(1).fill('BAST');await save();
 await page.getByRole('main').getByLabel('Pekerjaan',{exact:true}).selectOption(String(job.id));
 const stage=page.locator('article.stage');assert.equal(await stage.locator('.readiness').filter({hasText:'BELUM SIAP'}).count(),3);
 for(const doc of ['BAPP','BAST']){await stage.getByRole('button',{name:'Ubah status '+doc,exact:true}).click();await dialog.getByLabel('Status dokumen').selectOption('OK');await save()}
 assert.deepEqual(await stage.locator('.readiness').allTextContents(),['SIAP','SIAP','SIAP']);
 await stage.getByRole('button',{name:'Ubah status BAPP',exact:true}).click();await dialog.getByLabel('Status dokumen').selectOption('KURANG LENGKAP');await save();assert.deepEqual(await stage.locator('.readiness').allTextContents(),['BELUM SIAP','BELUM SIAP','SIAP']);
 await stage.getByRole('button',{name:'Catatan tahap',exact:true}).click();await dialog.getByLabel('Catatan',{exact:true}).fill('Persistent note');await save();
 await page.reload();await page.getByRole('main').getByLabel('Pekerjaan',{exact:true}).selectOption(String(job.id));await stage.getByText('Persistent note',{exact:true}).waitFor();
 // Failed writes preserve the draft and the saved readiness.
 await stage.getByRole('button',{name:'Ubah status BAPP',exact:true}).click();await dialog.getByLabel('Status dokumen').selectOption('PEMBUATAN');
 await page.route('**/billing-documents/stages/*',route=>route.fulfill({status:500,contentType:'application/json',body:JSON.stringify({error:'Dokumen penagihan tidak dapat diproses.'})}),{times:1});
 await dialog.getByRole('button',{name:'Simpan',exact:true}).click();await dialog.getByRole('alert').waitFor();assert.equal(await dialog.getByLabel('Status dokumen').inputValue(),'PEMBUATAN');await save();
 // Conflicting updates preserve the draft and explicitly require comparing latest data.
 await stage.getByRole('button',{name:'Catatan tahap',exact:true}).click();await dialog.getByLabel('Catatan',{exact:true}).fill('Local draft');
 const current=(await api('/billing-documents')).stages.find(s=>s.jobId===job.id);await api('/billing-documents/stages/'+current.id,'PATCH',{version:current.version,notes:'Remote note'});
 await dialog.getByRole('button',{name:'Simpan',exact:true}).click();await dialog.getByText('Remote note',{exact:true}).waitFor();assert.equal(await dialog.getByLabel('Catatan',{exact:true}).inputValue(),'Local draft');await dialog.getByRole('button',{name:'Gunakan versi terbaru untuk draf ini',exact:true}).click();await save();
 // Removing a non-OK document/milestone recalculates readiness after confirmation.
 await stage.getByRole('button',{name:'Edit requirement',exact:true}).click();await dialog.getByRole('button',{name:'Hapus milestone',exact:true}).first().click();await dialog.getByRole('button',{name:'Simpan',exact:true}).click();await dialog.getByRole('button',{name:'Konfirmasi dan simpan',exact:true}).click();await dialog.waitFor({state:'hidden'});assert.deepEqual(await stage.locator('.readiness').allTextContents(),['SIAP','SIAP']);
 await page.getByLabel('Cari penagihan').fill('BAST');assert.equal(await stage.count(),1);await page.getByLabel('Kesiapan tahap').selectOption('BELUM SIAP');assert.equal(await stage.count(),0);await page.getByLabel('Kesiapan tahap').selectOption('');
 await page.screenshot({path:'../graphify-out/billing-desktop.png',fullPage:true});
 await page.setViewportSize({width:464,height:900});await page.waitForFunction(()=>document.querySelector('.sidebar-container').getBoundingClientRect().right<=1);assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));await page.screenshot({path:'../graphify-out/billing-464.png',fullPage:true});
 await stage.getByRole('button',{name:'Edit requirement',exact:true}).click();assert.ok(await dialog.evaluate(el=>el.getBoundingClientRect().right<=innerWidth&&el.scrollWidth<=el.clientWidth));await dialog.getByRole('button',{name:'Batal',exact:true}).click();
 await login('HR');await page.goto(origin+'/billing-documents');await page.getByRole('alert').filter({hasText:'Akses penagihan ditolak.'}).waitFor();assert.equal((await context.request.get(backend+'/billing-documents')).status(),403);assert.deepEqual(errors,[]);
 console.log('PASS billing: create, readiness propagation, notes/reload, failed save, conflict, requirement removal, filters, desktop/464px, permissions.');
}catch(e){console.error('Browser errors:',errors);console.error('Visible dialogs:',await page.getByRole('dialog').allTextContents());await page.screenshot({path:'../graphify-out/billing-failure.png',fullPage:true});throw e}finally{await browser.close()}
