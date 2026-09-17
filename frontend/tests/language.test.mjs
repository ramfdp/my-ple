import test from 'node:test';
import assert from 'node:assert/strict';
import { messages } from '../src/lib/messages.ts';
import { readFileSync, readdirSync } from 'node:fs';
import { parse } from 'svelte/compiler';

test('both languages cover every static interface label', () => {
 for(const [key,pair] of Object.entries(messages)) {
  assert.equal(pair.length,2,key); assert.ok(pair.every(text=>typeof text==='string'&&text.trim()),key);
  assert.deepEqual([...pair[0].matchAll(/\{(\w+)\}/g)].map(m=>m[1]).sort(),[...pair[1].matchAll(/\{(\w+)\}/g)].map(m=>m[1]).sort(),key);
 }
 for(const file of readdirSync('src',{recursive:true}).filter(f=>f.endsWith('.svelte'))){
  const ast=parse(readFileSync('src/'+file,'utf8'),{modern:true});
  function walk(node,parents=[]){
   if(!node||typeof node!=='object')return;
   if(node.type==='Text'&&/[a-zA-Z]/.test(node.data)&&!parents.some(p=>['Attribute','StyleDirective'].includes(p.type))){
    assert.ok(['Bahasa Indonesia','English'].includes(node.data.trim()),file+': untranslated '+node.data.trim());
   }
   if(node.type==='CallExpression'&&node.callee?.name==='$translate'&&node.arguments[0]?.type==='Literal')assert.ok(Object.hasOwn(messages,node.arguments[0].value),file+': missing key '+node.arguments[0].value);
   for(const[k,v]of Object.entries(node)){if(['loc','metadata'].includes(k))continue;if(Array.isArray(v))v.forEach(n=>walk(n,[...parents,node]));else if(v&&typeof v==='object')walk(v,[...parents,node]);}
  }
  walk(ast.fragment);
 }
});

import { visibleFeatures } from '../src/lib/navigation.ts';
import { translate } from '../src/lib/i18n.ts';
test('menus use one permission-filtered feature list',()=>{
 for(const role of ['Admin','Management','HR','Finance','Project Engineer','IT','PMO','Business Development']){
  const items=visibleFeatures(role), paths=items.map(item=>item.href);
  assert.equal(new Set(paths).size,paths.length);
  assert.equal(paths.includes('/employees'),['Admin','Management','HR'].includes(role));
  assert.equal(paths.includes('/attendance/overview'),['Admin','Management','HR'].includes(role));
  assert.equal(paths.includes('/document-checklists'),['Admin','Management','Finance','Project Engineer'].includes(role));
  assert.equal(paths.includes('/leave/approvals'),['Admin','Management'].includes(role));
  assert.ok(paths.includes('/weekly-progress')&&paths.includes('/booking-zoom'));
  for(const item of items){assert.ok(messages[item.key]);assert.ok(messages[item.description]);}
 }
 assert.deepEqual(visibleFeatures(),[]);
});
test('translation preserves interpolation data and unknown content',()=>{
 assert.equal(translate('Hapus {name}?','en',{name:'Catatan'}),'Delete Catatan?');
 assert.equal(translate('Custom user content','en'),'Custom user content');
 assert.equal(translate('Save','id'),'Simpan');
 assert.equal(translate('constructor','en'),'constructor');
 assert.equal(translate('LOW','id'),'Rendah');
});
