// Real LocalService/Edge acceptance using synthetic originals only.
const fs=require('fs'),path=require('path'),assert=require('assert/strict'),crypto=require('crypto');
const {chromium}=require(process.env.IMMICH_BROWSER_MODULE);
(async()=>{
 const browser=await chromium.launch({channel:'msedge',headless:true});
 try {
  const page=await browser.newPage({viewport:{width:1280,height:900}});page.setDefaultTimeout(30000);
  const errors=[];page.on('pageerror',e=>errors.push(e.message));
  await page.goto('http://127.0.0.1:2283');await page.fill('#code',fs.readFileSync(path.join(process.env.IMMICH_TEST_STATE,'access-code.txt'),'utf8').trim());await page.click('#loginForm button');await page.waitForSelector('#library:not(.hidden)');
  const png=Buffer.from(await page.evaluate(()=>{const c=document.createElement('canvas');c.width=c.height=32;c.getContext('2d').fillRect(0,0,32,32);return c.toDataURL('image/png').split(',')[1]}),'base64');
  const post=async(u,data)=>{const r=await page.request.post('http://127.0.0.1:2283'+u,{headers:{'X-Immich-Request':'1'},data});assert(r.ok(),u+' '+r.status());return r.json()};
  const originals=[];
  for(let i=0;i<65;i++){
   const bytes=Buffer.concat([png,Buffer.from('gallery-search-'+i)]),r=await post('/api/upload?name=search-'+String(i).padStart(3,'0')+'.png',bytes);originals.push({id:r.asset.id,bytes});
   await post('/api/assets/'+r.asset.id+'/tags',{tags:['SearchTrip']});
  }
  for(const [i,names] of [[0,['Alice']],[1,['Alice','Bob']],[64,['Bob']]])for(const [j,name] of names.entries())await post('/api/assets/'+originals[i].id+'/manual-face',{box:[.1+j*.3,.1,.2,.2],name});
  console.log('CHECK: fixture uploads complete');
  // Let previews settle before asserting revision-bound search pagination.
  await page.waitForFunction(async()=>{const p=await(await fetch('/api/gallery?sort=name&visibility=library&limit=100')).json();return p.assets.length===65&&p.assets.every(a=>a.preview!=='queued')});
  const search=async(q,count)=>{console.log('CHECK: released synthetic search',q,count);await page.fill('#galleryQuery',q);await page.locator('#gallerySearch button[type=submit]').click();await page.waitForFunction(({q,count})=>document.querySelector('#gallerySearchState').textContent.includes(count+' matching')&&document.querySelector('#galleryQuery').value===q,{q,count});assert.equal(new URL(page.url()).pathname,'/');};
  await search('ALICE',2);assert.equal(await page.locator('.card').count(),2);
  await search('Bob',2);assert.equal(await page.locator('.card').count(),2);
  await search('search-064',1);assert.equal(await page.locator('.card').getAttribute('data-asset'),originals[64].id);
  assert.match(await page.locator('.card .meta').textContent(),/Date imported:/);assert.doesNotMatch(await page.locator('.card .meta').textContent(),/zone|UTC|upload date/);
  await page.locator('.card .openMedia').click();assert.match(await page.locator('#viewDetails').textContent(),/Capture time zone: Not read/);await page.click('#closeViewer');assert.equal(await page.inputValue('#galleryQuery'),'search-064');
  await search('SearchTrip',65);assert.equal(await page.locator('.card').count(),60);const first=await page.locator('.card').evaluateAll(cs=>cs.map(c=>c.dataset.asset));
  await page.click('#nextPage');try{await page.waitForFunction(()=>document.querySelector('#pageLabel').textContent.startsWith('Page 2 ·'));}catch(e){console.log('CHECK: paging failure',await page.evaluate(()=>({status:document.querySelector('#status').textContent,label:document.querySelector('#pageLabel').textContent,query:galleryContext.query.toString(),hasAfter:!!galleryContext.after,hasNext:!!galleryContext.next})),errors);if(process.env.IMMICH_BROWSER_OUTPUT)await page.screenshot({path:path.join(process.env.IMMICH_BROWSER_OUTPUT,'gallery-search-failure.png')});throw e;}assert.equal(await page.locator('.card').count(),5);
  const second=await page.locator('.card').evaluateAll(cs=>cs.map(c=>c.dataset.asset));assert.equal(new Set([...first,...second]).size,65);
  const scroll=await page.evaluate(()=>{scrollTo(0,document.body.scrollHeight);return scrollY});await page.locator('.openMedia').last().click();await page.click('#closeViewer');assert.match(await page.locator('#pageLabel').textContent(),/^Page 2/);assert.equal(await page.inputValue('#galleryQuery'),'SearchTrip');assert(Math.abs(await page.evaluate(()=>scrollY)-scroll)<10);
  await page.click('#previousPage');await page.waitForFunction(()=>document.querySelector('#pageLabel').textContent.startsWith('Page 1 ·'));assert.deepEqual(await page.locator('.card').evaluateAll(cs=>cs.map(c=>c.dataset.asset)),first);
  await search('no-such-person',0);assert.equal(await page.locator('.card').count(),0);assert.match(await page.locator('#status').textContent(),/No matching media/);
  await page.click('#clearGallerySearch');await page.waitForFunction(()=>document.querySelectorAll('.card').length===60&&document.querySelector('#gallerySearchState').textContent==='');
  await post('/api/assets/'+originals[1].id+'/hide',{});await search('Alice',1);await page.selectOption('#visibility','hidden');await page.waitForFunction(()=>document.querySelector('#gallerySearchState').textContent.includes('1 matching item')&&document.querySelector('#gallerySearchState').textContent.includes('Hidden Media'));assert.equal(await page.locator('.card').getAttribute('data-asset'),originals[1].id);
  await page.reload();await page.waitForFunction(()=>document.querySelector('#gallerySearchState').textContent.includes('1 matching item'));assert.equal(await page.inputValue('#galleryQuery'),'Alice');assert.equal(await page.inputValue('#visibility'),'hidden');
  await page.route('**/api/gallery?**',route=>route.fulfill({status:500,body:'unavailable'}));await page.fill('#galleryQuery','new-query');await page.locator('#gallerySearch button[type=submit]').click();await page.waitForFunction(()=>document.querySelector('#status').textContent.includes('unavailable'));assert.equal(await page.locator('.card').count(),0);await page.unroute('**/api/gallery?**');
  await search('Alice',1);
  for(const width of [1280,390]){await page.setViewportSize({width,height:900});assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);if(process.env.IMMICH_BROWSER_OUTPUT)await page.screenshot({path:path.join(process.env.IMMICH_BROWSER_OUTPUT,'gallery-search-'+width+'.png'),fullPage:true});}
  for(const original of originals){const r=await page.request.get('http://127.0.0.1:2283/api/assets/'+original.id+'/original');assert(r.ok());assert.equal(crypto.createHash('sha256').update(await r.body()).digest('hex'),original.id);}
  assert.deepEqual(errors,[]);console.log('PASS: gallery assigned multi-person/filename/tag search, 65-item filtered paging, viewer return/scroll, clear/empty/error, hidden/reload, dates, desktop/narrow and original hashes');
 }finally{await browser.close()}
})().catch(e=>{console.error(e.stack);process.exit(1)});
