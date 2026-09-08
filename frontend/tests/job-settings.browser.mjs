import assert from 'node:assert/strict';

// Uses the existing dependency-free Chromium runner in weekly-history.browser.test.mjs.
export async function testJobSettings({send,evaluate,waitFor,click,user,jobs,requests}) {
 const navigate=path=>send('Page.navigate',{url:`http://127.0.0.1:5190${path}`});
 const select=async(label,index)=>{await evaluate(`{const el=[...document.querySelectorAll('label')].find(l=>l.textContent.trim().startsWith(${JSON.stringify(label)})).querySelector('select');el.selectedIndex=${index};el.dispatchEvent(new Event('change',{bubbles:true}));}`);};
 await navigate('/settings/jobs');
 await waitFor("document.querySelector('tbody')?.textContent.includes('Archived Project')");
 assert.ok(await evaluate("!!document.querySelector('a[href=\"/settings/jobs\"]')"));
 await click('+ Tambah Pekerjaan');
 await evaluate("{const input=document.querySelector('.job-form input');input.value='Browser job';input.dispatchEvent(new Event('input',{bubbles:true}));}");
 await click('Simpan pekerjaan');
 await waitFor("document.querySelector('tbody')?.textContent.includes('Browser job')");
 await evaluate("[...document.querySelectorAll('tr')].find(r=>r.textContent.includes('Browser job')).querySelector('button[role=switch]').click()");
 await waitFor("document.querySelector('[role=status]')?.textContent.includes('Nonaktif')");
 assert.ok(await evaluate("document.querySelector('tbody').textContent.includes('Browser job')"),'deactivated row disappeared');
 await select('Role',2); await waitFor("document.body.textContent.includes('Belum ada pekerjaan pada kategori ini.')");
 await select('Role',1); await waitFor("document.querySelector('tbody')?.textContent.includes('Browser job')");
 await evaluate("[...document.querySelectorAll('tr')].find(r=>r.textContent.includes('Browser job')).querySelector('.edit-report').click()");
 await evaluate("{const input=document.querySelector('.job-form input');input.value='Edited job';input.dispatchEvent(new Event('input',{bubbles:true}));}");
 await click('Simpan pekerjaan'); await waitFor("document.querySelector('tbody')?.textContent.includes('Edited job')");
 await navigate('/weekly-progress/new');
 await waitFor("document.querySelectorAll('select').length===4 && !document.body.innerText.includes('Memuat pekerjaan')");
 assert.equal(await evaluate("[...document.querySelectorAll('option')].some(o=>o.textContent.includes('Edited job'))"),false,'inactive new option');
 assert.ok(requests.some(r=>r.startsWith('/jobs?')&&r.includes('active=true')));
 jobs[0].isActive=false;
 await navigate('/weekly-progress?division=Project%20Engineer');
 await waitFor("!!document.querySelector('.edit-report')");await click('Edit');
 await waitFor("[...document.querySelectorAll('option')].some(o=>o.textContent.includes('tersimpan'))");
 assert.equal(await evaluate("document.querySelector('.report-submit').disabled"),false,'inactive historical edit blocked');
 await navigate('/settings/jobs');await waitFor("!!document.querySelector('.delete-report')");
 await evaluate("document.querySelector('.delete-report').click()");await waitFor("!!document.querySelector('dialog[open]')");
 await click('Hapus pekerjaan');await waitFor("document.querySelector('dialog').textContent.includes('Nonaktifkan pekerjaan ini.')");
 await click('Batal');
 await evaluate("[...document.querySelectorAll('tr')].find(r=>r.textContent.includes('Edited job')).querySelector('.delete-report').click()");
 await click('Hapus pekerjaan');await waitFor("!document.querySelector('dialog[open]') && !document.querySelector('tbody')?.textContent.includes('Edited job')");
 await send('Emulation.setDeviceMetricsOverride',{width:375,height:812,deviceScaleFactor:1,mobile:true});
 assert.ok(await evaluate('document.documentElement.scrollWidth <= innerWidth'),'mobile page overflow');
 await select('Divisi',1);await waitFor("document.body.textContent.includes('Belum ada pekerjaan pada kategori ini.')");
 await send('Emulation.clearDeviceMetricsOverride');
 for(const role of ['Admin','Management','HR','Project Engineer','Finance','IT','Business Development','PMO']) {
  user.role=role;await navigate('/settings/jobs');await waitFor("!!document.querySelector('.delete-report')");
  assert.equal(await evaluate("!!document.querySelector('a[href=\"/settings/jobs\"]')"),true,`${role} sidebar`);
  await click('+ Tambah Pekerjaan');await waitFor("!!document.querySelector('.job-form')");
 }
 user.role='Management';
 console.log('PASS: job settings filters, create, edit, toggle, safe deletion, active dropdowns, inactive history, RBAC UI, mobile layout');
}
