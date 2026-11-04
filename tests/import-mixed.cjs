// Synthetic media and actual browser uploads only; no intercepted requests.
const fs=require('fs'),path=require('path'),crypto=require('crypto'),assert=require('assert/strict');
const {chromium}=require(process.env.IMMICH_BROWSER_MODULE);
const hash=b=>crypto.createHash('sha256').update(b).digest('hex');
const watchdog=setTimeout(()=>{console.error('Mixed import phase exceeded eight-minute bound');process.exit(1)},480000);watchdog.unref();
(async()=>{
 const root=process.env.IMMICH_MIXED_ROOT,state=path.join(root,'state'),source=path.join(root,'mixed source'),receipt=path.join(root,'mixed.json'),phase=process.argv[2];
 const browser=await chromium.launch({channel:'msedge',headless:true});
 try{
  const context=await browser.newContext(),page=await context.newPage();page.setDefaultTimeout(60000);const errors=[];page.on('pageerror',e=>errors.push(e.message));
  await page.goto('http://127.0.0.1:2283/');await page.fill('#code',fs.readFileSync(path.join(state,'access-code.txt'),'utf8').trim());await page.click('#loginForm button');await page.waitForSelector('#library:not(.hidden)');
  const api=async url=>{const r=await page.request.get('http://127.0.0.1:2283'+url);assert(r.ok(),url+' '+r.status());return r.json()};
  const pause=async()=>page.waitForFunction(()=>!busy&&document.querySelector('#counts').textContent.startsWith('paused'));
  if(phase==='start'){
   fs.mkdirSync(source);
   const pngs=await page.evaluate(()=>Array.from({length:48},(_,i)=>{const c=document.createElement('canvas');c.width=64;c.height=48;const g=c.getContext('2d');g.fillStyle=`rgb(${i*5},${255-i*3},${i*2})`;g.fillRect(0,0,64,48);return c.toDataURL('image/png').split(',')[1]}));
   const bases=['synthetic.heic','hevc.mp4','rotated.mov'].map(n=>Buffer.from(fs.readFileSync(path.join('experiments/media-worker/fixtures',n+'.b64'),'utf8'),'base64'));
   const entries=[],buffers=[];let bytes=0;
   for(let i=0;i<240;i++){
    const kind=(i<192?i:i-192)%4,ext=['png','heic','mp4','mov'][kind];let data;
    if(i>=192)data=buffers[i-192];
    else if(kind===0)data=Buffer.from(pngs[i/4],'base64');
    else {const box=Buffer.alloc(i===82?(1<<20):16);box.writeUInt32BE(box.length);box.write('free',4);box.writeUInt32BE(i,8);data=Buffer.concat([bases[kind-1],box])}
    const name=String(i).padStart(4,'0')+'.'+ext;fs.writeFileSync(path.join(source,name),data);buffers.push(data);entries.push({name,id:hash(data),size:data.length,kind});bytes+=data.length;
   }
   assert.equal(new Set(entries.map(e=>e.id)).size,192);
   await page.goto('http://127.0.0.1:2283/import');await page.setInputFiles('#folder',source);
   const cdp=await context.newCDPSession(page);await cdp.send('Network.enable');await cdp.send('Network.emulateNetworkConditions',{offline:false,latency:0,downloadThroughput:10<<20,uploadThroughput:128<<10});
   await page.click('#start');
   const until=Date.now()+90000;let observed=false;
   while(Date.now()<until){const parts=fs.readdirSync(path.join(state,'staging')).filter(n=>n.endsWith('.part'));if(parts.some(n=>{try{return fs.statSync(path.join(state,'staging',n)).size>=65536}catch{return false}})){observed=true;break}await new Promise(r=>setTimeout(r,50))}
   assert(observed,'Actual partial staging write was not observed');
   const id=await page.evaluate(()=>localStorage.getItem('immich-import')),saved=await api('/api/imports/'+id);assert.equal(saved.session.accepted_files,82);assert.equal(saved.session.new_files,82);
   fs.writeFileSync(receipt,JSON.stringify({id,entries,bytes}));fs.writeFileSync(path.join(root,'crash-ready'),'ready');
   await page.waitForFunction(()=>!busy);assert.match(await page.locator('#message').textContent(),/unavailable|interrupted|incomplete|failed/i);
   console.log('PASS: actual mixed-file transfer observed partially staged before forced service termination');
  }else if(phase==='finish'){
   const baseline=JSON.parse(fs.readFileSync(receipt)),{id,entries,bytes}=baseline;
   let saved=await api('/api/imports/'+id);assert.equal(saved.session.status,'paused');assert.equal(saved.session.accepted_files,82);assert(!saved.entries[82].asset_id);assert.equal(saved.background_paused,false);assert.deepEqual(fs.readdirSync(path.join(state,'staging')),[]);
   for(let i=0;i<82;i++)assert.equal(saved.entries[i].asset_id,entries[i].id);
   await page.goto('http://127.0.0.1:2283/import');await page.selectOption('#history',id);await page.click('#open');await pause();await page.setInputFiles('#folder',source);await page.click('#resume');
   await page.waitForFunction(()=>document.querySelector('#counts').textContent.startsWith('completed')&&document.querySelector('#counts').textContent.includes('240 / 240'));
   for(let after=0;after>=0;){saved=await api('/api/imports/'+id+'?after='+after);assert.equal(saved.session.accepted_files,240);assert.equal(saved.session.accepted_bytes,bytes);assert.equal(saved.session.new_files,192);assert.equal(saved.session.duplicates,48);assert.equal(saved.background_paused,false);for(const e of saved.entries){assert.equal(e.asset_id,entries[e.index].id);assert(!e.error);assert.equal(e.duplicate,e.index>=192)}after=saved.next}
   const until=Date.now()+300000;let assets=[];
   while(Date.now()<until){assets=await api('/api/assets');assert.equal(assets.length,192);const failed=assets.filter(a=>a.preview==='failed');assert.deepEqual(failed.map(a=>a.name),[],'Mixed media preview failed');const ready=assets.filter(a=>a.preview==='ready').length;if(ready===192)break;await new Promise(r=>setTimeout(r,1000))}
   assert(assets.every(a=>a.preview==='ready'&&a.width>0&&a.height>0),'Preview processing did not complete');
   const counts={};for(const a of assets){counts[a.mime]=(counts[a.mime]||0)+1;const r=await page.request.get('http://127.0.0.1:2283/api/assets/'+a.id+'/original');assert(r.ok());assert.equal(hash(await r.body()),a.id);const aliases=entries.filter(e=>e.id===a.id).map(e=>'mixed source/'+e.name).sort();assert.deepEqual(a.sources.sort(),aliases)}
   assert.deepEqual(counts,{'image/png':48,'image/heic':48,'video/mp4':48,'video/quicktime':48});
   for(const e of entries)assert.equal(hash(fs.readFileSync(path.join(source,e.name))),e.id);
   const archive=await page.request.get('http://127.0.0.1:2283/api/backup',{timeout:90000});assert(archive.ok());fs.writeFileSync(path.join(root,'mixed.zip'),await archive.body());
   await page.setViewportSize({width:390,height:844});assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
   console.log('PASS: 240 real mixed imports, 192 distinct originals, 48 duplicates, crash/replay/resume, all previews, source aliases, original hashes and archive');
  }else throw Error('Unknown mixed import phase');assert.deepEqual(errors,[]);
 }finally{await browser.close()}
})().catch(e=>{console.error(e);process.exit(1)});
