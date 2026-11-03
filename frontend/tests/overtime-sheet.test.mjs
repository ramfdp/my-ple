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
 assert.match(body,/2,1/);
 assert.match(body,/7,6/);
 assert.equal((body.match(/scope="row" title=/g)||[]).length,31);
 assert.match(body,/aria-label="Tidak tersedia"/);
 assert.doesNotMatch(body,/<input/);
});

test('editable sheet uses existing preview and supports short months',()=>{
 const row={...sheet.entries[0],workDate:'2024-02-01',startTime:'17:00',endTime:'19:30',overrideReason:''};
 const {body}=render(Sheet,{props:{sheet,period:'2024-02',editable:true,rows:{'2024-02-01':row}}});
 assert.equal((body.match(/scope="row" title=/g)||[]).length,29);
 assert.match(body,/1,5/);
 assert.match(body,/aria-label="Jam mulai 2024-02-01"/);
 assert.match(body,/value="WORKDAY"/);
});

test('decimal columns follow clock inputs in editable and read-only sheets',()=>{
 for(const editable of [true,false]){
  for(const [startTime,endTime,startDecimal,endDecimal] of [
   ['18:15','19:30','18,25','19,5'],
   ['22:45','00:00','22,75','0'],
   ['08:10','','8,17',''],
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
