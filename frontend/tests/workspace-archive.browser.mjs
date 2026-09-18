// Run with TestDocumentChecklistBrowserServer on 3001 and Vite on 5189.
import assert from 'node:assert/strict';
const {chromium}=await import(process.env.PLAYWRIGHT_MODULE ?? '../../graphify-out/responsive-tools/node_modules/playwright/index.mjs');
const origin='http://127.0.0.1:5189',backend='http://127.0.0.1:3001';
const browser=await chromium.launch({executablePath:process.env.CHROME_PATH??'C:/Program Files/Google/Chrome/Application/chrome.exe',headless:true,args:['--disable-gpu','--in-process-gpu']});
const context=await browser.newContext({viewport:{width:1440,height:1000}}),page=await context.newPage(),errors=[];
page.on('pageerror',e=>errors.push(e.message));
async function api(method,path,data){const r=await context.request.fetch(backend+path,{method,data});assert.ok(r.ok(),`${method} ${path}: ${r.status()} ${await r.text()}`);return r.status()===204?null:r.json()}
async function login(p){await p.goto(origin);await p.locator('#username').fill('browser-Admin');await p.locator('#password').fill('Checklist-test-only-2026');await p.locator('button[type=submit]').click();await p.waitForURL('**/dashboard');await p.locator('.dashboard-actions select').waitFor()}
async function language(lang){await page.goto(origin+'/dashboard');const done=page.waitForResponse(r=>r.url().endsWith('/profile/language'));await page.locator('.dashboard-actions select').selectOption(lang);await done;await page.waitForFunction(l=>document.documentElement.lang===l,lang)}
try{
 await login(page);await language('en');
 const {workspaces}=await api('GET','/workspaces');const wid=workspaces[0].id;
 let task=(await api('POST',`/workspaces/${wid}/tasks`,{title:'Archive browser '+Date.now(),description:'Original job content',relatedUrl:'https://example.test',assigneeIds:[]})).task;
 await api('POST',`/work-items/${task.id}/comments`,{content:'Preserved discussion'});
 await page.goto(origin+'/workspace/'+wid);await page.getByRole('button',{name:'Table',exact:true}).click();
 await page.getByRole('button',{name:task.title,exact:true}).click();await page.getByRole('button',{name:'Archive',exact:true}).click();await page.getByRole('button',{name:'Yes, archive',exact:true}).click();await page.locator('dialog[open]').waitFor({state:'hidden'});
 await page.getByRole('button',{name:'Archived Jobs',exact:true}).click();await page.getByRole('heading',{name:'Archived Jobs',exact:true}).waitFor();
 await page.getByRole('button',{name:task.title,exact:true}).click();const modal=page.locator('dialog[open]');
 await modal.getByText('Original job content',{exact:true}).waitFor();await modal.getByText('Preserved discussion',{exact:true}).waitFor();assert.equal(await modal.locator('form').count(),0);
 await page.getByRole('button',{name:'Restore',exact:true}).click();
 await page.route('**/work-items/*/restore',route=>route.fulfill({status:500,contentType:'application/json',body:JSON.stringify({error:'Terjadi kesalahan pada server.'})}),{times:1});
 await page.getByRole('button',{name:'Yes, restore',exact:true}).click();await modal.getByRole('alert').waitFor();assert.equal(await modal.isVisible(),true);
 const peerContext=await browser.newContext(),peer=await peerContext.newPage();await login(peer);await peer.goto(origin+'/workspace/'+wid);await peer.getByRole('button',{name:'Archived Jobs',exact:true}).click();await peer.getByRole('button',{name:task.title,exact:true}).waitFor();
 await page.getByRole('button',{name:'Yes, restore',exact:true}).click();await modal.waitFor({state:'hidden'});await peer.getByRole('button',{name:task.title,exact:true}).waitFor({state:'hidden'});await peerContext.close();
 await page.getByRole('button',{name:'Back to active jobs',exact:true}).click();await page.getByRole('button',{name:task.title,exact:true}).waitFor();
 task=(await api('GET',`/work-items/${task.id}`)).task;assert.equal(task.description,'Original job content');assert.equal(task.commentCount,1);
 await api('DELETE',`/work-items/${task.id}`,{version:task.version});
 await language('id');await page.goto(origin+'/workspace/'+wid);await page.getByRole('button',{name:'Arsip Pekerjaan',exact:true}).click();await page.getByRole('button',{name:task.title,exact:true}).waitFor();
 await page.setViewportSize({width:390,height:844});assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true);
 await page.getByRole('button',{name:task.title,exact:true}).click();await modal.getByRole('button',{name:'Pulihkan',exact:true}).waitFor();await page.keyboard.press('Escape');await modal.waitFor({state:'hidden'});
 await page.locator('input[type=search]').fill('No matching job '+Date.now());await page.getByRole('button',{name:'Terapkan',exact:true}).click();await page.getByText('Belum ada pekerjaan yang diarsipkan.',{exact:true}).waitFor();
 assert.deepEqual(errors,[]);console.log('PASS: archive, read-only details/history, failed restore, restore, live sync, languages, mobile, keyboard, empty filter.');
}finally{await browser.close()}
