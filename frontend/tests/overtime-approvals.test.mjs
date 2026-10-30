import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

// Exercise the page's handlers with mocked API and lifecycle dependencies.
function approvalPage() {
 const source = readFileSync(new URL('../src/routes/overtime/approvals/+page.svelte', import.meta.url), 'utf8');
 const script = source.match(/<script lang="ts">([\s\S]*?)<\/script>/)[1].replace(/^\s*import .*;$/gm, '');
 let refresh, respond = async () => ({ employees: [{ userId: 'new-employee' }] });
 const context = vm.createContext({
  $state: value => value, $derived: value => value,
  $currentUser: { id: 'reviewer', role: 'Finance' },
  defaultOvertimePeriod: () => '2026-10', URLSearchParams, Intl,
  api: (...args) => respond(...args), onMount: callback => callback(),
  subscribeChanges: (_events, callback) => { refresh = callback; }
 });
 vm.runInContext(ts.transpile(script, { target: ts.ScriptTarget.ESNext }), context);
 return {
  run: code => vm.runInContext(code, context),
  refresh: () => refresh(),
  respond: callback => { respond = callback; }
 };
}

test('realtime refresh closes detail and restores the note only for the same sheet', async () => {
 const page = approvalPage();
 await page.run('load()');
 page.run("selected={id:'sheet-1',status:'SUBMITTED'};updateReviewerNote('Mohon periksa jam lembur');");
 let complete;
 page.respond(() => new Promise(resolve => { complete = resolve; }));
 const refreshing = page.refresh();
 assert.equal(page.run('loading'), true);
 assert.equal(page.run('selected'), null);
 complete({ employees: [{ userId: 'another-employee' }] });
 await refreshing;
 assert.equal(page.run('employees[0].userId'), 'another-employee');
 assert.equal(page.run('selected'), null);
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
