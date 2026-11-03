import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const component = readFileSync(new URL('../src/lib/components/NotificationBell.svelte', import.meta.url), 'utf8');
const styles = readFileSync(new URL('../src/lib/styles/dashboard.css', import.meta.url), 'utf8');

function notificationBell() {
 const script = component.match(/<script lang="ts">([\s\S]*?)<\/script>/)[1].replace(/^\s*import .*;$/gm, '');
 let effect, cleanup, nextFrame = 1;
 const frames = new Map();
 const context = vm.createContext({
  $state: value => value,
  $effect: callback => { effect = callback; },
  $notifications: {unread: 0},
  $locale: 'id-ID',
  $translate: value => value,
  tick: async () => {},
  onDestroy: callback => { cleanup = callback; },
  requestAnimationFrame: callback => { const id = nextFrame++; frames.set(id, callback); return id; },
  cancelAnimationFrame: id => frames.delete(id),
  window: {innerWidth: 1440, innerHeight: 900},
  Date, Intl
 });
 vm.runInContext(ts.transpile(script, {target: ts.ScriptTarget.ESNext}), context);
 return {
  run: code => vm.runInContext(code, context),
  update: async unread => { context.$notifications.unread = unread; effect(); await new Promise(resolve => setImmediate(resolve)); },
  flushFrame: () => { const callback = [...frames.values()].at(-1); frames.clear(); callback?.(); },
  cleanup: () => cleanup()
 };
}

test('badge animation replays when unread increases and the panel opens', async () => {
 const bell = notificationBell();
 await bell.update(0);
 await bell.update(1);
 assert.equal(bell.run('badgeOpen'), false);
 bell.flushFrame();
 assert.equal(bell.run('badgeOpen'), true);

 await bell.update(2);
 assert.equal(bell.run('badgeOpen'), false);
 bell.flushFrame();
 assert.equal(bell.run('badgeOpen'), true);

 bell.run("panel={matches:()=>false,togglePopover:()=>{}};toggle({currentTarget:{getBoundingClientRect:()=>({right:120,bottom:48})}})");
 await new Promise(resolve => setImmediate(resolve));
 assert.equal(bell.run('badgeOpen'), false);
 bell.flushFrame();
 assert.equal(bell.run('badgeOpen'), true);
 bell.cleanup();
});

test('badge hides at zero and preserves the remix motion contract', async () => {
 const bell = notificationBell();
 await bell.update(1);
 bell.flushFrame();
 await bell.update(0);
 assert.equal(bell.run('badgeOpen'), false);
 bell.cleanup();

 for (const value of ['260ms', '500ms', '400ms', '180ms', 'cubic-bezier(.22, 1, .36, 1)', 'cubic-bezier(.34, 1.36, .64, 1)', 'cubic-bezier(.4, 0, .2, 1)', 'translate(-8.2px, 12.4px)', 'blur(2px)', 'prefers-reduced-motion: reduce']) {
  assert.ok(styles.includes(value), `missing motion value: ${value}`);
 }
});
