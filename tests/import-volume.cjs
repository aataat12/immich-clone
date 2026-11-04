// Disposable installed-service volume detach. No intercepted requests or synthetic I/O errors.
const fs=require('fs'),path=require('path'),crypto=require('crypto'),assert=require('assert/strict');
const {chromium}=require(process.env.IMMICH_BROWSER_MODULE);
const hash=b=>crypto.createHash('sha256').update(b).digest('hex');
const sleep=ms=>new Promise(r=>setTimeout(r,ms));
const watchdog=setTimeout(()=>{console.error('Volume phase exceeded three minutes');process.exit(1)},180000);watchdog.unref();
(async()=>{
 const root=process.env.IMMICH_IMPORT_VOLUME_ROOT,state=process.env.IMMICH_IMPORT_VOLUME_STATE,source=path.join(root,'volume source'),receipt=path.join(root,'volume.json'),phase=process.argv[2];
 if(phase==='start'){
  fs.mkdirSync(source);const small=Buffer.from('durable receipt before actual mounted volume detach'),base=Buffer.from(fs.readFileSync('experiments/media-worker/fixtures/hevc.mp4.b64','utf8'),'base64'),box=Buffer.alloc(16<<20);box.writeUInt32BE(box.length);box.write('free',4);box.writeUInt32BE(743,8);
  const payloads=[small,Buffer.concat([base,box]),small],names=['a-preserved.txt','b-pending.mp4','c-duplicate.txt'];
  for(let i=0;i<names.length;i++)fs.writeFileSync(path.join(source,names[i]),payloads[i]);
  fs.writeFileSync(receipt,JSON.stringify({entries:names.map((name,i)=>({name,id:hash(payloads[i]),size:payloads[i].length}))}));
 }
 const baseline=JSON.parse(fs.readFileSync(receipt)),browser=await chromium.launch({channel:'msedge',headless:true});
 try{
  const context=await browser.newContext(),page=await context.newPage();page.setDefaultTimeout(60000);const errors=[];page.on('pageerror',e=>errors.push(e.message));
  await page.goto('http://127.0.0.1:2283/');await page.fill('#code',fs.readFileSync(path.join(state,'access-code.txt'),'utf8').trim());await page.click('#loginForm button');await page.waitForSelector('#library:not(.hidden)');await page.goto('http://127.0.0.1:2283/import');
  const get=async u=>{const r=await page.request.get('http://127.0.0.1:2283'+u);assert(r.ok(),u+' '+r.status());return r.json()};
  if(phase==='start'){
   await page.setInputFiles('#files',baseline.entries.map(e=>path.join(source,e.name)));const cdp=await context.newCDPSession(page);await cdp.send('Network.enable');await cdp.send('Network.emulateNetworkConditions',{offline:false,latency:0,downloadThroughput:10<<20,uploadThroughput:1<<20});await page.click('#start');
   const until=Date.now()+60000;let partial=false;
   while(Date.now()<until){if(fs.readdirSync(path.join(state,'staging')).some(n=>{try{return n.endsWith('.part')&&fs.statSync(path.join(state,'staging',n)).size>=131072}catch{return false}})){partial=true;break}await sleep(50)}
   assert(partial,'No actual partial staging before detach');baseline.id=await page.evaluate(()=>localStorage.getItem('immich-import'));const saved=await get('/api/imports/'+baseline.id);assert.equal(saved.session.accepted_files,1);assert.equal(saved.entries[0].asset_id,baseline.entries[0].id);assert(!saved.entries[1].asset_id);
   fs.writeFileSync(receipt,JSON.stringify(baseline));fs.writeFileSync(path.join(root,'detach-ready'),'ready');
   await page.waitForFunction(()=>!busy);assert.match(await page.locator('#message').textContent(),/unavailable|interrupted|incomplete|failed/i);console.log('PASS: actual volume detach interrupted upload without claiming completion');
  }else if(phase==='resume'){
   let saved=await get('/api/imports/'+baseline.id);assert.equal(saved.session.accepted_files,1);assert.equal(saved.session.accepted_bytes,baseline.entries[0].size);assert.equal(saved.entries[0].asset_id,baseline.entries[0].id);assert(!saved.entries[1].asset_id);assert.equal(saved.session.status,'paused');assert.equal(saved.background_paused,false);assert.deepEqual(fs.readdirSync(path.join(state,'staging')),[]);
   await page.selectOption('#history',baseline.id);await page.click('#open');await page.waitForFunction(()=>document.querySelector('#counts').textContent.startsWith('paused'));await page.setInputFiles('#files',baseline.entries.map(e=>path.join(source,e.name)));await page.click('#resume');await page.waitForFunction(()=>!busy&&document.querySelector('#counts').textContent.startsWith('completed'));
   saved=await get('/api/imports/'+baseline.id);assert.equal(saved.session.accepted_files,3);assert.equal(saved.session.new_files,2);assert.equal(saved.session.duplicates,1);assert.equal(saved.session.accepted_bytes,baseline.entries.reduce((n,e)=>n+e.size,0));assert.equal(saved.background_paused,false);
   for(const e of saved.entries){assert.equal(e.asset_id,baseline.entries[e.index].id);assert(!e.error)}
   for(const e of baseline.entries){assert.equal(hash(fs.readFileSync(path.join(source,e.name))),e.id);const r=await page.request.get('http://127.0.0.1:2283/api/assets/'+e.id+'/original');assert(r.ok());assert.equal(hash(await r.body()),e.id)}assert.deepEqual(fs.readdirSync(path.join(state,'staging')),[]);
   console.log('PASS: mounted-volume reconnect, durable earlier receipt, partial cleanup, resume, duplicate accounting and all source/original SHA256s');
  }else throw Error('Unknown phase');assert.deepEqual(errors,[]);
 }finally{await browser.close()}
})().catch(e=>{console.error(e);process.exit(1)});
