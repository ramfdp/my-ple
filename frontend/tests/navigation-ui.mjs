import assert from 'node:assert/strict';
import {mkdir} from 'node:fs/promises';
import {fileURLToPath} from 'node:url';

// Called by browser-revisions.mjs; shares its mocked API and isolated browser.
export async function checkNavigationUI({page, context, origin, fixtures, user}) {
 const output = new URL('../../graphify-out/navigation-ui/', import.meta.url);
 await mkdir(output, {recursive:true});
 const today = new Date().toISOString().slice(0,10), now = new Date().toISOString();
 const job = fixtures['/jobs'].jobs[0];
 fixtures['/workspaces/1/tasks']={tasks:[fixtures['/work-items/t1'].task],total:1};
 fixtures['/zoom-bookings'].bookings=fixtures['/zoom-bookings'].bookings.slice(0,1);
 Object.assign(fixtures, {
  '/attendance':{records:[],today:null,open:null,todayDate:today,officeConfigured:true},
  '/attendance/overview':{date:today,todayDate:today,employees:[{userId:user.id,name:user.name,role:'Admin',status:'MISSING',leaveTypes:'',mode:'',clockIn:null,clockOut:null,latitude:null,longitude:null}]},
  '/leave-requests':{requests:[]}, '/leave-requests/approvals':{requests:[]},
  '/billing-documents':{jobs:[job],stages:[{id:1,jobId:job.id,name:'Tahap penagihan dengan nama panjang',periodLabel:'Triwulan pertama',notes:'Catatan pengujian layout responsif',updatedAt:now,version:1,readiness:'BELUM SIAP',milestones:[{id:1,name:'Milestone pengujian dokumen',notes:'',updatedAt:now,readiness:'BELUM SIAP',documents:[{id:1,name:'Berita acara pemeriksaan pekerjaan dengan nama panjang',status:'BELUM ADA',updatedAt:now}]}]}]},
  '/document-checklists':{jobs:[job]}, '/document-checklists/1':{checklists:[]}
 });
 const routes = ['/dashboard','/attendance','/attendance/overview','/billing-documents','/document-checklists','/employees','/my-profile','/settings/jobs','/leave','/leave/approvals','/booking-zoom','/weekly-progress','/weekly-progress/new','/weekly-progress/history','/workspace','/workspace/1'];
 const back = () => page.locator('.back-link, .booking-back, .page-back button');
 async function layout(label) {
  const result = await page.evaluate(() => {
   const covered = [...document.querySelectorAll('main button, main a, main input, main select')].filter(el => {
    const r=el.getClientRects()[0] ?? el.getBoundingClientRect();
    if(!el.checkVisibility() || el.closest('[inert]') || r.width<1 || r.height<1 || r.top<56 || r.bottom>innerHeight || r.left<0 || r.right>innerWidth) return false;
    // Offscreen cells inside intentional horizontal scroll containers are not visible controls.
    for(let p=el.parentElement;p;p=p.parentElement) {
     const s=getComputedStyle(p), b=p.getBoundingClientRect();
     if(/auto|scroll|hidden/.test(s.overflowX) && (r.left<b.left || r.right>b.right)) return false;
     if(/auto|scroll|hidden/.test(s.overflowY) && (r.top<b.top || r.bottom>b.bottom)) return false;
    }
    const hit=document.elementFromPoint(r.x+r.width/2,r.y+r.height/2);
    return hit && !el.contains(hit);
   }).map(el => { const r=el.getClientRects()[0] ?? el.getBoundingClientRect(); return {control:el.outerHTML.slice(0,180),cover:document.elementFromPoint(r.x+r.width/2,r.y+r.height/2)?.outerHTML.slice(0,180)}; });
   return {width:innerWidth,scroll:document.documentElement.scrollWidth,covered};
  });
  if(result.scroll>result.width+1 || result.covered.length) await page.screenshot({path:fileURLToPath(new URL('failure.png',output)),fullPage:true});
  assert.ok(result.scroll<=result.width+1, `${label}: page overflow ${JSON.stringify(result)}`);
  assert.deepEqual(result.covered, [], `${label}: controls covered by other content`);
 }
 const originalName=user.name;
 user.name='Nama Karyawan Pengujian Dengan Nama Lengkap Sangat Panjang';
 for(const language of ['id','en']) {
  user.language=language;
  for(const route of routes) {
   await page.goto(origin+route);
   await page.locator('main').waitFor();
   await page.waitForLoadState('networkidle');
   assert.equal(await back().count(),route==='/dashboard'?0:1,`${language} ${route}: exactly one page back link`);
   assert.equal(await page.locator('a').filter({hasText:/^←/}).count(),route==='/dashboard'?0:1,`${route}: no second back destination`);
   if(route!=='/dashboard') assert.equal(await back().getAttribute('href'),'/dashboard');
   for(const width of [320,390,768,867,869,1024,1440]) {
    await page.setViewportSize({width,height:900});
    await page.evaluate(()=>new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve))));
    await layout(`${language} ${route} ${width}`);
    if(language==='id' && [320,768,1440].includes(width)) await page.screenshot({path:fileURLToPath(new URL(`${route.slice(1).replaceAll('/','-')}-${width}.png`,output)),fullPage:true});
   }
   if(route!=='/dashboard') {
    await back().click();
    await page.waitForURL(origin+'/dashboard');
    assert.equal(await back().count(),0);
   }
  }
 }
 console.log('PASS navigation/layout: 16 routes, two languages, seven widths');
 user.name=originalName; user.language='id';
 await page.goto(origin+'/profile');
 await page.waitForURL(origin+'/my-profile');
 assert.equal(await back().count(),1);
 await page.reload(); await page.locator('main').waitFor();
 assert.equal(await back().count(),1);
 await page.setViewportSize({width:320,height:740});
 await page.locator('.mobile-menu-btn').click();
 assert.equal(await page.locator('.feature-content').getAttribute('inert'),'');
 await page.keyboard.press('Escape');
 assert.equal(await page.locator('.mobile-menu-btn').evaluate(el=>el===document.activeElement),true);

 // Exercise modal forms at phone/tablet widths, including nested signer fields.
 for(const width of [320,768]) {
  await page.setViewportSize({width,height:740});
  for(const [route,button] of [['/billing-documents','Tambah tahap penagihan'],['/leave','+ Ajukan Izin / Cuti'],['/my-profile','Ubah Profil'],['/document-checklists','+ Buat Tabel Checklist']]) {
   await page.goto(origin+route); await page.waitForLoadState('networkidle');
   if(route==='/document-checklists') await page.locator('.job-heading').click();
   await page.getByRole('button',{name:button,exact:true}).click();
   const dialog=page.getByRole('dialog'); await dialog.waitFor();
   if(route==='/document-checklists') await dialog.getByLabel('Berapa banyak TTD Internal?').fill('2');
   assert.ok(await dialog.evaluate(el=>{const r=el.getBoundingClientRect();return r.left>=0&&r.right<=innerWidth+1&&r.top>=0&&r.bottom<=innerHeight+1&&el.scrollWidth<=el.clientWidth+1}),`${route} modal ${width}`);
   await page.screenshot({path:fileURLToPath(new URL(`modal-${route.slice(1)}-${width}.png`,output)),fullPage:true});
  }
 }
 await page.setViewportSize({width:320,height:740});
 await page.goto(origin+'/dashboard');
 await page.locator('.notification-button').click();
 const panel=page.locator('.notification-panel');
 assert.equal(await panel.evaluate(el=>el.matches(':popover-open')),true);
 assert.ok(await panel.evaluate(el=>{const r=el.getClientRects()[0] ?? el.getBoundingClientRect();return r.left>=0&&r.right<=innerWidth&&r.bottom<=innerHeight}));
 await page.keyboard.press('Escape');

 // Normal motion runs once on route entry, never on typing or query-only changes.
 await page.emulateMedia({reducedMotion:'no-preference'});
 await page.addInitScript(() => {
  const animate=Element.prototype.animate;
  window.pageFades=0;
  Element.prototype.animate=function(...args) {
   if(this.matches('.feature-content, .login-content-frame')) window.pageFades++;
   return animate.apply(this,args);
  };
 });
 await page.goto(origin+'/weekly-progress/new');
 const draft=page.locator('textarea').first(); await draft.waitFor();
 const count=await page.evaluate(()=>window.pageFades);
 assert.ok(count>0);
 await draft.fill('Draf yang tidak boleh hilang');
 await page.evaluate(()=>history.replaceState(history.state,'',location.pathname+'?division=Project%20Engineer'));
 assert.equal(await draft.inputValue(),'Draf yang tidak boleh hilang');
 assert.equal(await page.evaluate(()=>window.pageFades),count);
 await back().click(); await page.waitForURL(origin+'/dashboard');
 await page.waitForFunction(n=>window.pageFades>n,count);
 await page.emulateMedia({reducedMotion:'reduce'});
 await page.goto(origin+'/weekly-progress/new'); await page.locator('main').waitFor();
 assert.equal(await page.evaluate(()=>window.pageFades),0);

 // Verify login without touching any real session, and loading/failed/empty feature states.
 const auth=fixtures['/auth/me']; delete fixtures['/auth/me'];
 await page.goto(origin); await page.locator('#username').waitFor();
 assert.equal(await back().count(),0); await layout('login 320');
 fixtures['/auth/me']=auth;
 const billing=fixtures['/billing-documents'];
 fixtures['/billing-documents']={jobs:[],stages:[]};
 await page.goto(origin+'/billing-documents'); await page.waitForLoadState('networkidle'); await layout('empty billing');
 const failure=route=>route.fulfill({status:500,contentType:'application/json',body:JSON.stringify({error:'Test unavailable'}),headers:{'Access-Control-Allow-Origin':origin,'Access-Control-Allow-Credentials':'true'}});
 await context.route('**:3000/billing-documents',failure);
 await page.reload(); await page.getByRole('alert').waitFor(); await layout('failed billing');
 await context.unroute('**:3000/billing-documents',failure); fixtures['/billing-documents']=billing;
 await page.setViewportSize({width:1280,height:800});
 console.log('PASS navigation UI: 16 routes, two languages, seven widths, single dashboard return, sidebar, popover, motion, drafts, login, empty/error states');
}
