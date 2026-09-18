// Run against the local dev server. Reuses the repository's existing browser tools;
// PLAYWRIGHT_MODULE (module/file URL), CHROME_PATH and RESPONSIVE_ORIGIN may override them.
// All application HTTP and WebSocket traffic is mocked: no production accounts/email.
// Generate the actual download fixtures first: in backend, set DOCUMENT_TEST_OUTPUT
// to the absolute graphify-out directory and run go test ./controller -run TestLongDocuments -count=1.
const {chromium}=await import(process.env.PLAYWRIGHT_MODULE??'../../graphify-out/responsive-tools/node_modules/playwright/index.mjs');
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {checkNavigationUI} from './navigation-ui.mjs';


const origin=process.env.RESPONSIVE_ORIGIN??'http://127.0.0.1:5188';
const long = 'Pekerjaan Pengujian';
const now = new Date().toISOString(), day = now.slice(0,10);
const user = {id:'u1',name:long,email:long+'@example.test',username:'pegawai',roleId:1,role:'Admin',jobTitle:long,salary:15000000,residence:long,phoneNumber:'+628123456789',employmentStatus:'Permanent',avatarUrl:'',isActive:true,createdAt:now,updatedAt:now};
const member = {id:'u1',name:long,avatarUrl:''}, colleague={id:'u2',name:'Rekan Divisi',avatarUrl:''};
const job = {id:1,divisionId:1,roleId:1,jobCategoryId:1,name:long,description:long,isActive:true,sortOrder:0,division:'Project Engineer',role:'PE',category:'MK supervisi',referenced:false,pics:[{userId:'u1',name:long,isActive:true}]};
const workspace = {id:1,name:'Project Engineer',canManage:true,memberCount:3,activeTaskCount:4};
const task = {id:'t1',workspaceId:1,jobId:1,jobName:long,title:long,description:long,relatedUrl:'https://example.test/'+long,startDate:day,dueDate:day,status:'IN_PROGRESS',priority:'HIGH',createdBy:'u1',createdAt:now,updatedAt:now,requiresQa:true,qaStage:'WAITING',canCheckQa:true,version:1,assignees:[member],commentCount:1,canEdit:true,canDelete:true};
const note = {id:'n1',workspaceId:1,parentId:null,title:long,content:[{kind:'text',text:long,checked:false}],visibility:'DIVISION',createdBy:'u1',updatedAt:now,version:1,canEdit:true};
const report = {id:'r1',userId:'u1',jobId:1,primaryDivision:'Project Engineer',workDivision:'MK supervisi',role:'Engineer',work:long,number:4,version:1,description:long,completedAt:now,pics:job.pics};
const archive = {id:'a1',division:'Project Engineer',weekStart:day,weekEnd:day,status:'finalized',finalizedAt:now,pdfGeneratedAt:now,filename:'report.pdf',items:[{...report,employeeName:long,employeeRole:'Engineer'}]};
const resource = {id:1,name:'Zoom 1',code:'Z1',description:long,meetingUrl:'https://example.test/'+long,meetingId:'123456789',hostKey:'123456',createdAt:now,updatedAt:now};
const fixtures = {
 '/auth/me':{user},'/me':{user},'/users':{users:[user]},'/roles':{roles:[{id:1,name:'Admin'}]},
 '/jobs/masters':{divisions:[{id:1,name:'Project Engineer'}],roles:[{id:1,name:'PE',divisionId:1}],categories:[{id:1,name:'MK supervisi',divisionId:1,roleId:1}]},'/jobs':{jobs:[job]},'/jobs/pic-users':{users:[{userId:'u2',name:long,username:long,isActive:true}]},
 '/notifications':{notifications:[{id:'nt1',type:'workspace_task',actor:{id:'u2',name:long},workspace:{title:long,workspaceId:1,message:'Mengubah pekerjaan'},createdAt:now,readAt:null}]},'/notifications/unread-count':{count:1},
 '/workspaces':{workspaces:[workspace]},'/workspaces/1':{workspace,members:[member,colleague]},'/workspaces/1/jobs':{jobs:[job]},'/workspaces/1/tasks':{tasks:[task],total:1},'/workspaces/1/overview':{overview:{active:1,inProgress:1,needRevision:0,done:0,overdue:0}},'/workspaces/1/activity':{activity:[]},
 '/workspaces/1/workload':{workload:[{member,activeTaskCount:1,highPriorityCount:1,dueSoonCount:1,overdueCount:0,overlapCount:0,workloadScore:5,workloadLevel:'NORMAL',tasks:[task]}]},
 '/workspaces/1/pages':{pages:[note]},'/workspace-pages/n1':{page:note},'/work-items/t1':{task},'/work-items/t1/comments':{comments:[{id:'c1',userId:'u1',name:long,content:'See (https://example.test/a).\n<script>alert(1)</script> javascript:alert(1)',createdAt:now}]},'/work-items/t1/activity':{activity:[]},
 '/reports':{reports:[report],groups:[{name:'Engineer',categories:[{name:'MK/Supervisi',jobs:[{key:'1',name:long,pics:job.pics,progress:[{number:1,content:long,report}]}]}]}],period:{start:day,end:day,label:'Periode aktif',readOnly:false}},'/reports/history':{archives:[archive],hasMore:false},'/reports/history/a1':{archive},'/reports/period':{start:day,end:day,label:'Periode aktif',readOnly:false},
 '/zoom-resources':{resources:[resource]},'/zoom-bookings':{bookings:[{id:'b1',userId:'u1',createdBy:long,title:long,description:long,zoomResource:resource,startAt:day+'T02:00:00Z',endAt:day+'T03:00:00Z',createdAt:now,updatedAt:now}]}
};
const browser = await chromium.launch({executablePath:process.env.CHROME_PATH??'C:/Program Files/Google/Chrome/Application/chrome.exe',headless:true,args:['--disable-gpu','--in-process-gpu']});
const context = await browser.newContext({viewport:{width:1280,height:800},hasTouch:true,reducedMotion:'reduce'});
const sockets=[]; await context.routeWebSocket(/ws\/notifications/,ws=>sockets.push(ws));
const missing = new Set(), errors=[], exports=[], mutations=[]; let exportFailure=false;
await context.route('**/*',async route=>{
 const u=new URL(route.request().url());
 if(u.hostname==='fonts.googleapis.com'||u.hostname==='fonts.gstatic.com')return route.fulfill({body:''});
 if(u.port!=='3000')return route.continue();
 if(['/reports.pdf','/reports.docx'].includes(u.pathname)){
  exports.push(u.pathname);
  await new Promise(r=>setTimeout(r,250));
  if(exportFailure)return route.fulfill({status:500,contentType:'application/json',body:JSON.stringify({error:'Export test failure'}),headers:{'Access-Control-Allow-Origin':origin,'Access-Control-Allow-Credentials':'true'}});
  const format=u.pathname.split('.').at(-1);
  return route.fulfill({body:await readFile(new URL('../../graphify-out/long-report.'+format,import.meta.url)),headers:{'Content-Type':format==='pdf'?'application/pdf':'application/vnd.openxmlformats-officedocument.wordprocessingml.document','Access-Control-Expose-Headers':'Content-Disposition','Content-Disposition':'attachment; filename="report.'+format+'"','Access-Control-Allow-Origin':origin,'Access-Control-Allow-Credentials':'true'}});
 }
 if(route.request().method()==='PATCH'&&['/work-items/t1','/work-items/t1/assignees','/work-items/t1/qa'].includes(u.pathname)){
  const body=route.request().postDataJSON();mutations.push({path:u.pathname,body});
  task.version++;
  if(body.assigneeIds)task.assignees=[member,colleague].filter(m=>body.assigneeIds.includes(m.id));
  if(body.decision)Object.assign(task,{status:body.decision,qaStage:body.decision==='DONE'?'COMPLETE':'',qaValidatedAt:body.decision==='DONE'?now:null,qaValidatedBy:body.decision==='DONE'?'u1':null,qaValidatedByName:body.decision==='DONE'?long:''});
  return route.fulfill({contentType:'application/json',body:JSON.stringify({task}),headers:{'Access-Control-Allow-Origin':origin,'Access-Control-Allow-Credentials':'true'}});
 }
 if(route.request().method()==='POST'&&u.pathname==='/work-items/t1/comments'){mutations.push({path:u.pathname,body:route.request().postDataJSON()});return route.fulfill({status:204,headers:{'Access-Control-Allow-Origin':origin,'Access-Control-Allow-Credentials':'true'}})}
 if(u.pathname==='/reports/r1'&&['PATCH','PUT'].includes(route.request().method()))return route.fulfill({status:409,contentType:'application/json',body:JSON.stringify({error:'Laporan sudah berubah',code:'report_conflict',report}),headers:{'Access-Control-Allow-Origin':origin,'Access-Control-Allow-Credentials':'true'}});
 const value=u.pathname==='/workspaces/1/tasks'?{tasks:fixtures[u.pathname].tasks.slice(Number(u.searchParams.get('offset')??0),Number(u.searchParams.get('offset')??0)+Number(u.searchParams.get('limit')??100)),total:fixtures[u.pathname].total}:u.pathname==='/reports'&&route.request().method()==='POST'?{report}:fixtures[u.pathname];
 if(!value&&u.pathname!=='/auth/me')missing.add(u.pathname);
 await route.fulfill({status:value?200:404,contentType:'application/json',body:JSON.stringify(value??{error:'Missing browser fixture'}),headers:{'Access-Control-Allow-Origin':origin,'Access-Control-Allow-Credentials':'true'}});
});
const page=await context.newPage();page.on('pageerror',e=>errors.push(e.message));

