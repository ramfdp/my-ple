import test from 'node:test';
import assert from 'node:assert/strict';
import {commentLinks} from '../src/lib/utils/comment-links.ts';
import {capitalizeReportDescription,groupReports} from '../src/lib/utils/report-table.ts';
import {subscribeChanges} from '../src/lib/realtime.ts';

test('weekly preview flattens the shared server hierarchy into one table',()=>{
 const report=(id)=>({id,userId:'u',primaryDivision:'Project Engineer',role:'',workDivision:'',work:'',number:99,version:1,description:id,completedAt:''});
 const groups=[
  {name:'Project Engineer',categories:[{name:'MK/Supervisi',jobs:[{key:'1',name:'PE job',pics:[{userId:'u',name:'PIC A',isActive:true}],progress:[{number:1,content:'a',report:report('a')},{number:2,content:'b',report:report('b')}]}]}]},
  {name:'Quality Assurance',categories:[{name:'QA 1',jobs:[{key:'2',name:'QA job',progress:[{number:1,content:'q',report:report('q')}]}]}]},
  {name:'CIHPK',categories:[{name:'MK/Supervisi',jobs:[{key:'3',name:'CIHPK job',progress:[{number:1,content:'c',report:report('c')}]}]}]}
 ];
 const rows=groupReports(groups);
 assert.deepEqual(rows.filter(r=>r.type==='work').map(r=>[r.role,r.category,r.work]),[
  ['Project Engineer','MK/Supervisi','PE job'],['Quality Assurance','QA 1','QA job'],['CIHPK','MK/Supervisi','CIHPK job']
 ]);
 assert.deepEqual(rows.filter(r=>r.type==='report').map(r=>[r.report.id,r.displayNumber]),[['a',1],['b',2],['q',1],['c',1]]);
 assert.equal(rows.length,7);
});

test('weekly descriptions capitalize the first letter of every line and bullet',()=>{
 assert.equal(
  capitalizeReportDescription('halo\n\n  halo\n- halo\n  - halo'),
  'Halo\n\n  Halo\n- Halo\n  - Halo'
 );
});

test('comment linkification retains text and only activates safe schemes',()=>{
 const text='See (https://example.test/a),\nhttp://example.test/wiki_(test). <img onerror=alert(1)> javascript:alert(1) data:text/html,x';
 const parts=commentLinks(text);
 assert.equal(parts.map(p=>p.text).join(''),text);
 assert.deepEqual(parts.filter(p=>p.href).map(p=>p.href),['https://example.test/a','http://example.test/wiki_(test)']);
});

test('shared subscription resynchronizes and removes listeners on navigation',async()=>{
 globalThis.window=new EventTarget();
 let calls=0,release;
 const stop=subscribeChanges(['workspace.changed'],async()=>{calls++;if(calls===1)await new Promise(r=>release=r)});
 window.dispatchEvent(new Event('workspace.changed'));
 window.dispatchEvent(new Event('workspace.changed'));
 window.dispatchEvent(new Event('realtime.resync'));
 release();await new Promise(r=>setImmediate(r));
 assert.equal(calls,2);
 stop();window.dispatchEvent(new Event('workspace.changed'));window.dispatchEvent(new Event('realtime.resync'));
 assert.equal(calls,2);
});

test('realtime retains distinct entities and continues after a failed refresh',async()=>{
 globalThis.window=new EventTarget();
 const calls=[];let release;
 const stop=subscribeChanges(['document-checklists.changed'],async change=>{
  calls.push(change.entityId);
  if(calls.length===1){await new Promise(r=>release=r);throw new Error('temporary failure')}
 },change=>change.entityId!=='unrelated');
 const send=(entityId,revision)=>window.dispatchEvent(new CustomEvent('document-checklists.changed',{detail:{entityId,revision}}));
 send('1',1);send('2',2);send('2',2);send('3',3);send('unrelated',4);
 release();await new Promise(r=>setImmediate(r));
 assert.deepEqual(calls,['1','2','3']);stop();
});
