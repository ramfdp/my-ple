import assert from 'node:assert/strict';

export async function testWeeklyEntry({send,evaluate,waitFor,click,scenario}) {
 await send('Page.navigate',{url:'http://127.0.0.1:5190/weekly-progress/new'});
 await waitFor("document.querySelectorAll('select').length===4 && !document.body.innerText.includes('Memuat pekerjaan')");
 const choose=async(label,value)=>{await evaluate(`{const el=[...document.querySelectorAll('label')].find(l=>l.textContent.trim().startsWith(${JSON.stringify(label)})).querySelector('select');el.value=${JSON.stringify(String(value))};el.dispatchEvent(new Event('change',{bubbles:true}));}`);};
 const type=text=>evaluate(`{const el=document.querySelector('textarea');el.value=${JSON.stringify(text)};el.dispatchEvent(new Event('input',{bubbles:true}));}`);
 const context=()=>evaluate("[...document.querySelectorAll('select')].map(el=>el.value)");
 await choose('Pekerjaan',1);
 const initial=await context();
 scenario.hold=true;
 await type('First continuous entry');await click('Simpan laporan');
 await waitFor("document.querySelector('button[type=submit]').disabled");
 await evaluate("document.querySelector('form').requestSubmit();document.querySelector('form').requestSubmit()");
 for(let i=0;i<100&&!scenario.release;i++)await new Promise(resolve=>setTimeout(resolve,50));
 assert.equal(scenario.posts.length,1,'duplicate requests while pending');
 assert.equal(await evaluate("document.querySelector('textarea').value"),'First continuous entry','cleared before confirmation');
 scenario.hold=false;scenario.release();
 await waitFor("document.querySelector('textarea')?.value==='' && document.activeElement===document.querySelector('textarea')");
 assert.equal(await evaluate('location.pathname'),'/weekly-progress/new');
 assert.deepEqual(await context(),initial);
 assert.ok(await evaluate("document.body.innerText.includes('Laporan No. 41 berhasil ditambahkan.')"));
 await type('Second continuous entry');await click('Simpan laporan');
 await waitFor("document.querySelector('textarea')?.value==='' && document.body.innerText.includes('No. 42')");
 assert.deepEqual(await context(),initial);
 scenario.failNext=true;await type('Do not lose this');await click('Simpan laporan');
 await waitFor("document.body.innerText.includes('Test save failure') && !document.querySelector('button[type=submit]').disabled");
 assert.equal(await evaluate("document.querySelector('textarea').value"),'Do not lose this');
 assert.deepEqual(await context(),initial);
 await click('Simpan laporan');await waitFor("document.querySelector('textarea')?.value==='' && document.activeElement===document.querySelector('textarea')");
 // The retained context remains editable; parents reset incompatible children.
 await choose('Pekerjaan',6);assert.equal((await context()).at(-1),'6');
 await choose('Kategori',3);await waitFor("document.querySelector('select:last-of-type') && [...document.querySelectorAll('select')].at(-1).value==='3'");
 await choose('Role',2);await waitFor("[...document.querySelectorAll('select')].at(-1).value==='4'");
 await choose('Divisi','AI Dev');await waitFor("document.querySelectorAll('select').length===2 && [...document.querySelectorAll('select')].at(-1).value==='5'");
 await type('Different division');await click('Simpan laporan');await waitFor("document.querySelector('textarea')?.value===''");
 const sent=scenario.posts.at(-1);assert.equal(sent.jobId,5);assert.equal(sent.role,'');assert.equal(sent.workDivision,'');
 assert.ok(scenario.posts.every(p=>!('number' in p)&&!('sequenceNumber' in p)),'frontend assigned numbers');
 await evaluate("document.querySelector('.back-link').click()");await waitFor("location.pathname==='/weekly-progress'");
 console.log('PASS: continuous entry, retained IDs, focus, failure/retry, double-submit guard, editable dependent selectors, manual back');
}
