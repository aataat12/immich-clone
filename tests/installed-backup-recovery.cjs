// Actual accepted installer; disposable NTFS state; streamed bytes, no large body buffers.
const fs=require('fs'),path=require('path'),crypto=require('crypto'),http=require('http'),assert=require('assert/strict');
const {Readable,Transform}=require('stream'),{pipeline}=require('stream/promises');
const {chromium}=require(process.env.IMMICH_BROWSER_MODULE);
const root=process.env.IMMICH_BACKUP_RECOVERY_ROOT,state=process.env.IMMICH_BACKUP_RECOVERY_STATE,phase=process.argv[2],base='http://127.0.0.1:2283';
const watchdog=setTimeout(()=>{console.error('Bounded installed backup phase exceeded 12 minutes');process.exit(1)},12*60*1000);
const baselineFile=path.join(root,'baseline.json'),limit=4*1024**3;
const hash=b=>crypto.createHash('sha256').update(b).digest('hex');
const delay=ms=>new Promise(r=>setTimeout(r,ms));
async function* entropy(size,seed,digest){const key=Buffer.alloc(32);key[0]=seed;const cipher=crypto.createCipheriv('aes-256-ctr',key,Buffer.alloc(16)),zero=Buffer.alloc(1<<20);let n=0;while(n<size){const b=cipher.update(zero.subarray(0,Math.min(zero.length,size-n)));if(digest)digest.update(b);n+=b.length;yield b}}
const hashFile=async file=>{const h=crypto.createHash('sha256');let bytes=0;for await(const b of fs.createReadStream(file)){h.update(b);bytes+=b.length}return{hash:h.digest('hex'),bytes}};
(async()=>{
 const browser=await chromium.launch({channel:'msedge',headless:true});
 try{
  const context=await browser.newContext(),page=await context.newPage();page.setDefaultTimeout(45000);
  await page.goto(base);await page.fill('#code',fs.readFileSync(path.join(state,'access-code.txt'),'utf8').trim());await page.click('#loginForm button');await page.waitForSelector('#library:not(.hidden)');
  const get=async url=>{const r=await page.request.get(base+url);assert(r.ok(),'Read refused '+url);return r.json()};
  const post=async(url,data)=>{const r=await page.request.post(base+url,{headers:{'X-Immich-Request':'1'},data:JSON.stringify(data||{})});assert(r.ok(),'Write refused '+url+' '+r.status());return r.json()};
  const cookie=(await context.cookies(base)).map(c=>c.name+'='+c.value).join(';');
  const poll=async(fn,ms=90000)=>{const end=Date.now()+ms;while(Date.now()<end){const v=await fn();if(v)return v;await delay(50)}throw Error('Bounded state transition timed out')};
  const job=()=>get('/api/backup-job');
  const start=async()=>{await page.goto(base+'/backups');await page.waitForFunction(()=>!document.querySelector('#start').disabled);await page.click('#start');return poll(async()=>{const j=await job();return j.id&&['queued','snapshot','copying','verifying','hashing','ready'].includes(j.status)?j:false},10000)};
  const stable=async saved=>{const assets=await get('/api/assets');assert.equal(assets.length,saved.files.length);for(const f of saved.files){const a=assets.find(a=>a.id===f.id);assert(a);assert.equal(a.size,f.size);assert.equal(a.name,f.name);assert(a.sources.includes(f.relative));if(f.jpeg){assert.equal(a.photo.orientation,6);assert.equal(a.photo.captured,'2024-03-02T10:11:12');assert.equal(a.width,10);assert.equal(a.height,20)}}const receipt=await get('/api/imports/'+saved.importID);assert.equal(receipt.session.status,'completed');assert.equal(receipt.session.accepted_files,saved.files.length);assert.equal(receipt.session.new_files,saved.files.length);assert.equal(receipt.session.duplicates,0);assert.equal(receipt.session.accepted_bytes,saved.bytes);assert.deepEqual(receipt.entries.map(e=>e.asset_id),saved.files.map(f=>f.id));return assets};
  const fullDownload=async(j,file)=>{let bytes=0;const digest=crypto.createHash('sha256');await new Promise((resolve,reject)=>{const req=http.get(base+'/api/backup-job/download?id='+j.id,{headers:{Cookie:cookie}},res=>{if(res.statusCode!==200){res.resume();reject(Error('Archive download refused'));return}assert.equal(res.headers['x-backup-sha256'],j.sha256);pipeline(res,new Transform({transform(b,encoding,cb){bytes+=b.length;digest.update(b);cb(null,b)}}),fs.createWriteStream(file,{flags:'wx'})).then(resolve,reject)});req.setTimeout(120000,()=>req.destroy(Error('Archive stream idle timeout')));req.on('error',reject)});assert.equal(bytes,j.archive_bytes);assert.equal(digest.digest('hex'),j.sha256);return bytes};
  if(phase==='seed-crash'){
   const jpg=fs.readFileSync(path.join(root,'fixture.jpg')),files=[{name:'large-4gib.bin',relative:'BackupFixture/large-4gib.bin',size:limit,seed:1},{name:'extra.bin',relative:'BackupFixture/extra.bin',size:1<<20,seed:2}];
   for(let i=0;i<16;i++){const b=Buffer.concat([jpg,Buffer.from('backup-'+i)]);files.push({name:'camera-'+String(i).padStart(2,'0')+'.jpg',relative:'BackupFixture/camera-'+i+'.jpg',size:b.length,jpeg:true,data:b})}
   const m=await post('/api/imports',{files:files.map(({name,relative,size})=>({name,relative,size,modified:1700000000000}))});await post('/api/imports/'+m.id+'/start');
   const started=Date.now();
   for(let i=0;i<files.length;i++){
    const f=files[i];let e;
    if(f.jpeg){const r=await page.request.post(base+'/api/imports/'+m.id+'/files/'+i,{headers:{'X-Immich-Request':'1'},data:f.data});assert(r.ok());e=await r.json();f.id=hash(f.data);delete f.data}
    else{
     const digest=crypto.createHash('sha256');
     e=await new Promise((resolve,reject)=>{const req=http.request(base+'/api/imports/'+m.id+'/files/'+i,{method:'POST',headers:{Cookie:cookie,'X-Immich-Request':'1','Content-Length':f.size}},res=>{let data='';res.on('data',b=>{data+=b;assert(data.length<65536)});res.on('end',()=>{if(res.statusCode!==200){reject(Error('Large import failed '+res.statusCode));return}try{resolve(JSON.parse(data))}catch(e){reject(e)}});res.on('error',reject)});req.setTimeout(120000,()=>req.destroy(Error('Import idle timeout')));req.on('error',reject);pipeline(Readable.from(entropy(f.size,f.seed,digest),{objectMode:false,highWaterMark:1<<20}),req).catch(reject)});f.id=digest.digest('hex');
    }
    assert.equal(e.asset_id,f.id);
   }
   await post('/api/imports/'+m.id+'/finish');
   const saved={files,bytes:files.reduce((n,f)=>n+f.size,0),importID:m.id,secretHash:hash(fs.readFileSync(path.join(state,'access-code.txt')))};
   await poll(async()=>{const assets=await get('/api/assets');return files.every(f=>assets.some(a=>a.id===f.id&&a.preview!=='queued'))},90000);
   for(const f of files.filter(f=>f.jpeg)){const a=(await get('/api/assets')).find(a=>a.id===f.id);assert.equal(a.preview,'ready')}
   await post('/api/assets/'+files[2].id+'/tags',{tags:['Backup preservation tag']});
   await post('/api/assets/'+files[2].id+'/manual-face',{box:[0,0,1,1],name:'Backup preserved person'});
   const marked=(await get('/api/organization')).faces.find(f=>f.asset_id===files[2].id&&f.name==='Backup preserved person');assert(marked);saved.faceID=marked.id;
   await post('/api/assets/'+files[3].id+'/hide');
   fs.writeFileSync(baselineFile,JSON.stringify(saved));await stable(saved);
   console.log('PASS: actual installed streamed import physically stored '+saved.bytes+' bytes across '+files.length+' originals; EXIF, tags, named manual face, hidden state and completed receipts prepared in '+(Date.now()-started)+'ms');
   const j=await start();await poll(async()=>{const j=await job();assert(!['failed','cancelled','ready'].includes(j.status));return j.status==='copying'&&j.progress.bytes>=(64<<20)?j:false},30000);
   fs.writeFileSync(path.join(root,'crash-ready'),j.id);await poll(async()=>{try{await job();return false}catch{return true}},30000);
   console.log('PASS: real nonzero backup copy observed before forced installed-service termination');
  }else if(phase==='recover-first'||phase==='recover-second'){
   const saved=JSON.parse(fs.readFileSync(baselineFile));await stable(saved);let previous=await job();if(phase==='recover-first')assert.equal(previous.status,'interrupted');else assert(['failed','interrupted'].includes(previous.status),'Outage must remain failed/interrupted without Ready');assert.equal((await page.request.get(base+'/api/backup-job/download?id='+previous.id)).status(),409);assert(fs.readdirSync(path.join(state,'backups')).every(n=>!n.endsWith('.part')));
   const started=Date.now(),j0=await start(),ready=await poll(async()=>{const j=await job();assert(!['failed','cancelled','unavailable'].includes(j.status),'Backup failed');return j.status==='ready'?j:false},180000);assert.equal(ready.id,j0.id);assert(ready.archive_bytes>limit);assert.equal(ready.progress.bytes,ready.progress.total_bytes);
   assert((await page.request.post(base+'/api/backup-job/start',{headers:{'X-Immich-Request':'1'}})).status()===409);
   if(phase==='recover-first'){
    const res=await new Promise((resolve,reject)=>{const req=http.get(base+'/api/backup-job/download?id='+ready.id,{headers:{Cookie:cookie}},r=>{if(r.statusCode!==200){reject(Error('Partial download refused'));return}r.once('data',()=>{r.pause();resolve(r)})});req.on('error',reject)});
    const blocked=await page.request.post(base+'/api/backup-job/discard',{headers:{'X-Immich-Request':'1'},data:JSON.stringify({id:ready.id,confirm:true})});assert.equal(blocked.status(),409);res.destroy();await delay(300);
   }
   const file=path.join(root,phase==='recover-first'?'download-first.zip':'download-second.zip');await fullDownload(ready,file);
   const range=await page.request.get(base+'/api/backup-job/download?id='+ready.id,{headers:{Range:'bytes=1048576-1114111'}});assert.equal(range.status(),206);const fd=fs.openSync(file,'r'),segment=Buffer.alloc(65536);fs.readSync(fd,segment,0,segment.length,1048576);fs.closeSync(fd);assert.deepEqual(await range.body(),segment);
   const guideWait=page.waitForEvent('download');await page.click('#recoveryDownload');const guide=await guideWait;const guideFile=path.join(root,phase+'-recovery.txt');await guide.saveAs(guideFile);assert(fs.readFileSync(guideFile,'utf8').includes(ready.sha256));
   saved[phase]={id:ready.id,sha256:ready.sha256,size:ready.archive_bytes,elapsed_ms:Date.now()-started};fs.writeFileSync(baselineFile,JSON.stringify(saved));
   await page.reload();await page.waitForFunction(()=>document.querySelector('#status').textContent==='ready');
   console.log('PASS: installed >4GiB archive create/full verification/streamed download, interrupted download protection, exact range bytes, recovery guide and refresh retention; cycle '+phase+' '+(Date.now()-started)+'ms');
  }else if(phase==='start-outage'){
   const saved=JSON.parse(fs.readFileSync(baselineFile));await stable(saved);const retained=await job();assert.equal(retained.status,'ready');assert.equal(retained.id,saved['recover-first'].id);assert.equal(retained.sha256,saved['recover-first'].sha256);
   await post('/api/backup-job/discard',{id:retained.id,confirm:true});assert.deepEqual(await hashFile(path.join(root,'download-first.zip')),{hash:retained.sha256,bytes:retained.archive_bytes});
   const j=await start();await poll(async()=>{const j=await job();assert(!['failed','cancelled','ready'].includes(j.status));return j.status==='copying'&&j.progress.bytes>=(64<<20)?j:false},30000);
   fs.writeFileSync(path.join(root,'outage-ready'),j.id);
   await poll(async()=>{try{const j=await job();assert.notEqual(j.status,'ready','Partial outage job must not publish Ready');if(j.status==='failed'||j.status==='interrupted'){const r=await page.request.get(base+'/api/backup-job/download?id='+j.id);assert.equal(r.status(),409);return true}return false}catch(e){if(e.code==='ERR_ASSERTION')throw e;return true}},45000);
   console.log('PASS: actual volume loss during nonzero backup copy exposed failure/unavailability without Ready publication');
  }else if(phase==='finish'){
   const saved=JSON.parse(fs.readFileSync(baselineFile));await stable(saved);const ready=await job();assert.equal(ready.status,'ready');assert.equal(ready.id,saved['recover-second'].id);
   assert.deepEqual(await hashFile(path.join(root,'download-first.zip')),{hash:saved['recover-first'].sha256,bytes:saved['recover-first'].size});
   assert.deepEqual(await hashFile(path.join(root,'download-second.zip')),{hash:ready.sha256,bytes:ready.archive_bytes});
   await post('/api/backup-job/discard',{id:ready.id,confirm:true});assert.equal((await job()).status,'discarded');assert(fs.readdirSync(path.join(state,'backups')).every(n=>!n.endsWith('.part')&&!n.endsWith('.zip')));
   console.log('PASS: two full archives retained unchanged through service crash/volume recovery; explicit temporary cleanup preserves originals and both independent downloads');
  }else throw Error('Unknown backup recovery phase');
 }finally{clearTimeout(watchdog);await browser.close()}
})().catch(e=>{console.error(e.message);process.exit(1)});
