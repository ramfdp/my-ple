import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

test('employee month rollover preserves entries, manual selection and in-flight requests',()=>{
 const source=readFileSync(new URL('../src/routes/overtime/+page.svelte',import.meta.url),'utf8');
 const callback=source.match(/watchOvertimePeriod\(current=>\{([\s\S]*?)\}\);/)[1];
 for(const guard of ['manualPeriod','busy','loading','hasEntries',null]){
  let loads=0;
  const context=vm.createContext({manualPeriod:false,busy:false,loading:false,hasEntries:false,period:'2026-10',current:'2026-11',load:()=>{loads++}});
  if(guard)context[guard]=true;
  vm.runInContext(`(()=>{${callback}})()`,context);
  assert.equal(context.period,guard?'2026-10':'2026-11');
  assert.equal(loads,guard?0:1);
 }
});

function approvalPage() {
 const source = readFileSync(new URL('../src/routes/overtime/approvals/+page.svelte', import.meta.url), 'utf8');
 const script = source.match(/<script lang="ts">([\s\S]*?)<\/script>/)[1].replace(/^\s*import .*;$/gm, '');
 let refresh, refreshPeriod, respond = async () => ({ employees: [{ userId: 'new-employee' }] });
 const context = vm.createContext({
  $state: value => value, $derived: value => value,
  $currentUser: { id: 'reviewer', role: 'Finance' },
  defaultOvertimePeriod: () => '2026-10', URLSearchParams, Intl,
  watchOvertimePeriod: callback => { refreshPeriod=callback; return ()=>{}; },
  api: (...args) => respond(...args), onMount: callback => callback(),
  subscribeChanges: (_events, callback) => { refresh = callback; }
 });
 vm.runInContext(ts.transpile(script, { target: ts.ScriptTarget.ESNext }), context);
 return {
  run: code => vm.runInContext(code, context),
  refresh: () => refresh(),
  refreshPeriod: period => refreshPeriod(period),
  respond: callback => { respond = callback; }
 };
}

test('month rollover updates approvals but preserves manual periods and open reviews',async()=>{
 const page=approvalPage();
 await page.run('load()');
 page.run('offset=50');
 page.refreshPeriod('2026-11');
 assert.equal(page.run('period'),'2026-11');
 assert.equal(page.run('offset'),0);
 await page.run('load()');
 page.run("selected={id:'sheet-1'}");
 page.refreshPeriod('2026-12');
 assert.equal(page.run('period'),'2026-11');
 page.run('selected=null;manualPeriod=true');
 page.refreshPeriod('2026-12');
 assert.equal(page.run('period'),'2026-11');
});

test('realtime refresh preserves detail and restores the note only for the same sheet', async () => {
 const page = approvalPage();
 await page.run('load()');
 page.run("selected={id:'sheet-1',status:'SUBMITTED'};updateReviewerNote('Mohon periksa jam lembur');");
 let complete;
 page.respond(() => new Promise(resolve => { complete = resolve; }));
 const refreshing = page.refresh();
 assert.equal(page.run('loading'), false);
 assert.equal(page.run('selected.id'), 'sheet-1');
 complete({ employees: [{ userId: 'another-employee' }] });
 await refreshing;
 assert.equal(page.run('employees[0].userId'), 'another-employee');
 assert.equal(page.run('selected.id'), 'sheet-1');
 assert.equal(page.run('reviewerNote'), 'Mohon periksa jam lembur');
 page.respond(async url => ({ sheet: { id: url.split('/').at(-1), status: 'SUBMITTED' } }));
 await page.run("open({sheetId:'sheet-2',status:'SUBMITTED'})");
 assert.equal(page.run('reviewerNote'), '');
 page.run("updateReviewerNote('Catatan pengajuan lain');selected=null;");
 await page.run("open({sheetId:'sheet-1',status:'SUBMITTED'})");
 assert.equal(page.run('reviewerNote'), 'Mohon periksa jam lembur');
 page.run("updateReviewerNote('');selected=null;");
 await page.run("open({sheetId:'sheet-1',status:'SUBMITTED'})");
 assert.equal(page.run('reviewerNote'), '');
});

test('decision failure keeps the note and successful decision clears its draft', async () => {
 const page = approvalPage();
 await page.run('load()');
 page.run("selected={id:'sheet-1',status:'SUBMITTED'};updateReviewerNote('Periksa kembali');");
 page.respond(async () => { throw new Error('Gagal menyimpan'); });
 await page.run("decide('REJECTED')");
 assert.equal(page.run("reviewerNoteDrafts.get('sheet-1')"), 'Periksa kembali');
 let submitted;
 page.respond(async (_url, options) => {
  if (options) {
   submitted = JSON.parse(options.body);
   return { sheet: { id: 'sheet-1', status: 'REJECTED' } };
  }
  return { employees: [] };
 });
 await page.run("decide('REJECTED')");
 assert.deepEqual(submitted, { status: 'REJECTED', reviewerNote: 'Periksa kembali' });
 assert.equal(page.run("reviewerNoteDrafts.has('sheet-1')"), false);
});

test('explicit filter reload still closes the previous review', async () => {
 const page = approvalPage();
 await page.run('load()');
 page.run("selected={id:'sheet-1'};");
 const loading = page.run('load()');
 assert.equal(page.run('loading'), true);
 assert.equal(page.run('selected'), null);
 await loading;
 assert.equal(page.run('loading'), false);
});
