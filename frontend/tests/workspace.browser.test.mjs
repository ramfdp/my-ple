// Run Vite on 127.0.0.1:5190, then node tests/workspace.browser.test.mjs.
// Uses an isolated Chromium profile and fixture APIs; never sends email.
import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { setTimeout as delay } from 'node:timers/promises';

const profile = await mkdtemp(join(tmpdir(), 'workspace-browser-'));
const browser = spawn(process.env.CHROME_PATH || 'C:/Program Files/Google/Chrome/Application/chrome.exe', ['--headless=new', '--disable-gpu', '--in-process-gpu', '--no-sandbox', '--disable-extensions', '--disable-background-networking', '--no-first-run', '--no-default-browser-check', '--remote-debugging-port=0', `--user-data-dir=${profile}`, 'about:blank'], { windowsHide: true, stdio: ['ignore','ignore','pipe'] });
let browserLog='';browser.stderr.on('data',data=>browserLog=(browserLog+data).slice(-3000));
let socket;
try {
  let port;
  for (let i = 0; i < 100; i++) { try { port = (await readFile(join(profile, 'DevToolsActivePort'), 'utf8')).split('\n')[0]; break; } catch { await delay(100); } }
  assert(port, 'Chromium failed to start');
  await delay(500); // Let the initial renderer finish attaching before CDP commands.
  const tabs = await (await fetch(`http://127.0.0.1:${port}/json/list`)).json();
  socket = new WebSocket(tabs.find(t => t.type === 'page').webSocketDebuggerUrl);
  await new Promise(resolve => socket.addEventListener('open', resolve, { once: true }));
  const pending = new Map(); let sequence = 0;
  socket.addEventListener('close', event => { for(const p of pending.values()) p.reject(new Error(`CDP closed: ${event.code} ${event.reason}`)); pending.clear(); });
  const errors = [];
  socket.addEventListener('message', ({ data }) => {
    const message = JSON.parse(data);
    if (message.method === 'Runtime.exceptionThrown') errors.push(message.params.exceptionDetails.text);
    if (message.id) { const p = pending.get(message.id); pending.delete(message.id); message.error ? p.reject(message.error) : p.resolve(message.result); }
  });
  const call = (method, params = {}) => new Promise((resolve, reject) => { const id = ++sequence; const timer=setTimeout(()=>reject(new Error('CDP timeout: '+method)),10000); pending.set(id, { resolve:value=>{clearTimeout(timer);resolve(value)}, reject:error=>{clearTimeout(timer);reject(error)} }); socket.send(JSON.stringify({ id, method, params })); });
  const evaluate = async expression => { const r = await call('Runtime.evaluate', { expression, awaitPromise: true, returnByValue: true, userGesture: true }); assert(!r.exceptionDetails, JSON.stringify(r.exceptionDetails)); return r.result.value; };
  const waitFor = async expression => { for (let i = 0; i < 100; i++) { if (await evaluate(expression)) return; await delay(100); } throw new Error(`Timed out: ${expression}`); };
  await call('Runtime.enable'); await call('Page.enable');

  await call('Page.addScriptToEvaluateOnNewDocument',{source:`
   window.saved=[];window.feed=[];window.comments=[];window.activities=[];
   window.task={id:'activity-task',workspaceId:1,title:'Activity fixture',description:'',relatedUrl:'',jobId:null,startDate:'',dueDate:'2099-01-01',status:'TODO',priority:'MEDIUM',assignees:[],canEdit:true,canDelete:true,version:1};
   window.jobs=[{id:17,name:'MPS SGR',label:'MPS SGR',isActive:true},{id:17,name:'MPS SGR',label:'MPS SGR',isActive:true},{id:18,name:'MPS PIT',isActive:true},{id:19,name:'MPS SGP',isActive:true},...Array.from({length:30},(_,i)=>({id:100+i,name:'Long project '+('name '.repeat(25))+i,isActive:true}))];
   const realFetch=window.fetch;
   window.fetch=async(input,options)=>{
    const url=new URL(typeof input==='string'?input:input.url,location.href);
    if(url.port!=='3000')return realFetch(input,options);
    let body={};
    if(url.pathname==='/auth/me')body={user:{id:'budi',name:'Budi',role:'Management',isActive:true}};
    if(url.pathname==='/workspaces/1')body={workspace:{id:1,name:'Project Engineer',canManage:true},members:[{id:'budi',name:'Budi',avatarUrl:''},{id:'andi',name:'Andi',avatarUrl:''}]};
    if(url.pathname.endsWith('/jobs'))body={jobs:window.jobs};
    if(url.pathname.endsWith('/tasks')){body={tasks:window.activities.length?[window.task]:[],total:window.activities.length?1:0};if(options?.method==='POST'){window.saved.push(JSON.parse(options.body));body={task:{id:'saved',...window.saved.at(-1)}};}}
    if(url.pathname.endsWith('/workload'))body={workload:[]};
    if(url.pathname.endsWith('/overview'))body={overview:{}};
    if(url.pathname.endsWith('/activity'))body={activity:window.activities};
    if(url.pathname==='/work-items/activity-task')body={task:window.task};
    if(url.pathname.endsWith('/comments')){if(options?.method==='POST')window.comments.push({id:'comment',name:'Budi',createdAt:new Date().toISOString(),...JSON.parse(options.body)});body={comments:window.comments};}
    if(url.pathname==='/notifications')body={notifications:window.feed};
    if(url.pathname==='/notifications/unread-count')body={count:window.feed.length};
    return new Response(JSON.stringify(body),{status:200,headers:{'Content-Type':'application/json'}});
   };
   window.WebSocket=class{constructor(){window.testSocket=this;setTimeout(()=>this.onopen?.(),20)}close(){}};
  `});
  await call('Page.navigate',{url:'http://127.0.0.1:5190/workspace/1'});
  await waitFor(`!!document.querySelector('.workspace-page header .ws-primary')`);
  const open=async()=>{await evaluate(`document.querySelector('.workspace-page header .ws-primary').click()`);await waitFor(`!!document.querySelector('dialog[open]')`)};
  const search=async text=>{await evaluate(`document.querySelector('[role=combobox]').focus()`);await call('Input.insertText',{text});await delay(50)};
  const clear=async()=>evaluate(`(()=>{const c=document.querySelector('[role=combobox]');c.value='';c.dispatchEvent(new Event('input',{bubbles:true}))})()`);
  const names=()=>evaluate(`Array.from(document.querySelectorAll('[role=option] button'),e=>e.textContent.trim())`);
  await open();
  await search('mps');
  assert.deepEqual((await names()).sort(),['MPS PIT','MPS SGP','MPS SGR']);
  assert.equal(await evaluate(`document.querySelector('[role=combobox]').getAttribute('aria-expanded')`),'true');
  for(const q of ['mps sgr','MPS SGR','  sgr  ']){await clear();await search(q);assert.deepEqual(await names(),['MPS SGR'])}
  await evaluate(`document.querySelector('[role=option] button').click()`);
  assert.equal(await evaluate(`document.querySelector('[role=combobox]').value`),'MPS SGR');
  assert.equal(await evaluate(`document.querySelector('[role=combobox]').getAttribute('aria-expanded')`),'false');
  await evaluate(`(()=>{const title=document.querySelector('input[required]');title.value='Review progress';title.dispatchEvent(new Event('input',{bubbles:true}));document.querySelector('dialog form').requestSubmit()})()`);
  await waitFor(`window.saved.length===1&&!document.querySelector('dialog[open]')`);
  assert.equal(await evaluate(`window.saved[0].jobId`),17);
  assert.match(await evaluate(`window.saved[0].requestId`),/^[\da-f-]{36}$/);
  await open();await search('abcxyz');
  assert.deepEqual(await names(),[]);assert(await evaluate(`document.querySelector('[role=listbox]').textContent.includes('Tidak ada project ditemukan')`));
  await call('Input.dispatchKeyEvent',{type:'keyDown',key:'Escape',code:'Escape'});
  assert(await evaluate(`!!document.querySelector('dialog[open]')&&!document.querySelector('[role=listbox]')`),'Escape closes results before dialog');
  await evaluate(`document.querySelector('[role=combobox]').click()`);await clear();await search('mps sgr');
  await call('Input.dispatchKeyEvent',{type:'keyDown',key:'ArrowDown',code:'ArrowDown'});
  await call('Input.dispatchKeyEvent',{type:'keyDown',key:'Enter',code:'Enter'});
  assert.equal(await evaluate(`document.querySelector('[role=combobox]').value`),'MPS SGR');
  await evaluate(`document.querySelector('[role=combobox]').click()`);await clear();
  await call('Input.dispatchKeyEvent',{type:'keyDown',key:'ArrowUp',code:'ArrowUp'});
  assert(await evaluate(`!!document.querySelector('[role=combobox]').getAttribute('aria-activedescendant')`));
  await call('Input.dispatchKeyEvent',{type:'keyDown',key:'Tab',code:'Tab'});
  assert(await evaluate(`!document.querySelector('[role=listbox]')&&document.activeElement!==document.querySelector('[role=combobox]')`),'normal Tab navigation');
  await evaluate(`document.querySelector('[role=combobox]').focus()`);
  await evaluate(`document.querySelector('input[required]').click();document.querySelector('input[required]').focus()`);
  assert.equal(await evaluate(`!!document.querySelector('[role=listbox]')`),false,'outside click closes');
  for(const width of [375,768,1440]){
   await call('Emulation.setDeviceMetricsOverride',{width,height:740,deviceScaleFactor:1,mobile:width<600});
   await evaluate(`document.querySelector('[role=combobox]').focus()`);await clear();await delay(100);
   const metrics=await evaluate(`(()=>{const p=document.querySelector('[role=listbox]'),d=document.querySelector('dialog'),r=p.getBoundingClientRect();return{left:r.left,right:r.right,viewport:innerWidth,scroll:p.scrollHeight,height:p.clientHeight,dialogScroll:d.scrollWidth,dialogWidth:d.clientWidth,pageScroll:document.documentElement.scrollWidth}})()`);
   assert(metrics.left>=0&&metrics.right<=metrics.viewport&&metrics.dialogScroll<=metrics.dialogWidth+1&&metrics.pageScroll<=metrics.viewport+1,JSON.stringify(metrics));
   assert(metrics.scroll>metrics.height&&metrics.height<=230,'internally scrolling result panel');
   const shot=await call('Page.captureScreenshot');await writeFile(resolve('../.tmp-gocache/workspace-combobox-'+width+'.png'),Buffer.from(shot.data,'base64'));
   await evaluate(`document.querySelector('dialog button[type=submit]').scrollIntoView({block:'center'})`);
   assert(await evaluate(`(()=>{const b=document.querySelector('dialog button[type=submit]').getBoundingClientRect();return b.top>=0&&b.bottom<=innerHeight})()`),'Save remains reachable');
  }
  await call('Emulation.setDeviceMetricsOverride',{width:375,height:740,deviceScaleFactor:1,mobile:true});
  await evaluate(`document.querySelector('[role=combobox]').focus()`);await clear();await search('mps sgr');
  const tap=await evaluate(`(()=>{const b=document.querySelector('[role=option] button');b.scrollIntoView({block:'center'});const r=b.getBoundingClientRect();return{x:r.x+r.width/2,y:r.y+r.height/2}})()`);
  await call('Input.dispatchTouchEvent',{type:'touchStart',touchPoints:[tap]});
  await call('Input.dispatchTouchEvent',{type:'touchEnd',touchPoints:[]});
  await waitFor(`document.querySelector('[role=combobox]').value==='MPS SGR'&&!document.querySelector('[role=listbox]')`);
  await evaluate(`document.querySelector('[role=combobox]').focus();document.querySelector('[role=combobox]').click()`);await clear();
  await evaluate(`Array.from(document.querySelectorAll('[role=option] button')).find(e=>e.textContent==='Tanpa project').click()`);
  await evaluate(`(()=>{const title=document.querySelector('input[required]');title.value='No project';title.dispatchEvent(new Event('input',{bubbles:true}));document.querySelector('dialog form').requestSubmit()})()`);
  await waitFor(`window.saved.length===2`);assert.equal(await evaluate(`window.saved[1].jobId`),null);
  // Activity presentation and comment form use the actual components with fixture APIs.
  await evaluate(`document.querySelector('.ws-tabs button').click()`);
  await waitFor(`document.querySelector('.ws-activity')?.textContent.includes('Belum ada aktivitas terbaru.')`);
  await evaluate(`window.activities=[
   {action:'CREATE',newValue:'MoM asistensi 20'},
   {action:'STATUS_CHANGE',oldValue:'TODO',newValue:'IN_PROGRESS'},
   {action:'STATUS_CHANGE',oldValue:'DONE',newValue:'NEED_REVISION'},
   {action:'ASSIGN',newValue:'internal-user-id'},
   {action:'FUTURE_EVENT',newValue:{secret:'hidden'}},
   {action:'CREATE',newValue:'<script>danger</script>'+('long'.repeat(100))}
  ].map((a,i)=>({id:String(i),actorName:'username'.repeat(30),createdAt:'2026-08-01T10:00:00Z',field:'internal_field',oldValue:null,newValue:null,...a}));document.querySelector('.ws-tabs button').click()`);
  await waitFor(`document.querySelector('.ws-activity')?.textContent.includes('MoM asistensi 20')`);
  const recentText=await evaluate(`document.querySelector('.ws-activity').textContent`);
  assert(recentText.includes('membuat pekerjaan "MoM asistensi 20"'));
  assert(recentText.includes('mengubah status dari Belum Dimulai menjadi Sedang Dikerjakan'));
  assert(recentText.includes('mengubah status dari Selesai menjadi Perlu Revisi'));
  assert(recentText.includes('menetapkan penanggung jawab')&&recentText.includes('memperbarui pekerjaan'));
  assert(!/CREATE|STATUS_CHANGE|IN_PROGRESS|ASSIGN|internal_field|internal-user-id|hidden|null/.test(recentText));
  assert(await evaluate(`!document.querySelector('.ws-activity script')`));
  await evaluate(`document.querySelector('.ws-panel .ws-link').click()`);
  await waitFor(`!!document.querySelector('.ws-comment-form')&&document.querySelector('dialog .ws-activity').textContent.includes('MoM asistensi 20')`);
  assert.equal(await evaluate(`document.querySelector('dialog .ws-activity').textContent`),recentText,'same formatter, old history rendered');
  assert(await evaluate(`document.querySelector('.ws-comment-form button').disabled`));
  for(const width of [320,360,390,414,768,1366]){
   await call('Emulation.setDeviceMetricsOverride',{width,height:900,deviceScaleFactor:1,mobile:width<600});
   for(const height of [96,240]){
    await evaluate(`document.querySelector('.ws-comment-form textarea').style.height='${height}px';document.querySelector('.ws-comment-form').scrollIntoView({block:'center'})`);
    const metrics=await evaluate(`(()=>{const f=document.querySelector('.ws-comment-form'),t=f.querySelector('textarea').getBoundingClientRect(),b=f.querySelector('button').getBoundingClientRect(),d=document.querySelector('dialog');return{gap:b.top-t.bottom,left:t.left,right:t.right,buttonRight:b.right,buttonHeight:b.height,viewport:innerWidth,dialogScroll:d.scrollWidth,dialogWidth:d.clientWidth,pageScroll:document.documentElement.scrollWidth}})()`);
    assert(metrics.gap>=12&&metrics.left>=0&&metrics.right<=width&&metrics.buttonRight<=width&&metrics.buttonHeight>=44&&metrics.dialogScroll<=metrics.dialogWidth+1&&metrics.pageScroll<=width+1,JSON.stringify(metrics));
   }
   const shot=await call('Page.captureScreenshot');await writeFile(resolve('../.tmp-gocache/workspace-comment-'+width+'.png'),Buffer.from(shot.data,'base64'));
  }
  await evaluate(`(()=>{const t=document.querySelector('.ws-comment-form textarea');t.value='Komentar uji';t.dispatchEvent(new Event('input',{bubbles:true}))})()`);
  await waitFor(`!document.querySelector('.ws-comment-form button').disabled`);
  await evaluate(`document.querySelector('.ws-comment-form').requestSubmit()`);
  await waitFor(`document.querySelector('.ws-comment')?.textContent.includes('Komentar uji')`);
  assert.equal(await evaluate(`window.comments[0].content`),'Komentar uji');
  assert(await evaluate(`document.querySelector('.ws-comment-form textarea').value===''&&document.querySelector('.ws-comment-form button').disabled`));
  await evaluate(`document.querySelector('dialog header button').click()`);
  for(const width of [320,360,390,414,768,1366]){
   await call('Emulation.setDeviceMetricsOverride',{width,height:900,deviceScaleFactor:1,mobile:width<600});
   assert(await evaluate(`document.documentElement.scrollWidth<=innerWidth+1&&Array.from(document.querySelectorAll('.ws-activity li, .ws-activity small')).every(e=>e.getBoundingClientRect().right<=innerWidth+1)`),'overview activity fits '+width);
  }
  console.log('PASS: shared activity labels, status labels, unknown fallback, raw payload safety, empty state, old history, comment submission, 320/360/390/414/768/1366 layouts and resized textarea.');
  // Verify assignment payload rendering through the existing workspace invalidation/refetch path.
  await call('Page.navigate',{url:'http://127.0.0.1:5190/dashboard'});
  await waitFor(`!!window.testSocket?.onmessage&&!!document.querySelector('.notification-button')`);
  const event={id:'assignment',type:'work_item_assigned',actor:{id:'manager',name:'Mas Jarwo'},entityType:'work_item',entityId:'task',createdAt:new Date().toISOString(),readAt:null,workspace:{title:'Review <script>progress</script>',message:'Anda ditugaskan ke pekerjaan baru',workspaceId:1,projectName:'MPS SGR',priority:'HIGH',startDate:'2026-09-09',dueDate:'2026-09-12'}};
  await evaluate(`window.feed=[${JSON.stringify(event)}];window.testSocket.onmessage({data:JSON.stringify({event:'workspace.changed',workspaceId:1})})`);
  await waitFor(`!!document.querySelector('.notification-badge')`);
  await evaluate(`document.querySelector('.notification-button').click()`);
  await waitFor(`document.querySelector('.notification-panel').textContent.includes('MPS SGR')`);
  assert(await evaluate(`document.querySelector('.notification-panel').textContent.includes('Mas Jarwo')&&!document.querySelector('.notification-panel script')`));
  assert.deepEqual(errors,[],'browser exceptions');
  console.log('PASS: ID dedup, substring/case/trim search stays open, ID save, keyboard/Escape/Tab/outside close, no-match/null option, 375/768/1440 layout, assignment feed rendering.');
} catch(error) { console.error(browserLog);throw error; } finally {
  socket?.close(); browser.kill();
  await delay(500);
  assert(profile.startsWith(join(tmpdir(),'workspace-browser-')));
  await rm(profile,{recursive:true,force:true,maxRetries:5,retryDelay:200});
}

