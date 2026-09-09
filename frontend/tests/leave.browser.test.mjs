// Run Vite on 127.0.0.1:5190, then node tests/leave.browser.test.mjs.
// Uses an isolated Chromium profile and fixture APIs; never sends email.
import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { setTimeout as delay } from 'node:timers/promises';

const profile = await mkdtemp(join(tmpdir(), 'leave-browser-'));
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
   window.soundCalls=[];window.blockAudio=false;
   const nativePlay=HTMLMediaElement.prototype.play;
   HTMLMediaElement.prototype.play=function(){window.soundCalls.push({muted:this.muted,src:this.src,volume:this.volume});if(window.blockAudio)return Promise.reject(new DOMException('Blocked','NotAllowedError'));return nativePlay.call(this)};
   window.role=sessionStorage.getItem('fixtureRole')||'Project Engineer';
   window.user={id:window.role==='Admin'?'admin':window.role==='Management'?'manager':'employee',name:'Budi',role:window.role,isActive:true};
   window.rows=[{id:'pending',userId:'employee',employeeName:'Budi '+('LongName'.repeat(30)),division:'Project Engineer',requestType:'CUTI',startDate:'2026-09-15',endDate:'2026-09-17',reason:'Keperluan keluarga '+('long'.repeat(150)),notes:'Catatan',status:'PENDING',reviewedBy:null,reviewedAt:null,reviewerName:'',reviewerNote:'',createdAt:'2026-09-09T07:45:00Z',updatedAt:'2026-09-09T07:45:00Z'},
   {id:'own',userId:'admin',employeeName:'Admin',division:'',requestType:'IZIN',startDate:'2026-09-18',endDate:'2026-09-18',reason:'Keperluan pribadi',notes:'',status:'PENDING',reviewedBy:null,reviewedAt:null,reviewerName:'',reviewerNote:'',createdAt:'2026-09-09T07:00:00Z',updatedAt:'2026-09-09T07:00:00Z'}];
   window.saved=[];window.decisions=[];window.sockets=[];
   window.event=(id,type='zoom_booking_created')=>({id,type,actor:{id:'someone',name:'Reviewer'},entityId:'pending',entityType:type.startsWith('leave_')?'leave_request':'zoom_booking',createdAt:new Date().toISOString(),readAt:null,...(type.startsWith('leave_')?{leave:{...window.rows[0],title:'Pengajuan Cuti Disetujui',message:'Pengajuan Anda Disetujui oleh Reviewer'}}:{booking:{zoomName:'Zoom Meeting',startAt:new Date().toISOString(),endAt:new Date().toISOString()}})});
   window.feed=[window.event('history')];
   const realFetch=window.fetch;
   window.fetch=async(input,options)=>{
    const url=new URL(typeof input==='string'?input:input.url,location.href);
    if(url.port!=='3000')return realFetch(input,options);
    let body={};let status=200;
    if(url.pathname==='/auth/me')body={user:window.user};
    if(url.pathname==='/notifications')body={notifications:window.feed};
    if(url.pathname==='/notifications/unread-count')body={count:window.feed.length};
    if(url.pathname.startsWith('/leave-requests')){
     const parts=url.pathname.split('/').filter(Boolean),review=parts[1]==='approvals';if(review)parts.splice(1,1);
     if(parts.length===1&&options?.method==='POST'){const data=JSON.parse(options.body);window.saved.push(data);const row={...window.rows[0],...data,id:'created',employeeName:'Budi',userId:window.user.id};window.rows.unshift(row);body={request:row}}
     else if(parts.length===1){let rows=window.rows.filter(r=>review||r.userId===window.user.id);if(url.searchParams.get('status'))rows=rows.filter(r=>r.status===url.searchParams.get('status'));if(url.searchParams.get('type'))rows=rows.filter(r=>r.requestType===url.searchParams.get('type'));body={requests:rows}}
     else{const row=window.rows.find(r=>r.id===parts[1]);if(parts[2]==='decision'){const data=JSON.parse(options.body);window.decisions.push(data);Object.assign(row,{...data,reviewedBy:window.user.id,reviewerName:'Reviewer',reviewedAt:new Date().toISOString()});if(window.conflict){row.status='APPROVED';window.conflict=false;status=409;body={error:'Pengajuan sudah diproses. Muat ulang data.'}}else body={request:row}}else body={request:row}}
    }
    return new Response(JSON.stringify(body),{status,headers:{'Content-Type':'application/json'}});
   };
   window.WebSocket=class{constructor(){window.sockets.push(this);window.testSocket=this;setTimeout(()=>this.onopen?.(),20)}close(){} };
  `});
  await call('Emulation.setDeviceMetricsOverride',{width:320,height:900,deviceScaleFactor:1,mobile:true});
  await call('Page.navigate',{url:'http://127.0.0.1:5190/leave'});
  await waitFor(`!!window.testSocket?.onmessage&&!!document.querySelector('.leave-card')`);
  assert.equal(await evaluate(`window.soundCalls.length`),0,'no historical sound');
  assert(!await evaluate(`!!document.querySelector('a[href="/leave/approvals"]')`),'approval navigation hidden');
  // No interaction: accepted notification renders, but no audio attempt.
  await evaluate(`window.testSocket.onmessage({data:JSON.stringify({event:'notification.created',notification:window.event('before-unlock')})})`);
  assert.equal(await evaluate(`window.soundCalls.length`),0);
  // Real gesture unlock: playback rejection is handled, a later gesture can retry.
  await evaluate(`window.blockAudio=true`);
  await call('Input.dispatchKeyEvent',{type:'keyDown',key:'Shift',code:'ShiftLeft'});
  await delay(100);
  assert.equal(await evaluate(`window.soundCalls.filter(c=>!c.muted).length`),0);
  await evaluate(`window.blockAudio=false`);
  await call('Input.dispatchKeyEvent',{type:'keyDown',key:'Shift',code:'ShiftLeft'});
  await delay(250);
  await evaluate(`window.testSocket.onmessage({data:JSON.stringify({event:'notification.created',notification:window.event('new-zoom')})})`);
  await delay(100);
  assert.equal(await evaluate(`window.soundCalls.filter(c=>!c.muted).length`),1,'one sound after unlock');
  await evaluate(`for(let i=0;i<3;i++)window.testSocket.onmessage({data:JSON.stringify({event:'notification.created',notification:window.event('new-zoom')})})`);
  await evaluate(`document.querySelector('.notification-button').click()`);
  assert.equal(await evaluate(`window.soundCalls.filter(c=>!c.muted).length`),1,'no replay or dropdown sound');
  assert(await evaluate(`document.querySelector('.notification-panel').textContent.includes('Zoom Meeting')`));
  await evaluate(`document.querySelector('.notification-panel').hidePopover()`);
  await delay(850);
  await evaluate(`window.feed.push(window.event('leave-decision','leave_request_approved'));window.testSocket.onmessage({data:JSON.stringify({event:'leave.changed'})})`);
  await waitFor(`window.soundCalls.filter(c=>!c.muted).length===2`);
  await evaluate(`window.testSocket.onmessage({data:JSON.stringify({event:'leave.changed'})})`);
  await delay(100);
  assert.equal(await evaluate(`window.soundCalls.filter(c=>!c.muted).length`),2);
  await evaluate(`window.socketCountBefore=window.sockets.length;window.testSocket.onclose()`);
  await waitFor(`window.sockets.length>window.socketCountBefore`);
  await delay(100);
  await evaluate(`window.testSocket.onmessage({data:JSON.stringify({event:'notification.created',notification:window.event('new-zoom')})})`);
  assert.equal(await evaluate(`window.soundCalls.filter(c=>!c.muted).length`),2,'reconnect is silent');
  await delay(850);
  await evaluate(`window.feed.push({...window.event('assignment'),type:'work_item_assigned',workspace:{title:'Task',workspaceId:1,message:'Anda ditugaskan'}});window.testSocket.onmessage({data:JSON.stringify({event:'workspace.changed',workspaceId:1})})`);
  await waitFor(`window.soundCalls.filter(c=>!c.muted).length===3`);
  await delay(850);
  await evaluate(`window.feed.push(window.event('batch1'),window.event('batch2'));window.testSocket.onmessage({data:JSON.stringify({event:'workspace.changed',workspaceId:1})});window.testSocket.onmessage({data:JSON.stringify({event:'workspace.changed',workspaceId:1})})`);
  await waitFor(`window.soundCalls.filter(c=>!c.muted).length===4`);
  await delay(100);
  assert.equal(await evaluate(`window.soundCalls.filter(c=>!c.muted).length`),4,'batch coalesced');
  assert(await evaluate(`window.soundCalls.every(c=>c.volume>0&&c.volume<0.5&&new URL(c.src).origin===location.origin)`));
  const setValue=async(selector,value)=>evaluate(`(()=>{const e=document.querySelector(${JSON.stringify(selector)});e.value=${JSON.stringify(value)};e.dispatchEvent(new Event('input',{bubbles:true}));e.dispatchEvent(new Event('change',{bubbles:true}))})()`);
  await evaluate(`Array.from(document.querySelectorAll('main button')).find(b=>b.textContent.includes('Ajukan Izin')).click()`);
  await waitFor(`!!document.querySelector('dialog[open] form')`);
  const widths=[320,360,390,414,768,1024,1366,1920];
  async function layout(name){
   for(const width of widths){
    await call('Emulation.setDeviceMetricsOverride',{width,height:900,deviceScaleFactor:1,mobile:width<600});
    await delay(150);
    await evaluate(`new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve)))`);
    const metrics=await evaluate(`(()=>{const d=document.querySelector('dialog[open]');document.body.getBoundingClientRect();return {page:document.documentElement.scrollWidth,viewport:innerWidth,dialog:d?.scrollWidth,client:d?.clientWidth,controls:Array.from((d||document).querySelectorAll('input,select,textarea,.leave-actions button')).every(e=>{const r=e.getBoundingClientRect();return r.left>=0&&r.right<=innerWidth+1})}})()`);
    assert(metrics.page<=width+1&&(!metrics.dialog||metrics.dialog<=metrics.client+1)&&metrics.controls,JSON.stringify({name,width,...metrics}));
    if(name==='form'||name==='decision'){
     await evaluate(`document.querySelector('dialog .leave-actions').scrollIntoView({block:'center'})`);
     assert(await evaluate(`Array.from(document.querySelectorAll('dialog .leave-actions button')).every(e=>{const r=e.getBoundingClientRect();return r.top>=0&&r.bottom<=innerHeight+1&&r.height>=44&&r.left>=0&&r.right<=innerWidth})`),'actions reachable '+width);
    }
    if(width===320||width===1366){const shot=await call('Page.captureScreenshot');await writeFile(resolve('../.tmp-gocache/leave-'+name+'-'+width+'.png'),Buffer.from(shot.data,'base64'))}
   }
  }
  await layout('form');
  await setValue('dialog select','IZIN');
  await setValue('dialog input[type=date]','2026-09-15');
  await setValue('dialog label:nth-of-type(3) input','2026-09-14');
  await setValue('dialog textarea','Keperluan pribadi');
  await evaluate(`document.querySelector('dialog form').requestSubmit()`);
  assert.equal(await evaluate(`window.saved.length`),0,'invalid date blocked');
  await setValue('dialog label:nth-of-type(3) input','2026-09-17');
  await evaluate(`document.querySelector('dialog form').requestSubmit()`);
  await waitFor(`window.saved.length===1&&!document.querySelector('dialog[open]')`);
  assert(await evaluate(`window.saved[0].requestType==='IZIN'&&!!window.saved[0].requestId&&!('userId' in window.saved[0])`));
  await layout('employee');
  await evaluate(`document.querySelector('.leave-open').click()`);
  await waitFor(`!!document.querySelector('dialog[open] dl')`);
  assert(!await evaluate(`Array.from(document.querySelectorAll('dialog button')).some(b=>b.textContent==='Setujui'||b.textContent==='Tolak')`),'no employee decision controls');
  await evaluate(`sessionStorage.setItem('fixtureRole','Admin')`);
  await call('Page.navigate',{url:'http://127.0.0.1:5190/leave/approvals'});
  await waitFor(`!!document.querySelector('.leave-card')&&!!document.querySelector('a[href="/leave/approvals"]')`);
  await layout('approvals');
  await evaluate(`document.querySelector('.leave-open').click()`);
  await waitFor(`!!document.querySelector('dialog textarea')`);
  await layout('decision');
  await setValue('dialog textarea','Disetujui sesuai jadwal');
  await evaluate(`document.querySelector('dialog form').requestSubmit()`);
  await waitFor(`window.decisions.length===1&&document.querySelector('dialog').textContent.includes('Tanggal Keputusan')`);
  assert(await evaluate(`window.decisions[0].status==='APPROVED'&&window.decisions[0].reviewerNote==='Disetujui sesuai jadwal'`));
  await evaluate(`document.querySelector('dialog header button').click();document.querySelector('.leave-open').click()`);
  await waitFor(`document.querySelector('dialog').textContent.includes('Pengajuan Anda harus ditinjau')`);
  assert(!await evaluate(`!!document.querySelector('dialog textarea')`),'self review controls absent');
  await evaluate(`sessionStorage.setItem('fixtureRole','Management')`);
  await call('Page.navigate',{url:'http://127.0.0.1:5190/leave/approvals?request=pending'});
  await waitFor(`!!document.querySelector('dialog[open] textarea')`);
  await setValue('dialog textarea','Agenda project');
  await evaluate(`Array.from(document.querySelectorAll('dialog button')).find(b=>b.textContent==='Tolak').click()`);
  await waitFor(`window.decisions.length===1&&document.querySelector('dialog').textContent.includes('Ditolak')`);
  assert.equal(await evaluate(`window.decisions[0].reviewerNote`),'Agenda project');
  await evaluate(`document.querySelector('dialog header button').click();document.querySelector('.leave-open').click()`);
  await waitFor(`!!document.querySelector('dialog textarea')`);
  await evaluate(`window.conflict=true;document.querySelector('dialog form').requestSubmit()`);
  await waitFor(`document.querySelector('dialog [role=alert]')?.textContent.includes('sudah diproses')`);
  assert(await evaluate(`document.querySelector('dialog').textContent.includes('Disetujui')&&!document.querySelector('dialog textarea')`),'stale decision refreshes authoritative state');
  await evaluate(`document.querySelector('dialog header button').click()`);
  await setValue('.ws-filters select','');
  await evaluate(`document.querySelector('.ws-filters').requestSubmit()`);
  await waitFor(`document.querySelector('.leave-cards').textContent.includes('Disetujui')&&document.querySelector('.leave-cards').textContent.includes('Ditolak')`);
  await setValue('.ws-filters select','PENDING');
  await evaluate(`document.querySelector('.ws-filters').requestSubmit()`);
  await waitFor(`document.querySelector('.ws-empty')?.textContent.includes('Tidak ada pengajuan yang menunggu persetujuan.')`);
  await call('Page.navigate',{url:'http://127.0.0.1:5190/leave'});
  await waitFor(`document.querySelector('.ws-empty')?.textContent.includes('Belum ada pengajuan izin atau cuti.')`);
  await evaluate(`sessionStorage.setItem('fixtureRole','Project Engineer')`);
  await call('Page.navigate',{url:'http://127.0.0.1:5190/leave/approvals'});
  await waitFor(`document.querySelector('[role=alert]')?.textContent.includes('hanya tersedia untuk Admin dan Management')`);
  assert(!await evaluate(`!!document.querySelector('.leave-open')`));
  assert.deepEqual(errors,[],'browser exceptions');
  console.log('PASS: employee submission/date validation, Admin approval, Management rejection, self-review UI, stale conflict refresh, all 8 widths; global Zoom/workspace/leave sound, historical silence, replay/reconnect dedup, batch coalescing, blocked-audio recovery.');
} catch(error) { console.error(browserLog);throw error; } finally {
  socket?.close(); browser.kill();
  await delay(500);
  assert(profile.startsWith(join(tmpdir(),'leave-browser-')));
  await rm(profile,{recursive:true,force:true,maxRetries:5,retryDelay:200});
}

