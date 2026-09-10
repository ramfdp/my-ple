import test from 'node:test';
import assert from 'node:assert/strict';
import {commentLinks} from '../src/lib/utils/comment-links.ts';
import {groupReports} from '../src/lib/utils/report-table.ts';
import {subscribeChanges} from '../src/lib/realtime.ts';

test('display numbers close gaps within each work group without changing IDs',()=>{
 const base={primaryDivision:'PE',role:'PE',workDivision:'MK supervisi',work:'Bridge'};
 const source=[{...base,id:'c',number:7},{...base,id:'b',number:2},{...base,id:'d',work:'Road',number:9}];
 const rows=groupReports(source).filter(r=>r.type==='report');
 assert.deepEqual(rows.map(r=>[r.report.id,r.displayNumber]),[['b',1],['c',2],['d',1]]);
 assert.deepEqual(source.map(r=>r.number),[7,2,9]);
 assert.deepEqual(groupReports(source.filter(r=>r.id!=='b')).filter(r=>r.type==='report').map(r=>r.displayNumber),[1,1]);
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
