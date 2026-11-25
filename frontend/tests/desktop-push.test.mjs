import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import vm from 'node:vm';
import ts from 'typescript';

test('desktop banners use a stable tag, a category-only preview, and an internal destination', async () => {
 const handlers = {};
 let banner, opened;
 const origin = 'https://my-ple.example';
 const self = {
  location:{origin},
  addEventListener: (name, fn) => handlers[name] = fn,
  registration:{showNotification: async (title, options) => {banner={title,...options};}},
  clients:{matchAll: async () => [],openWindow: async url => {opened=url;}}
 };
 vm.runInNewContext(readFileSync(new URL('../static/desktop-push.js',import.meta.url),'utf8'),{self, URL});
 for (const [kind, path] of Object.entries({
  attendance:'/attendance',zoom_booking_created:'/booking-zoom',workspace_task:'/workspace/12',work_item_assigned:'/workspace/12',
  leave_request_created:'/leave/approvals?request=abc',leave_request_approved:'/leave?request=abc',leave_request_rejected:'/leave?request=abc',
  overtime_submitted:'/overtime/approvals',overtime_approved:'/overtime',overtime_rejected:'/overtime',future_type:'/dashboard'
 })) {
  let done;
  handlers.push({data:{json:()=>({kind,id:'key',entity:'abc',workspace:'12',message:'private details'})},waitUntil:p=>done=p});
  await done;
  assert.equal(banner.title,'My PLE');
  assert.equal(banner.tag,'key');
  assert.ok(banner.body.length>0 && !banner.body.includes('private details'));
  handlers.notificationclick({notification:{...banner,close(){}},waitUntil:p=>done=p});
  await done;
  assert.equal(opened,origin+path);
 }
});

test('desktop opt-in follows the browser profile and authenticated user', async () => {
 const source = readFileSync(new URL('../src/lib/desktop-push.ts',import.meta.url),'utf8')
  .replace(/^import .*;$/gm,'').replace(/^export /gm,'');
 let state, subscription, user='alice', requests=0, prompts=0, unsubscribed=0;
 const owner = new Map();
 const permission={permission:'default',async requestPermission(){prompts++;return this.permission='granted';}};
 const reg={getNotifications:async()=>[], pushManager:{
  getSubscription:async()=>subscription,
  subscribe:async options=>(subscription={options:{applicationServerKey:options.applicationServerKey.buffer},endpoint:'device',
   toJSON:()=>({endpoint:'device'}),unsubscribe:async()=>{unsubscribed++;subscription=null;return true;}})
 }};
 const context=vm.createContext({
  window:{isSecureContext:true,PushManager:{},Notification:permission},Notification:permission,
  navigator:{serviceWorker:{register:async()=>reg,ready:Promise.resolve(reg),getRegistration:async()=>reg}},
  localStorage:{getItem:k=>owner.get(k),setItem:(k,v)=>owner.set(k,v),removeItem:k=>owner.delete(k)},
  atob,writable:()=>({set:value=>state=value}),session:()=>({user:{id:user}}),
  api:async(_path,opts)=>{if(!opts)return {publicKey:'AQ'};requests++;if(opts.method==='PUT')assert.equal(JSON.parse(opts.body).userId,user);}
 });
 vm.runInContext(ts.transpile(source,{target:ts.ScriptTarget.ESNext}),context);
 const run=code=>vm.runInContext(code,context);
 await run("restoreDesktopPush('alice')");
 assert.deepEqual([state,prompts,requests],['off',0,0]);
 await run('enableDesktopPush()');
 assert.deepEqual([state,prompts,requests],['on',1,1]);
 await run('detachDesktopPush(true)');
 await run("restoreDesktopPush('alice')");
 assert.deepEqual([state,prompts,requests],['on',1,2]);
 user='bob';
 await run("restoreDesktopPush('bob')");
 assert.deepEqual([state,unsubscribed,requests],['off',2,2]);
 permission.permission='denied';
 await run("restoreDesktopPush('bob')");
 assert.deepEqual([state,prompts],['blocked',1]);
});

