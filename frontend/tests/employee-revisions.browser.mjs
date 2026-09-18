// Run TestAttendanceBrowserServer with its guarded local database and Vite on port 5189.
import assert from 'node:assert/strict';
const {chromium}=await import(process.env.PLAYWRIGHT_MODULE ?? '../../graphify-out/responsive-tools/node_modules/playwright/index.mjs');
const browser=await chromium.launch({executablePath:process.env.CHROME_PATH ?? 'C:/Program Files/Google/Chrome/Application/chrome.exe',headless:true});
const origin='http://127.0.0.1:5189',backend='http://127.0.0.1:3001';
const context=await browser.newContext({viewport:{width:1440,height:1000},reducedMotion:'reduce',geolocation:{latitude:0,longitude:0,accuracy:5},permissions:['geolocation']});
const page=await context.newPage(),errors=[];
page.on('pageerror',e=>errors.push(e.message));
page.setDefaultTimeout(30000);
await context.route('https://**/*',route=>route.abort());
async function history(){return (await context.request.get(backend+'/attendance')).json()}
try {
 assert.equal((await context.request.post(backend+'/auth/login',{data:{username:'attendance-browser-Admin',password:'Checklist-test-only-2026'}})).status(),200);
 await page.goto(origin+'/attendance',{waitUntil:'domcontentloaded'});
 await page.getByRole('button',{name:/^WFA /}).click();
 const notes='Kunjungan lapangan\nTim A';
 await page.getByLabel('Catatan (opsional)').fill('  '+notes+'  ');
 await page.getByRole('button',{name:/^(Masuk kerja|Clock In)$/}).click();
 await page.getByRole('button',{name:/^(Pulang kerja|Clock Out)$/}).waitFor();
 assert.equal((await history()).today.notes,notes);
 await page.getByRole('cell',{name:notes,exact:true}).waitFor();
 await page.getByRole('button',{name:/^(Pulang kerja|Clock Out)$/}).click();
 await page.getByText('Absensi hari ini selesai.',{exact:true}).waitFor();
 assert.equal((await history()).today.notes,notes);
 for(const width of [320,1440]){
  await page.setViewportSize({width,height:1000});
  assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true,'attendance overflow '+width);
  await page.screenshot({path:`../graphify-out/revisions-attendance-${width}.png`,fullPage:true});
 }
 await page.goto(origin+'/attendance/overview',{waitUntil:'domcontentloaded'});
 await page.getByRole('cell',{name:notes,exact:true}).waitFor();
 console.log('Attendance notes persisted and visible in history and overview.');
 await page.goto(origin + '/settings/jobs', {waitUntil: 'domcontentloaded'});
 const masters = await (await context.request.get(backend + '/jobs/masters')).json();
 const division = masters.divisions.find(d => d.name === 'AI Dev');
 await page.getByLabel(/^Divisi/).selectOption(String(division.id));
 await page.getByRole('button', { name: 'Tambah Kategori', exact: true }).click();
 const categoryName = 'Browser category ' + Date.now();
 await page.getByLabel('Nama kategori', { exact: true }).fill(categoryName);
 await page.getByRole('button', { name: 'Simpan kategori', exact: true }).click();
 await page.getByText('Kategori berhasil ditambahkan.', { exact: true }).waitFor();
 assert.equal(await page.getByLabel(/^Kategori/).locator('option:checked').innerText(), categoryName);
 await page.getByRole('button', { name: '+ Tambah Pekerjaan', exact: true }).click();
 await page.getByLabel('Nama pekerjaan', { exact: true }).fill(categoryName + ' job');
 await page.getByRole('button', { name: 'Simpan pekerjaan', exact: true }).click();
 await page.getByRole('cell', { name: categoryName, exact: true }).waitFor();
 await page.reload({waitUntil: 'domcontentloaded'});
 await page.getByLabel(/^Divisi/).selectOption(String(division.id));
 await page.getByLabel(/^Kategori/).selectOption({label: categoryName});
 await page.getByRole('cell', { name: categoryName, exact: true }).waitFor();
 const pe = masters.divisions.find(d => d.name === 'Project Engineer');
 const cihpk = masters.roles.find(r => r.name === 'CIHPK');
 await page.getByLabel(/^Divisi/).selectOption(String(pe.id));
 await page.getByLabel(/^(Role|Peran)/).selectOption(String(cihpk.id));
 assert.equal(await page.getByRole('button', { name: 'Tambah Kategori', exact: true }).isDisabled(), true);
 assert.equal((await context.request.post(backend+'/jobs/categories',{data:{divisionId:pe.id,roleId:cihpk.id,name:'Rejected'}})).status(),400);
 for (const width of [320,1440]) {
  await page.setViewportSize({width,height:1000});
  assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth),true,'jobs overflow '+width);
 }
 const users = (await (await context.request.get(backend+'/users')).json()).users;
 const admin = users.find(u=>u.username==='attendance-browser-Admin');
 assert.equal((await context.request.patch(backend+'/users/'+admin.id,{data:{...admin,salary:1234567}})).status(),200);
 await page.goto(origin+'/my-profile', {waitUntil: 'domcontentloaded'});
 await page.getByRole('heading', {name: /Profil Saya|My Profile/}).waitFor();
 assert.equal(await page.getByText(/^(Salary|Gaji)$/).count(),0);
 await page.goto(origin+'/employees', {waitUntil: 'domcontentloaded'});
 await page.getByRole('row').filter({hasText:'attendance-browser-Admin'}).getByRole('button',{name:/^(Edit|Ubah)$/}).click();
 assert.equal(await page.getByLabel(/^(Salary|Gaji)$/).count(),0);
 await page.getByRole('button',{name:/^(Simpan|Save)$/}).click();
 await page.getByRole('row').filter({hasText:'attendance-browser-Admin'}).waitFor();
 const updated = (await (await context.request.get(backend+'/users')).json()).users.find(u=>u.id===admin.id);
 assert.equal(updated.salary,1234567);
 assert.equal(await page.getByRole('columnheader',{name:/^(Salary|Gaji)$/}).count(),0);

 assert.deepEqual(errors,[]);
 console.log('Passed: attendance notes, categories, CIHPK guard, hidden salary preservation, mobile and desktop.');
}catch(error){console.error(page.url(),await page.locator('body').innerText());throw error}
finally{await browser.close()}
