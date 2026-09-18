// VITE_API_URL=http://localhost:3000 npm run dev -- --port 5188 --host 127.0.0.1
import assert from 'node:assert/strict';
const {chromium}=await import('../../graphify-out/responsive-tools/node_modules/playwright/index.mjs');
const browser=await chromium.launch({executablePath:process.env.CHROME_PATH??'C:/Program Files/Google/Chrome/Application/chrome.exe',args:['--disable-gpu','--in-process-gpu']});
const origin='http://127.0.0.1:5188',now=new Date().toISOString();
const job={id:1,name:'Pekerjaan PDF',division:'Project Engineer',isActive:true};
const doc={id:1,name:'BAST',status:'OK',updatedAt:now};
const stage={id:1,jobId:1,name:'Tahap Pertama',periodLabel:'',version:1,updatedAt:now,notes:'Catatan panjang\n'.repeat(80)+'AKHIR CATATAN',readiness:'BELUM SIAP',documents:[doc],milestones:[{id:1,name:'Sub A',notes:'',documents:[{...doc,id:2,name:'BAPP',status:'BELUM ADA'}]},{id:2,name:'Sub B',notes:'',documents:[]}]};
const checklist={id:1,title:'Checklist PDF',version:1,requireDigitalSignature:true,signers:[{id:1,type:'internal',name:'Penanda Tangan',position:'Manager',archived:false}],items:[{id:1,documentName:'Dokumen <aman>',documentPosition:'Kantor',sentDate:'2026-09-18',notes:'Catatan lengkap',digitalSigned:true,signatures:{1:false}}]};
try {
 const page=await browser.newPage({reducedMotion:'reduce'});
 await page.route('**://localhost:3000/**',route=>{
  const path=new URL(route.request().url()).pathname;
  const fixtures={'/auth/me':{user:{id:'u1',name:'Penguji',role:'Admin',language:'id'}},'/billing-documents':{jobs:[job],stages:[stage,{...stage,id:2,name:'TAHAP DISEMBUNYIKAN',readiness:'SIAP'}]},'/document-checklists':{jobs:[job]},'/document-checklists/1':{checklists:[checklist]}};
  return route.fulfill({status:200,contentType:'application/json',headers:{'Access-Control-Allow-Origin':origin,'Access-Control-Allow-Credentials':'true','Access-Control-Allow-Headers':'Content-Type'},body:JSON.stringify(fixtures[path]??{})});
 });
 // Keep print documents open for assertions instead of opening the OS print dialog.
 await page.addInitScript(()=>{
  const descriptor=Object.getOwnPropertyDescriptor(HTMLIFrameElement.prototype,'contentWindow');
  Object.defineProperty(HTMLIFrameElement.prototype,'contentWindow',{...descriptor,get(){const win=descriptor.get.call(this);if(win)win.print=()=>{};return win;}});
 });
 async function print(name){
  await page.getByRole('button',{name:'Cetak / Simpan PDF',exact:true}).click();
  const frame=page.frameLocator('iframe[data-table-print]');
  await frame.locator('table').waitFor();
  assert.equal(await frame.locator('button, input, select, textarea').count(),0);
  const html=await frame.locator('html').evaluate(el=>el.outerHTML);
  const preview=await browser.newPage();
  await preview.setContent(html);
  await preview.pdf({path:`../graphify-out/${name}.pdf`,preferCSSPageSize:true,printBackground:true});
  await preview.close();
  return frame;
 }
 await page.goto(origin+'/billing-documents');
 await page.locator('.billing-table').waitFor();
 await page.getByLabel('Kesiapan tahap').selectOption('BELUM SIAP');
 const billing=await print('billing-website');
 assert.equal(await billing.locator('tbody tr').count(),2);
 assert.equal(await billing.getByText('TAHAP DISEMBUNYIKAN').count(),0);
 assert.equal(await billing.locator('td[rowspan="2"] span').filter({hasText:/^OK$/}).count(),1);
 assert.equal(await billing.getByText('OK',{exact:true}).evaluate(el=>getComputedStyle(el).backgroundColor),'rgb(17, 120, 79)');
 assert.ok((await billing.locator('body').textContent()).includes('AKHIR CATATAN'));
 assert.equal(await billing.locator('thead th').first().getAttribute('colspan'),'3');
 await page.goto(origin+'/document-checklists');
 await page.locator('.job-heading').click();
 await page.locator('.table-scroll table').waitFor();
 const check=await print('checklist-website');
 assert.equal(await check.getByText('Dokumen <aman>',{exact:true}).count(),1);
 assert.equal(await check.getByText('☑',{exact:true}).count(),1);
 assert.equal(await check.getByText('☐',{exact:true}).count(),1);
 assert.ok((await check.locator('body').textContent()).includes('2026-09-18'));
 assert.equal(await check.getByText('Penanda Tangan',{exact:false}).count(),1);
 console.log('PASS website PDFs: filters, merged cells, colours, full notes, checkboxes, dates and escaped text');
} finally { await browser.close(); }