try {
 await page.goto(origin+'/workspace/1?task=t1');
 await page.getByRole('dialog').waitFor();
 assert.equal(await page.locator('.ws-comment a').getAttribute('href'),'https://example.test/a');
 assert.equal(await page.locator('.ws-comment a').getAttribute('rel'),'noopener noreferrer');
 assert.equal(await page.locator('.ws-comment script').count(),0);
 await page.getByLabel('Komentar baru').fill('@Re');
 await page.getByRole('button',{name:'@Rekan Divisi',exact:true}).click();
 await page.getByRole('button',{name:'Kirim Komentar',exact:true}).click();
 assert.deepEqual(mutations.at(-1),{path:'/work-items/t1/comments',body:{content:'@Rekan Divisi ',mentionIds:['u2']}});
 await page.getByLabel('Tutup detail').click();
 assert.equal(await page.getByRole('link',{name:'Kembali ke Dashboard'}).getAttribute('href'),'/dashboard');
 const connected=sockets.length;
 await page.getByRole('link',{name:'Kembali ke Dashboard'}).click();
 await page.waitForURL(origin+'/dashboard');
 assert.equal(sockets.length,connected,'client navigation must reuse the layout socket');
 await page.goto(origin+'/workspace/1');
 const done={...task,id:'done',title:'Selesai Tersimpan',status:'DONE',qaStage:'COMPLETE'};
 fixtures['/workspaces/1/tasks']={tasks:[task,...Array.from({length:100},(_,i)=>({...done,id:'done-'+i,title:'Arsip '+i})),done],total:102};
 await page.getByRole('button',{name:'Kalender',exact:true}).click();
 await page.locator('.ws-calendar').waitFor();
 assert.equal(await page.getByRole('button',{name:'Selesai Tersimpan',exact:true}).count(),0);
 await page.locator('.ws-calendar-day[aria-label="'+day+'"]').getByRole('button').first().click();
 await page.getByRole('button',{name:'Selesai Tersimpan',exact:true}).waitFor();
 assert.equal(await page.locator('dialog[open] .ws-link').count(),102,'date dialog includes later pages');
 assert.equal(await page.locator('.ws-panel .ws-link').count(),0,'no detail list below calendar');
 await page.getByRole('button',{name:'Tutup tanggal',exact:true}).click();
 const calendarBounds=await page.locator('.ws-calendar').boundingBox();
 await page.locator('.ws-calendar-task').first().click();
 await page.getByRole('dialog',{name:'Detail Pekerjaan',exact:true}).waitFor();
 assert.deepEqual(await page.locator('.ws-calendar').boundingBox(),calendarBounds,'modal must not move calendar');
 assert.equal(await page.getByRole('dialog').evaluate(el=>el.matches(':modal')),true);
 await page.keyboard.press('Escape');
 assert.equal(await page.locator('dialog[open]').count(),0);
 await page.getByRole('button',{name:'+ Tambah Pekerjaan',exact:true}).click();
 assert.equal(await page.getByLabel('Butuh validasi QA sebelum pekerjaan dapat diselesaikan').isChecked(),false);
 assert.equal(await page.getByRole('dialog').getByRole('combobox',{name:/^Status/}).inputValue(),'TODO');
 await page.getByLabel('Tutup detail').click();
 task.canEdit=false;task.canCheckQa=false;
 await page.goto(origin+'/workspace/1?task=t1');
 assert.equal(await page.getByRole('button',{name:'Validasi QA',exact:true}).isEnabled(),true);
 task.canCheckQa=true;
 const peer=await context.newPage();peer.on('pageerror',e=>errors.push(e.message));
 await peer.goto(origin+'/workspace/1?task=t1');
 await peer.getByRole('dialog',{name:'Detail Pekerjaan',exact:true}).waitFor();
 assert.equal(await page.getByLabel('Rekan Divisi',{exact:true}).isDisabled(),true);
 task.canEdit=true;
 await page.goto(origin+'/workspace/1?task=t1');
 await page.getByLabel('Rekan Divisi',{exact:true}).check();
 await page.getByRole('button',{name:'Simpan Pekerjaan',exact:true}).click();
 await page.getByRole('dialog',{name:'Detail Pekerjaan',exact:true}).waitFor({state:'hidden'});
 assert.deepEqual(mutations.at(-1).body.assigneeIds,['u1','u2']);
 assert.deepEqual(Object.keys(mutations.at(-1).body).sort(),['assigneeIds','version']);
 for(const ws of sockets)try{ws.send(JSON.stringify({event:'workspace.changed',workspaceId:1,entityId:'t1',revision:task.version}))}catch{}
 await peer.waitForFunction(()=>Array.from(document.querySelectorAll('dialog[open] input[type=checkbox]')).some(el=>el.value==='u2'&&el.checked));
 await page.goto(origin+'/workspace/1?task=t1');
 await page.getByRole('button',{name:'Validasi QA',exact:true}).click();
 const validation=page.getByRole('dialog',{name:'Validasi QA',exact:true});
 await validation.getByRole('combobox').selectOption('DONE');
 await validation.getByRole('button',{name:'Simpan Validasi',exact:true}).click();
 await page.getByText('Divalidasi QA',{exact:true}).waitFor();
 assert.equal(task.status,'DONE');
 assert.ok(task.qaValidatedAt);
 for(const ws of sockets)try{ws.send(JSON.stringify({event:'workspace.changed',workspaceId:1,entityId:'t1',revision:task.version}))}catch{}
 await peer.getByText('Divalidasi QA',{exact:true}).waitFor();
 await peer.close();
 await page.getByLabel('Tutup detail').click();
 task.canEdit=true;
 for(const view of ['Tabel','Papan','Linimasa','Beban Kerja']){
  await page.getByRole('button',{name:view,exact:true}).click();
  await page.getByRole('button',{name:long,exact:true}).first().click();
  await page.getByRole('dialog',{name:'Detail Pekerjaan',exact:true}).getByText('Divalidasi QA',{exact:true}).waitFor();
 await page.getByLabel('Tutup detail').click();
 }
 Object.assign(task,{status:'IN_PROGRESS',qaStage:'',qaValidatedAt:null,qaValidatedBy:null,qaValidatedByName:''});
 await page.goto(origin+'/workspace/1?task=t1');
 await page.getByRole('button',{name:'Validasi QA',exact:true}).click();
 await page.getByRole('dialog',{name:'Validasi QA',exact:true}).getByRole('combobox').selectOption('NEED_REVISION');
 await page.getByRole('button',{name:'Simpan Validasi',exact:true}).click();
 assert.equal(task.status,'NEED_REVISION');
 await page.getByLabel('Tutup detail').click();
 Object.assign(task,{status:'IN_PROGRESS',qaStage:'',qaValidatedAt:null,qaValidatedBy:null,qaValidatedByName:''});
 for(const width of [1280,390]){
  await page.setViewportSize({width,height:740});
  await page.goto(origin+'/workspace/1');
  await page.getByRole('button',{name:'Kalender',exact:true}).click();
  await page.locator('.ws-calendar-task').first().click();
  const detail=page.getByRole('dialog',{name:'Detail Pekerjaan',exact:true});
  await detail.waitFor();
  const bounds=await detail.boundingBox();
  assert.ok(bounds.width<=width&&bounds.height<=740&&bounds.x>=0&&bounds.y>=0,JSON.stringify(bounds));
  await page.keyboard.press('Escape');
 }
 await page.setViewportSize({width:1280,height:800});
 await page.goto(origin+'/weekly-progress?division=Project%20Engineer');
 await page.locator('.edit-report').first().click();
 await page.getByLabel('Uraian pekerjaan yang dilakukan').fill('Draf QA tetap utuh');
 report.version=2; report.description='Hasil PE terbaru';
 await page.getByRole('button',{name:'Simpan perubahan',exact:true}).click();
 await page.getByText('Data server berubah (versi 2).',{exact:false}).waitFor();
 assert.equal(await page.getByLabel('Uraian pekerjaan yang dilakukan').inputValue(),'Draf QA tetap utuh');
 report.version=3;report.description='Pembaruan lintas sesi';
 for(const ws of sockets)try{ws.send(JSON.stringify({event:'reports.changed',entityId:'r1',revision:3}))}catch{}
 await page.getByText('Data server berubah (versi 3).',{exact:false}).waitFor();
 assert.equal(await page.getByLabel('Uraian pekerjaan yang dilakukan').inputValue(),'Draf QA tetap utuh');
 assert.equal((await page.locator('.task-number').first().innerText()).trim(),'1');
 assert.ok(!(await page.locator('body').innerText()).includes('Confidential'));
 const before=sockets.length;sockets.at(-1).close({code:1012,reason:'test reconnect'});
 await page.waitForFunction(()=>true);
 for(let n=0;n<30&&sockets.length===before;n++)await new Promise(r=>setTimeout(r,100));
 assert.equal(sockets.length,before+1);
 await page.getByRole('button',{name:'Gunakan data server',exact:true}).click();
 assert.equal(await page.getByLabel('Uraian pekerjaan yang dilakukan').inputValue(),'Pembaruan lintas sesi');
 await page.getByRole('button',{name:'Batal',exact:true}).click();
 await page.getByRole('button',{name:'Ekspor',exact:true}).click();
 await page.getByRole('dialog',{name:'Ekspor Laporan Mingguan',exact:true}).waitFor();
 assert.equal(exports.length,0,'opening format modal must not download');
 await page.getByRole('dialog',{name:'Ekspor Laporan Mingguan',exact:true}).getByRole('button',{name:'Batal',exact:true}).click();
 assert.equal(exports.length,0);
 for(const format of ['PDF','Word']){
  await page.getByRole('button',{name:'Ekspor',exact:true}).click();
  const download=page.waitForEvent('download');
  await page.getByRole('button',{name:'Ekspor '+format,exact:true}).click();
  assert.equal(await page.getByRole('button',{name:'Ekspor '+format,exact:true}).isDisabled(),true);
  assert.equal((await download).suggestedFilename(),format==='PDF'?'report.pdf':'report.docx');
  await page.getByRole('dialog',{name:'Ekspor Laporan Mingguan',exact:true}).waitFor({state:'hidden'});
 }
 assert.deepEqual(exports,['/reports.pdf','/reports.docx']);
 exportFailure=true;
 await page.getByRole('button',{name:'Ekspor',exact:true}).click();
 await page.getByRole('button',{name:'Ekspor Word',exact:true}).click();
 await page.getByRole('alert').getByText('Unduh gagal.',{exact:true}).waitFor();
 await page.getByRole('button',{name:'Batal',exact:true}).click();
 await page.goto(origin+'/weekly-progress/new?division=Project%20Engineer');
 const progress=page.getByLabel('Uraian pekerjaan yang dilakukan');
 assert.equal(await progress.getAttribute('maxlength'),null);
 await progress.fill('Isi laporan panjang\n'.repeat(400));
 await page.getByRole('button',{name:'Simpan laporan',exact:true}).click();
 await page.getByText('Laporan berhasil ditambahkan',{exact:true}).waitFor();
 fixtures['/zoom-bookings'].bookings=Array.from({length:12},(_,i)=>({...fixtures['/zoom-bookings'].bookings[0],id:'booking-'+i,title:'Jadwal '+i}));
 await page.goto(origin+'/booking-zoom');
 const calendar=page.locator('.calendar-scroll');
 await calendar.waitFor();
 for(const width of [1440,1280,800,390]){
  await page.setViewportSize({width,height:740});
  const dimensions=await calendar.evaluate(el=>{el.scrollTop=200;return {height:el.clientHeight,full:el.scrollHeight,top:el.scrollTop,overflow:getComputedStyle(el).overflowY,lane:el.querySelector('.calendar-lane')?.getBoundingClientRect().height,grid:el.querySelector('.zoom-calendar')?.getBoundingClientRect().height}});
  assert.ok(dimensions.full>dimensions.height&&dimensions.top>0,JSON.stringify(dimensions));
  assert.ok(['auto','scroll'].includes(dimensions.overflow));
  const layout=await calendar.evaluate(el=>{
   el.scrollTop=el.scrollHeight;el.scrollLeft=el.scrollWidth;
   const card=el.closest('.schedule-card').getBoundingClientRect(),body=el.getBoundingClientRect();
   return {gap:card.bottom-body.bottom,bottom:el.scrollHeight-el.clientHeight-el.scrollTop,horizontal:el.scrollLeft,extraWidth:el.scrollWidth-el.clientWidth};
  });
  assert.ok(Math.abs(layout.gap)<3,JSON.stringify(layout));
  assert.ok(Math.abs(layout.bottom)<2,JSON.stringify(layout));
  assert.equal(layout.horizontal,layout.extraWidth,'horizontal scrolling reaches the final column when needed');
 }
 await checkNavigationUI({page,context,origin,fixtures,user});
 assert.deepEqual(errors,[]);
 assert.deepEqual([...missing],[]);
 console.log('PASS browser: comments, dashboard, assignment, audited QA, calendar overlay, report conflict/draft/reconnect, PDF/Word modal/download/error, long progress toast, Zoom fill/scroll at four widths');
} finally {await browser.close()}
