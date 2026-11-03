// Actual filesystem source loss and same-metadata changed replay, no route mocks.
const fs=require('fs'),path=require('path'),crypto=require('crypto'),assert=require('assert/strict');
const {chromium}=require(process.env.IMMICH_BROWSER_MODULE);
const hash=b=>crypto.createHash('sha256').update(b).digest('hex');
(async()=>{
 const browser=await chromium.launch({channel:'msedge',headless:true});
 try{
  const context=await browser.newContext(),page=await context.newPage();page.setDefaultTimeout(45000);const errors=[];page.on('pageerror',e=>errors.push(e.message));
  const source=process.env.IMMICH_IMPORT_SOURCE_FIXTURES;fs.mkdirSync(source,{recursive:true});
  const first=Buffer.from('earlier accepted original remains authoritative'),second=Buffer.alloc(8<<20,73);
  const a=path.join(source,'a-preserved.txt'),b=path.join(source,'b-unavailable.bin'),stamp=new Date(1700000000000);
  const write=(file,data)=>{fs.writeFileSync(file,data);fs.utimesSync(file,stamp,stamp)};write(a,first);write(b,second);
  await page.goto('http://127.0.0.1:2283/');await page.fill('#code',fs.readFileSync(path.join(process.env.IMMICH_TEST_STATE,'access-code.txt'),'utf8').trim());await page.click('#loginForm button');await page.waitForSelector('#library:not(.hidden)');await page.goto('http://127.0.0.1:2283/import');
  const api=async url=>{const r=await page.request.get('http://127.0.0.1:2283'+url);assert(r.ok());return r.json()};
  const paused=async()=>page.waitForFunction(()=>!busy&&document.querySelector('#counts').textContent.startsWith('paused')&&document.querySelector('#counts').textContent.includes('1 / 2'));
  await page.setInputFiles('#files',[a,b]);await page.waitForFunction(()=>!document.querySelector('#start').disabled);
  fs.unlinkSync(b);assert(!fs.existsSync(b),'The actual selected source must be absent');await page.click('#start');await paused();
  assert.match(await page.locator('#message').textContent(),/source became unavailable|Copy failed/i);
  const id=await page.evaluate(()=>localStorage.getItem('immich-import'));let saved=await api('/api/imports/'+id);
  assert.equal(saved.session.accepted_files,1);assert.equal(saved.session.accepted_bytes,first.length);assert.equal(saved.entries[0].asset_id,hash(first));assert(!saved.entries[1].asset_id);assert.equal(saved.background_paused,false);
  const download=await page.request.get('http://127.0.0.1:2283/api/assets/'+hash(first)+'/original');assert(download.ok());assert.deepEqual(await download.body(),first);
  // Reopen the page and physically restore the missing source. The accepted
  // file is changed without changing its size or timestamp: metadata must not
  // bypass the existing hash receipt.
  write(b,second);const changed=Buffer.from(first);changed[changed.length-1]^=1;write(a,changed);
  await page.reload();await paused();await page.setInputFiles('#files',[a,b]);await page.click('#resume');await paused();
  await page.getByText('Source size or content changed; original receipt retained. Start a new import for changed files.',{exact:true}).first().waitFor();
  assert.match(await page.locator('#message').textContent(),/Source size or content changed/);
  saved=await api('/api/imports/'+id);assert.equal(saved.session.accepted_files,1);assert.equal(saved.session.accepted_bytes,first.length);assert.equal(saved.entries[0].asset_id,hash(first));assert(!saved.entries[1].asset_id);
  const retained=await page.request.get('http://127.0.0.1:2283/api/assets/'+hash(first)+'/original');assert.deepEqual(await retained.body(),first);
  write(a,first);await page.setInputFiles('#files',[a,b]);await page.click('#resume');await page.waitForFunction(()=>document.querySelector('#counts').textContent.startsWith('completed')&&document.querySelector('#counts').textContent.includes('2 / 2'));
  saved=await api('/api/imports/'+id);assert.equal(saved.session.accepted_files,2);assert.equal(saved.session.new_files,2);assert.equal(saved.session.duplicates,0);assert.equal(saved.session.accepted_bytes,first.length+second.length);assert(saved.entries.every(e=>!e.error));assert.equal(saved.entries[1].asset_id,hash(second));
  const completed=await page.request.get('http://127.0.0.1:2283/api/assets/'+hash(second)+'/original');assert(completed.ok());assert.equal(hash(await completed.body()),hash(second));assert.deepEqual(fs.readFileSync(a),first);assert.deepEqual(fs.readFileSync(b),second);assert.equal(saved.background_paused,false);assert.deepEqual(errors,[]);
  await page.setViewportSize({width:390,height:844});assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
  console.log('PASS: real selected-file disappearance, page reopen, same-metadata changed-source rejection, exact receipt replay, resume, hashes and counters under LocalService');
 }finally{await browser.close()}
})().catch(e=>{console.error(e);process.exit(1)});
