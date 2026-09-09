// Run Vite on 127.0.0.1:5190, then node tests/zoom.browser.test.mjs.
// Uses an isolated Chromium profile and fixture APIs; never sends email.
import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { setTimeout as delay } from 'node:timers/promises';

const profile = await mkdtemp(join(tmpdir(), 'zoom-browser-'));
const browser = spawn(process.env.CHROME_PATH || 'C:/Program Files/Google/Chrome/Application/chrome.exe', ['--headless=new', '--disable-gpu', '--in-process-gpu', '--no-first-run', '--no-default-browser-check', '--remote-debugging-port=0', `--user-data-dir=${profile}`, 'about:blank'], { windowsHide: true, stdio: ['ignore','ignore','pipe'] });
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
  await call('Page.addScriptToEvaluateOnNewDocument', { source: `
    window.testRole = new URLSearchParams(location.search).get('role') || 'PMO'; window.copied = [];
    Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText: async text => window.copied.push(text) } });
    window.resources = ['NP','MRP','IB2'].map((code,i) => ({ id:i+1, name:'Zoom '+(i+1), code, meetingUrl:'https://zoom.us/j/'+(i+1)+'?pwd='+('x'.repeat(160)), passcode:'123', meetingId:'1', hostKey:'', description:'Room information' }));
    const realFetch = window.fetch;
    window.fetch = async (input, options) => {
      const url = new URL(typeof input==='string'?input:input.url, location.href);
      if (url.port!=='3000') return realFetch(input,options);
      let body = {};
      if (url.pathname==='/auth/me') body={user:{id:'creator',name:'Budi',role:window.testRole,email:'test@example.com',isActive:true}};
      if (url.pathname==='/zoom-resources') body={resources:window.resources};
      if (url.pathname==='/zoom-bookings') body={bookings:[]};
      if (url.pathname==='/notifications') body={notifications:[]};
      if (url.pathname==='/notifications/unread-count') body={count:0};
      return new Response(JSON.stringify(body),{status:200,headers:{'Content-Type':'application/json'}});
    };
    window.WebSocket = class { constructor(){window.testSocket=this;setTimeout(()=>this.onopen?.(),20);} close(){} };
  ` });
  await call('Page.navigate', { url: 'http://127.0.0.1:5190/booking-zoom' });
  await waitFor(`document.querySelectorAll('.copy-actions button').length===3`);
  assert.equal(await evaluate(`document.querySelectorAll('.zoom-info-heading button').length`), 0, 'view-only user must not get edit controls');
  for (const width of [375, 768, 1440]) {
    await call('Emulation.setDeviceMetricsOverride', { width, height: 900, deviceScaleFactor: 1, mobile: width < 600 });
    for (let i = 0; i < 3; i++) {
      await evaluate(`document.querySelectorAll('.copy-actions button')[${i}].click()`);
      await waitFor(`document.querySelectorAll('.copy-actions span')[${i}].textContent==='Tersalin!'`);
      assert.equal(await evaluate(`window.copied.at(-1)`), await evaluate(`window.resources[${i}].meetingUrl`));
    }
    assert(await evaluate(`Array.from(document.querySelectorAll('.zoom-info-item')).every(item=>{const link=item.querySelector('dd a').getBoundingClientRect(),button=item.querySelector('.copy-actions button').getBoundingClientRect(),section=item.getBoundingClientRect();return button.top>=link.bottom && button.right<=section.right && link.right<=section.right && section.right<=innerWidth;})`), `sidebar overlaps at ${width}px`);
    if (width===375) { const shot=await call('Page.captureScreenshot',{captureBeyondViewport:true});await writeFile(resolve('../.tmp-gocache/zoom-mobile.png'),Buffer.from(shot.data,'base64')); }
  }
  await delay(2100);
  assert.equal(await evaluate(`document.querySelector('.copy-actions span').textContent`), '');
  await evaluate(`navigator.clipboard.writeText=async()=>{throw new Error('denied')};document.querySelector('.copy-actions button').click()`);
  await waitFor(`document.querySelector('.copy-actions span').textContent==='Link tidak dapat disalin.'`);
  assert(await evaluate(`document.querySelector('.zoom-info-item dd a').textContent.startsWith('https://zoom.us/')`));
  await evaluate(`Object.defineProperty(navigator,'clipboard',{value:undefined});document.execCommand=command=>{window.fallbackCopy=document.querySelector('textarea').value;return command==='copy'};document.querySelector('.copy-actions button').click()`);
  await waitFor(`window.fallbackCopy===window.resources[0].meetingUrl`);
  await evaluate(`document.execCommand=()=>false;document.querySelector('.copy-actions button').click()`);
  await waitFor(`document.querySelector('.copy-actions span').textContent==='Link tidak dapat disalin.'`);
  for (const role of ['Admin','Management']) {
    await call('Page.navigate',{url:'http://127.0.0.1:5190/booking-zoom?role='+role});
    await waitFor(`document.querySelectorAll('.copy-actions button').length===3`);
    assert.equal(await evaluate(`document.querySelectorAll('.zoom-info-heading button').length`),3,role+' retains edit controls');
    await evaluate(`document.querySelector('.copy-actions button').click()`);
    await waitFor(`window.copied.length===1`);
    assert.equal(await evaluate(`document.querySelectorAll('.zoom-modal').length`),0,'copy must not open editor');
  }
  await call('Page.navigate', { url: 'http://127.0.0.1:5190/dashboard' });
  await waitFor(`!!window.testSocket?.onmessage && !!document.querySelector('.notification-button')`);
  const notification = { id:'event1', type:'zoom_booking_created', actor:{id:'other',name:'Sari'}, entityType:'zoom_booking', entityId:'booking1', createdAt:new Date().toISOString(), readAt:null, booking:{id:'booking1',zoomName:'Zoom 2 - MRP',startAt:'2026-09-09T03:00:00Z',endAt:'2026-09-09T04:00:00Z',description:'Coordination '+ 'x'.repeat(950)} };
  await evaluate(`window.testSocket.onmessage({data:JSON.stringify({event:'notification.created',notification:${JSON.stringify(notification)}})})`);
  await evaluate(`document.querySelector('.notification-button').click()`);
  await waitFor(`!!document.querySelector('.booking-description')`);
  assert((await evaluate(`document.querySelector('.booking-description').textContent`)).includes(notification.booking.description));
  await evaluate(`window.testSocket.onmessage({data:JSON.stringify({event:'notification.created',notification:${JSON.stringify({...notification,id:'self',actor:{id:'creator',name:'Budi'}})}})})`);
  assert.equal(await evaluate(`document.querySelectorAll('.notification-panel li').length`), 1, 'self event must be excluded');
  for (const width of [375,1440]) {
    await call('Emulation.setDeviceMetricsOverride',{width,height:900,deviceScaleFactor:1,mobile:width<600});
    await delay(100);
    await evaluate(`if(!document.querySelector('.notification-panel').matches(':popover-open'))document.querySelector('.notification-button').click()`);
    const metrics=await evaluate(`(()=>{const p=document.querySelector('.notification-panel'),d=document.querySelector('.booking-description');return {scroll:p.scrollWidth,width:p.clientWidth,right:p.getBoundingClientRect().right,viewport:innerWidth,height:d.clientHeight,lineHeight:parseFloat(getComputedStyle(d).lineHeight)||parseFloat(getComputedStyle(d).fontSize)*1.5}})()`);
    assert(metrics.scroll<=metrics.width+1 && metrics.right<=metrics.viewport && metrics.height<=3*parseFloat(metrics.lineHeight)+1, `description overflow at ${width}: ${JSON.stringify(metrics)}`);
    const shot=await call('Page.captureScreenshot');await writeFile(resolve('../.tmp-gocache/zoom-notification-'+width+'.png'),Buffer.from(shot.data,'base64'));
  }
  assert.deepEqual(errors,[],'browser exceptions');
  console.log('PASS: three URLs, view-only copy, feedback/reset, failure/fallback, 375/768/1440 layouts, WebSocket description and creator exclusion.');
} catch(error) { console.error(browserLog);throw error; } finally {
  socket?.close(); browser.kill();
  await delay(500);
  assert(profile.startsWith(join(tmpdir(),'zoom-browser-')));
  await rm(profile,{recursive:true,force:true,maxRetries:5,retryDelay:200});
}

