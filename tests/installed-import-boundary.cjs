// Actual installed application, real disk-backed File selection and streamed hash readback.
const fs=require('fs'),path=require('path'),crypto=require('crypto'),http=require('http'),assert=require('assert/strict');
const {chromium}=require(process.env.IMMICH_BROWSER_MODULE);
const limit=4294967296,watchdog=setTimeout(()=>{console.error('Installed boundary phase exceeded ten-minute bound');process.exit(1)},600000);watchdog.unref();
const hashFile=async file=>{const h=crypto.createHash('sha256');let count=0;for await(const b of fs.createReadStream(file)){h.update(b);count+=b.length}return{hash:h.digest('hex'),count}};
(async()=>{
 const root=process.env.IMMICH_INSTALLED_BOUNDARY_ROOT,state=path.join(root,'state'),source=path.join(root,'source'),receipt=path.join(root,'boundary.json'),phase=process.argv[2];
 const small=path.join(source,'a-preserved.txt'),large=path.join(source,'b-limit.bin'),tiny=Buffer.from('installed boundary earlier accepted original');
 let createdHash;
 if(phase==='start'){
  fs.mkdirSync(source);fs.writeFileSync(small,tiny);const block=Buffer.alloc(65536);for(let i=0;i<block.length;i++)block[i]=(i*29+17)%251;
  const h=crypto.createHash('sha256'),fd=fs.openSync(large,'wx');let written=0;
  try{while(written<limit){const n=fs.writeSync(fd,block,0,Math.min(block.length,limit-written));assert(n>0);h.update(block.subarray(0,n));written+=n}assert.equal(fs.writeSync(fd,Buffer.from([1])),1);fs.fsyncSync(fd)}finally{fs.closeSync(fd)}
  createdHash=h.digest('hex');assert.equal(fs.statSync(large).size,limit+1);console.log('Prepared actual 4 GiB + 1 byte source with bounded 64 KiB generator');
 }
 const browser=await chromium.launch({channel:'msedge',headless:true});
 try{
  const context=await browser.newContext(),page=await context.newPage();page.setDefaultTimeout(120000);const errors=[];page.on('pageerror',e=>errors.push(e.message));
  await page.goto('http://127.0.0.1:2283/');await page.fill('#code',fs.readFileSync(path.join(state,'access-code.txt'),'utf8').trim());await page.click('#loginForm button');await page.waitForSelector('#library:not(.hidden)');await page.goto('http://127.0.0.1:2283/import');
  const get=async url=>{const r=await page.request.get('http://127.0.0.1:2283'+url);assert(r.ok());return r.json()};
  if(phase==='start'){
   await page.setInputFiles('#files',large);await page.locator('#selection').filter({hasText:'Selection rejected'}).waitFor();assert(await page.locator('#start').isDisabled());assert.equal((await get('/api/imports')).length,0);
   await page.setInputFiles('#files',[]);fs.truncateSync(large,limit);assert.equal(fs.statSync(large).size,limit);await page.setInputFiles('#files',[small,large]);assert(!(await page.locator('#start').isDisabled()));
   const selected=await page.locator('#files').evaluate(e=>Array.from(e.files).map(f=>f.size));assert.deepEqual(selected,[tiny.length,limit]);
   const cdp=await context.newCDPSession(page);await cdp.send('Network.enable');await cdp.send('Network.emulateNetworkConditions',{offline:false,latency:0,downloadThroughput:10<<20,uploadThroughput:2<<20});
   await page.click('#start');const until=Date.now()+90000;let partial=false;
   while(Date.now()<until){if(fs.readdirSync(path.join(state,'staging')).some(n=>{try{return n.endsWith('.part')&&fs.statSync(path.join(state,'staging',n)).size>=131072}catch{return false}})){partial=true;break}await new Promise(r=>setTimeout(r,50))}
   assert(partial,'Actual partial upload was not observed');const id=await page.evaluate(()=>localStorage.getItem('immich-import')),saved=await get('/api/imports/'+id);assert.equal(saved.session.accepted_files,1);assert.equal(saved.session.accepted_bytes,tiny.length);assert(!saved.entries[1].asset_id);
   fs.writeFileSync(receipt,JSON.stringify({id,hash:createdHash,smallHash:crypto.createHash('sha256').update(tiny).digest('hex')}));fs.writeFileSync(path.join(root,'crash-ready'),'ready');
   await page.waitForFunction(()=>!busy);assert.match(await page.locator('#message').textContent(),/unavailable|interrupted|failed|incomplete/i);console.log('PASS: installed picker rejects +1 byte, selects exact 4 GiB, preserves first receipt and observes actual partial staging before forced termination');
  }else if(phase==='finish'){
   const baseline=JSON.parse(fs.readFileSync(receipt)),{id,hash,smallHash}=baseline;let saved=await get('/api/imports/'+id);assert.equal(saved.session.status,'paused');assert.equal(saved.session.accepted_files,1);assert.equal(saved.session.accepted_bytes,tiny.length);assert.equal(saved.entries[0].asset_id,smallHash);assert(!saved.entries[1].asset_id);assert.equal(saved.background_paused,false);assert.deepEqual(fs.readdirSync(path.join(state,'staging')),[]);
   await page.selectOption('#history',id);await page.click('#open');await page.waitForFunction(()=>document.querySelector('#counts').textContent.startsWith('paused'));await page.setInputFiles('#files',[small,large]);await page.click('#resume');
   await page.waitForFunction(()=>!busy&&document.querySelector('#counts').textContent.startsWith('completed')&&document.querySelector('#counts').textContent.includes('2 / 2'),null,{timeout:300000});
   saved=await get('/api/imports/'+id);assert.equal(saved.session.accepted_files,2);assert.equal(saved.session.accepted_bytes,limit+tiny.length);assert.equal(saved.session.new_files,2);assert.equal(saved.session.duplicates,0);assert.equal(saved.entries[1].asset_id,hash);assert(saved.entries.every(e=>!e.error));assert.equal(saved.background_paused,false);assert.equal(fs.statSync(path.join(state,'originals',hash)).size,limit);
   const cookies=await context.cookies('http://127.0.0.1:2283');const cookie=cookies.map(c=>c.name+'='+c.value).join(';');
   const downloaded=await new Promise((resolve,reject)=>{const digest=crypto.createHash('sha256');let bytes=0;const req=http.get('http://127.0.0.1:2283/api/assets/'+hash+'/original',{headers:{Cookie:cookie}},res=>{if(res.statusCode!==200){res.resume();reject(Error('Original download status '+res.statusCode));return}res.on('data',b=>{digest.update(b);bytes+=b.length});res.on('error',reject);res.on('end',()=>resolve({hash:digest.digest('hex'),count:bytes}))});req.setTimeout(120000,()=>req.destroy(Error('Original readback idle timeout')));req.on('error',reject)});
   assert.deepEqual(downloaded,{hash,count:limit});assert.deepEqual(await hashFile(large),{hash,count:limit});assert.deepEqual(fs.readFileSync(small),tiny);assert.deepEqual(fs.readdirSync(path.join(state,'staging')),[]);assert.deepEqual(errors,[]);
   console.log('PASS: actual installed LocalService/Edge 4 GiB transfer after restart, exact receipts/counters, full streamed original readback and source SHA256 preservation');
  }else throw Error('Unknown phase');assert.deepEqual(errors,[]);
 }finally{await browser.close()}
})().catch(e=>{console.error(e);process.exit(1)});
