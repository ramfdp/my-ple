// Run with the dev server on 5188 and VITE_API_URL=http://localhost:3000.
import assert from 'node:assert/strict';
const {chromium} = await import('../../graphify-out/responsive-tools/node_modules/playwright/index.mjs');
const browser = await chromium.launch({executablePath:process.env.CHROME_PATH ?? 'C:/Program Files/Google/Chrome/Application/chrome.exe',args:['--disable-gpu','--in-process-gpu']});
try {
 const page = await browser.newPage();
 const origin = process.env.RESPONSIVE_ORIGIN ?? 'http://127.0.0.1:5188';
 const logo = '/logo%20MyHIS%20terbaru.jpeg';
 let authenticated = false;
 await page.route('**://localhost:3000/**', route => {
  const path = new URL(route.request().url()).pathname;
  const auth = path.startsWith('/auth/');
  return route.fulfill({status:route.request().method() !== 'OPTIONS' && auth && !authenticated ? 401 : 200,contentType:'application/json',headers:{'Access-Control-Allow-Origin':origin,'Access-Control-Allow-Credentials':'true','Access-Control-Allow-Headers':'Content-Type','Access-Control-Allow-Methods':'GET,POST,OPTIONS'},body:JSON.stringify(auth ? {user:{id:'test',name:'Test',role:'Admin',language:'id'}} : {jobs:[],checklists:[]})});
 });
 for (const width of [320,768,1440]) {
  await page.setViewportSize({width,height:900});
  await page.goto(origin);
  await page.locator('.login-intro').waitFor();
  assert.equal(await page.locator('#username').count(),0,'Form stays hidden during intro');
  await page.locator('#username').waitFor();
  assert.equal(await page.locator('.brand-logo').getAttribute('src'),logo);
  assert.ok(await page.locator('.brand-logo').evaluate(img=>img.complete && img.naturalWidth>0));
  assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
  await page.screenshot({path:`../graphify-out/login-brand-${width}.png`,fullPage:true});
 }
 await page.locator('#username').fill('wrong');
 await page.locator('#password').fill('wrong');
 await page.locator('button[type=submit]').click();
 await page.locator('.form-note').waitFor();
 assert.equal(await page.locator('.login-intro').count(),0,'Failed login does not replay intro');
 assert.equal(await page.locator('#username').inputValue(),'wrong');
 await page.emulateMedia({reducedMotion:'reduce'});
 await page.goto(origin);
 await page.locator('#username').waitFor();
 assert.equal(await page.locator('.login-intro').count(),0);
 assert.equal(await page.locator('.login-page').evaluate(el=>getComputedStyle(el).animationName),'none');
 assert.equal(new URL(await page.locator('link[rel=icon]').getAttribute('href'),origin).pathname,logo);
 authenticated = true;
 await page.goto(origin+'/document-checklists');
 await page.locator('.sidebar-logo').waitFor();
 assert.equal(await page.locator('.sidebar-logo').getAttribute('src'),logo);
 await page.setViewportSize({width:390,height:844});
 await page.locator('.mobile-logo-img').waitFor();
 assert.equal(await page.locator('.mobile-logo-img').getAttribute('src'),logo);
 assert.equal(await page.locator('.login-intro').count(),0);
 console.log('PASS intro, responsive logo, failed login, reduced motion, favicon, desktop/mobile branding');
} finally { await browser.close(); }
