import assert from 'node:assert/strict';
import test from 'node:test';
import {planPath, saveReviewAndRefresh} from '../src/lib/ai-planner.ts';

test('date selection queries the server, independent of history pagination', () => {
 assert.equal(planPath(1, undefined, '2026-09-01'), '/ai-plans/workspaces/1?date=2026-09-01');
 assert.equal(planPath(1, 'draft-id', '2026-09-01'), '/ai-plans/workspaces/1/draft-id');
});

test('confirmed write with failed refresh is not reported as full success', async () => {
 let writes = 0;
 const result = await saveReviewAndRefresh(async () => { writes++; }, async () => false);
 assert.equal(writes, 1);
 assert.equal(result, 'refresh-failed');
});

test('successful refresh confirms save; failed write never refreshes', async () => {
 assert.equal(await saveReviewAndRefresh(async () => {}, async () => true), 'saved');
 let refreshed = false;
 await assert.rejects(saveReviewAndRefresh(async () => { throw new Error('write failed'); }, async () => { refreshed = true; return true; }), /write failed/);
 assert.equal(refreshed, false);
});
