import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {compile} from 'svelte/compiler';
import {render} from 'svelte/server';

const source=readFileSync(new URL('../src/lib/components/OvertimeSheetView.svelte',import.meta.url),'utf8');
const compiled=compile(source,{generate:'server',filename:'OvertimeSheetView.svelte'}).js.code;
const componentCode=compiled.replace("'svelte/internal/server'", JSON.stringify(import.meta.resolve('svelte/internal/server'))).replace("'$lib/overtime'", JSON.stringify(new URL('../src/lib/overtime.ts',import.meta.url).href));
const {default:Sheet}=await import(`data:text/javascript;base64,${Buffer.from(componentCode).toString('base64')}`);
const sheet={employeeName:'Test Employee',entries:[{workDate:'2026-10-07',startTime:'08:00',endTime:'21:00',dayType:'WORKDAY',project:'Test',description:'Long description',actualMinutes:123,weightedMinutes:456}],weightedMinutes:456,reviewedAt:null};

test('read-only sheet preserves stored results, complete calendar, placeholders and escaped content',()=>{
 const {body}=render(Sheet,{props:{sheet:{...sheet,employeeName:'<script>unsafe</script>'},period:'2026-10'}});
 assert.match(body,/&lt;script>unsafe&lt;\/script>/);
 assert.match(body,/2,05/);
 assert.match(body,/7,60/);
 assert.equal((body.match(/scope="row" title=/g)||[]).length,31);
 assert.match(body,/aria-label="Tidak tersedia"/);
 assert.doesNotMatch(body,/<input/);
});

test('editable sheet uses existing preview and supports short months',()=>{
 const row={...sheet.entries[0],workDate:'2024-02-01',startTime:'17:00',endTime:'19:30',overrideReason:''};
 const {body}=render(Sheet,{props:{sheet,period:'2024-02',editable:true,rows:{'2024-02-01':row}}});
 assert.equal((body.match(/scope="row" title=/g)||[]).length,29);
 assert.match(body,/3,00/);
 assert.match(body,/aria-label="Jam mulai 2024-02-01"/);
 assert.match(body,/value="WORKDAY"/);
});

test('payment columns and totals match Excel rules in editable and read-only sheets',()=>{
 for(const editable of [true,false]){
  for(const [endTime,dayType,actualMinutes,weightedMinutes,first,remaining] of [
   ['17:00','WORKDAY',0,0,'0,00','0,00'],
   ['17:29','WORKDAY',0,0,'0,00','0,00'],
   ['17:30','WORKDAY',60,60,'1,00','0,00'],
   ['18:00','WORKDAY',60,60,'1,00','0,00'],
   ['18:29','WORKDAY',60,60,'1,00','0,00'],
   ['18:30','WORKDAY',120,120,'1,00','1,00'],
   ['19:00','WORKDAY',120,120,'1,00','1,00'],
   ['21:00','WORKDAY',240,240,'1,00','3,00'],
   ['10:00','HOLIDAY',120,240,'0,00','0,00'],
   ['09:30','HOLIDAY',120,240,'0,00','0,00']
  ]){
   const row={...sheet.entries[0],startTime:'08:00',endTime,dayType,actualMinutes,weightedMinutes};
   const {body}=render(Sheet,{props:{sheet:{...sheet,entries:[row],weightedMinutes},period:'2026-10',editable,rows:{[row.workDate]:row}}});
   const dateRow=body.match(/<tr\b[^>]*>[^]*?<\/tr>/g)?.find(row=>row.includes('title="2026-10-07"'));
   assert.ok(dateRow);
   const calculated=[...dateRow.matchAll(/<td class="calculated[^\"]*">([^]*?)<\/td>/g)];
   assert.equal(calculated[2][1],first);
   assert.equal(calculated[3][1],remaining);
   const results=[...dateRow.matchAll(/<td class="result[^\"]*">([^]*?)<\/td>/g)];
   const expectedHours=minutes=>(minutes/60).toLocaleString('id-ID',{minimumFractionDigits:2,maximumFractionDigits:2});
   assert.equal(results[0][1],expectedHours(actualMinutes));
   assert.equal(results[1][1],expectedHours(weightedMinutes));
   const footer=body.match(/<tfoot\b[^>]*>([^]*?)<\/tfoot>/);
   assert.ok(footer);
   assert.equal(footer[1].match(/<td class="result[^\"]*">([^]*?)<\/td>/)?.[1],expectedHours(weightedMinutes));
  }
 }
});

test('decimal clock columns round at thirty minutes with two decimal places in editable and read-only sheets',()=>{
 for(const editable of [true,false]){
  for(const [startTime,endTime,startDecimal,endDecimal] of [
   ['08:00','19:30','8,00','20,00'],
   ['08:15','19:29','8,00','19,00'],
   ['08:29','19:15','8,00','19,00'],
   ['08:30','19:00','9,00','19,00'],
   ['08:59','19:59','9,00','20,00'],
   ['22:45','00:00','23,00','24,00'],
   ['00:00','23:45','24,00','24,00'],
   ['00:01','00:29','24,00','24,00'],
   ['00:29','00:01','24,00','24,00'],
   ['00:30','00:59','1,00','1,00'],
   ['23:30','23:59','24,00','24,00'],
   ['08:10','','8,00',''],
   ['','','',''],
   ['25:00','12:60','','']
  ]){
   const row={...sheet.entries[0],startTime,endTime};
   const {body}=render(Sheet,{props:{sheet:{...sheet,entries:[row]},period:'2026-10',editable,rows:{[row.workDate]:row}}});
   const dateRow=body.match(/<tr\b[^>]*>[^]*?<\/tr>/g)?.find(row=>row.includes('title="2026-10-07"'));
   assert.ok(dateRow);
   const cells=[...dateRow.matchAll(/<td class="calculated[^\"]*">([^]*?)<\/td>/g)];
   assert.equal(cells[0][1],startDecimal);
   assert.equal(cells[1][1],endDecimal);
  }
 }
});
