// Use TestDocumentChecklistBrowserServer on 3001 and Vite on 5189.
import assert from 'node:assert/strict';
const {chromium}=await import(process.env.PLAYWRIGHT_MODULE ?? '../../graphify-out/responsive-tools/node_modules/playwright/index.mjs');
const origin='http://127.0.0.1:5189',backend='http://127.0.0.1:3001';
const browser=await chromium.launch({executablePath:process.env.CHROME_PATH??'C:/Program Files/Google/Chrome/Application/chrome.exe',headless:true,args:['--disable-gpu','--in-process-gpu']});
const context=await browser.newContext({viewport:{width:1440,height:1000}}),page=await context.newPage(),errors=[];
page.on('pageerror',error=>errors.push(error.message));
async function login(role){
 await context.clearCookies();await page.goto(origin);await page.locator('#username').fill('browser-'+role);await page.locator('#password').fill('Checklist-test-only-2026');await page.locator('button[type=submit]').click();await page.waitForURL('**/dashboard');await page.locator('.menu-card').first().waitFor();
}
async function language(value){
 const done=page.waitForResponse(r=>r.url().endsWith('/profile/language')&&r.request().method()==='PATCH');
 await page.locator('.dashboard-actions select').selectOption(value);assert.equal((await done).status(),200);await page.waitForFunction(lang=>document.documentElement.lang===lang,value);
}
async function parity(){
 const cards=await page.locator('.menu-card').evaluateAll(nodes=>nodes.map(n=>[n.getAttribute('href'),n.querySelector('h3').textContent]).sort());
 const nav=await page.locator('.sidebar-nav a').evaluateAll(nodes=>nodes.filter(n=>n.getAttribute('href')!=='/dashboard').map(n=>[n.getAttribute('href'),n.querySelector('.nav-text').textContent]).sort());assert.deepEqual(cards,nav);
 return cards.map(([href])=>href);
}
try{
 await login('Admin');await language('en');const routes=await parity();
 await page.reload();await page.locator('.menu-card').first().waitFor();assert.equal(await page.locator('html').getAttribute('lang'),'en');
 // Another browser must get the account language, not this browser's local cache.
 const other=await browser.newContext(),peer=await other.newPage();
 await peer.goto(origin);await peer.locator('#username').fill('browser-Admin');await peer.locator('#password').fill('Checklist-test-only-2026');await peer.locator('button[type=submit]').click();await peer.waitForURL('**/dashboard');await peer.getByRole('heading',{name:'Main menu',exact:true}).waitFor();
 for(const lang of ['id','en']){
  await page.goto(origin+'/dashboard');await language(lang);await parity();
  await peer.waitForFunction(value=>document.documentElement.lang===value,lang);
  for(const route of [...routes,'/weekly-progress/new','/weekly-progress/history']){
   await page.goto(origin+route);await page.locator('main').waitFor();assert.equal(await page.locator('html').getAttribute('lang'),lang);
  }
  await page.goto(origin+'/booking-zoom');await page.locator('.zoom-info-item').first().waitFor();await page.locator('.zoom-calendar').waitFor();
  const sizes=await page.evaluate(()=>{const a=document.querySelector('.schedule-card').getBoundingClientRect(),b=document.querySelector('.zoom-info-panel').getBoundingClientRect();return [a.top,a.bottom,b.top,b.bottom]});
  assert.ok(Math.abs(sizes[0]-sizes[2])<2&&Math.abs(sizes[1]-sizes[3])<2,JSON.stringify(sizes));
  assert.ok(await page.locator('.calendar-scroll').evaluate(el=>el.scrollHeight>el.clientHeight));
  await page.screenshot({path:'../graphify-out/zoom-'+lang+'.png',fullPage:true});
 }
 // A live language update must preserve an open form's user-entered values.
 await peer.goto(origin+'/booking-zoom');await peer.getByRole('button',{name:'+ Create Booking',exact:true}).click();
 const title=peer.locator('.zoom-modal input').first();
 await title.fill('Catatan pengguna');await peer.locator('.zoom-modal textarea').fill('Isi asli tetap sama.');
 await page.goto(origin+'/dashboard');await language('id');
 await peer.waitForFunction(()=>document.documentElement.lang==='id');
 assert.equal(await title.inputValue(),'Catatan pengguna');assert.equal(await peer.locator('.zoom-modal textarea').inputValue(),'Isi asli tetap sama.');
 await peer.getByRole('button',{name:'Simpan booking',exact:true}).waitFor();
 await other.close();
 await page.setViewportSize({width:390,height:844});await page.goto(origin+'/booking-zoom');await page.locator('.zoom-info-item').first().waitFor();assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true);
 await page.setViewportSize({width:1440,height:1000});
 for(const role of ['Project Engineer','Finance','Management','HR']){await login(role);await language('id');await parity();}
 await login('Admin');await language('id');
 // Failed preference saves must leave the current language and selection intact.
 await page.route('**/profile/language',route=>route.fulfill({status:500,contentType:'application/json',body:'{"error":"failure"}'}),{times:1});
 await page.locator('.dashboard-actions select').selectOption('en');await page.getByRole('alert').filter({hasText:'Gagal menyimpan bahasa.'}).waitFor();assert.equal(await page.locator('html').getAttribute('lang'),'id');assert.equal(await page.locator('.dashboard-actions select').inputValue(),'id');
 assert.deepEqual(errors,[]);console.log('PASS: account language, cross-browser live sync, all feature routes, permission-aware menu parity, Zoom equal heights, mobile layout, save failure.');
}finally{await browser.close()}
