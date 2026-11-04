// Bounded sustained mixed-format ingestion through the installed browser UI.
// Distinct originals use synthetic fixtures or byte-varied copies of three public
// reduced camera samples. This is workload scale, not a diverse natural corpus.
const fs=require('fs'),path=require('path'),crypto=require('crypto'),assert=require('assert/strict');
const {chromium}=require(process.env.IMMICH_BROWSER_MODULE),hash=b=>crypto.createHash('sha256').update(b).digest('hex');
const sleep=ms=>new Promise(r=>setTimeout(r,ms));
const watchdog=setTimeout(()=>{console.error('Endurance phase exceeded twelve minutes');process.exit(1)},720000);watchdog.unref();
(async()=>{
 const root=process.env.IMMICH_IMPORT_ENDURANCE_ROOT,state=path.join(root,'state'),source=path.join(root,'mixed endurance'),record=path.join(root,'endurance.json'),phase=process.argv[2];
 const browser=await chromium.launch({channel:'msedge',headless:true});
 try{
  const page=await browser.newPage();page.setDefaultTimeout(60000);const errors=[];page.on('pageerror',e=>errors.push(e.message));
  await page.goto('http://127.0.0.1:2283/');await page.fill('#code',fs.readFileSync(path.join(state,'access-code.txt'),'utf8').trim());await page.click('#loginForm button');await page.waitForSelector('#library:not(.hidden)');
  const get=async u=>{const r=await page.request.get('http://127.0.0.1:2283'+u);assert(r.ok(),u+' '+r.status());return r.json()};
  if(phase==='first'){
   fs.mkdirSync(source);const pngs=await page.evaluate(()=>Array.from({length:256},(_,i)=>{const c=document.createElement('canvas');c.width=320;c.height=240;const g=c.getContext('2d');g.fillStyle=`rgb(${i},${255-i},${i^127})`;g.fillRect(0,0,320,240);g.fillStyle='white';g.fillRect(i%100,i%80,32,24);return c.toDataURL('image/png').split(',')[1]}));
   const fixtures=process.env.IMMICH_COMPAT_FIXTURES,camera=['Canon_40D.jpg','Nikon_D70.jpg','Pentax_K10D.jpg','12mp.jpg'].map(n=>fs.readFileSync(path.join(fixtures,n))),tiff=fs.readFileSync(path.join(fixtures,'first-raw.tiff'));
   const native=['synthetic.heic','hevc.mp4','rotated.mov'].map(n=>Buffer.from(fs.readFileSync(path.join('experiments/media-worker/fixtures',n+'.b64'),'utf8'),'base64'));
   const entries=[];let bytes=0;
   for(let i=0;i<2048;i++){
    const ordinal=Math.floor(i/6),kind=i%6,ext=['png','jpg','tiff','heic','mp4','mov'][kind];let data;
    if(i>=1536)data=fs.readFileSync(path.join(source,entries[i-1536].name));
    else if(kind===0)data=Buffer.from(pngs[ordinal],'base64');
    else if(kind===1){const base=camera[ordinal%4],comment=Buffer.alloc(18);comment.writeUInt16BE(0xfffe);comment.writeUInt16BE(16,2);comment.writeUInt32BE(i,4);data=Buffer.concat([base.subarray(0,2),comment,base.subarray(2)])}
    else {const suffix=Buffer.alloc(16);suffix.writeUInt32BE(16);suffix.write('free',4);suffix.writeUInt32BE(i,8);data=Buffer.concat([kind===2?tiff:native[kind-3],suffix])}
    // Duplicates retain the original extension, including where 1536 cycles back.
    const name=String(i).padStart(5,'0')+'.'+ext;fs.writeFileSync(path.join(source,name),data);entries.push({name,id:hash(data),size:data.length});bytes+=data.length;
   }
   assert.equal(new Set(entries.map(e=>e.id)).size,1536);fs.writeFileSync(record,JSON.stringify({entries,bytes,sessions:[],timings:[]}));
  }
  const saved=JSON.parse(fs.readFileSync(record)),{entries,bytes}=saved;
  if(phase==='first'||phase==='second'){
   const started=Date.now();await page.goto('http://127.0.0.1:2283/import');await page.setInputFiles('#folder',source);assert(!(await page.locator('#start').isDisabled()));await page.click('#start');await page.waitForFunction(()=>!busy&&document.querySelector('#counts').textContent.startsWith('completed'),null,{timeout:600000});
   const id=await page.evaluate(()=>localStorage.getItem('immich-import'));assert(!saved.sessions.includes(id));saved.sessions.push(id);saved.timings.push({phase,copy_ms:Date.now()-started});
   let after=0,checked=0;
   do{const p=await get('/api/imports/'+id+'?after='+after);assert.equal(p.session.accepted_files,2048);assert.equal(p.session.accepted_bytes,bytes);assert.equal(p.session.new_files,phase==='first'?1536:0);assert.equal(p.session.duplicates,phase==='first'?512:2048);assert.equal(p.background_paused,false);assert(p.entries.length<=250);for(const e of p.entries){assert.equal(e.asset_id,entries[e.index].id);assert(!e.error);assert.equal(e.duplicate,phase==='second'||e.index>=1536);checked++}after=p.next}while(after>=0);assert.equal(checked,2048);
   fs.writeFileSync(record,JSON.stringify(saved));console.log('PASS: '+phase+' installed browser batch, 2048 receipts, exact bytes/counts and bounded history pages; '+saved.timings.at(-1).copy_ms+' ms');
  }else if(phase==='verify'){
   const expected=new Map(entries.map(e=>[e.id,e])),started=Date.now();let assets;
   const scan=async()=>{const all=[];let cursor='';do{const p=await get('/api/gallery?sort=name&limit=60'+(cursor?'&after='+encodeURIComponent(cursor):''));assert.equal(p.total,1536);assert(p.assets.length<=60);all.push(...p.assets);cursor=p.next}while(cursor);assert.equal(all.length,1536);assert.equal(new Set(all.map(a=>a.id)).size,1536);return all};
   const until=Date.now()+600000;
   while(Date.now()<until){try{assets=await scan()}catch(e){if(String(e).includes('409')){await sleep(250);continue}throw e}const failed=assets.filter(a=>a.preview==='failed');assert.deepEqual(failed.map(a=>a.name),[],'Actual media preview failed');if(assets.every(a=>a.preview==='ready'))break;await sleep(1000)}
   assert(assets&&assets.every(a=>a.preview==='ready'&&a.width>0&&a.height>0),'All mixed previews must drain after service restart');
   const types={};for(const a of assets){assert(expected.has(a.id));types[a.mime]=(types[a.mime]||0)+1;const r=await page.request.get('http://127.0.0.1:2283/api/assets/'+a.id+'/original');assert(r.ok());assert.equal(hash(await r.body()),a.id);const aliases=entries.filter(e=>e.id===a.id).map(e=>'mixed endurance/'+e.name).sort();assert.deepEqual(a.sources.sort(),aliases)}
   assert.deepEqual(types,{'image/png':256,'image/jpeg':256,'image/tiff':256,'image/heic':256,'video/mp4':256,'video/quicktime':256});
   for(const e of entries)assert.equal(hash(fs.readFileSync(path.join(source,e.name))),e.id);
   for(const id of saved.sessions){const p=await get('/api/imports/'+id);assert.equal(p.session.status,'completed');assert.equal(p.session.accepted_files,2048);assert.equal(p.session.accepted_bytes,bytes)}
   assert.deepEqual(fs.readdirSync(path.join(state,'staging')),[]);await page.goto('http://127.0.0.1:2283/');await page.waitForSelector('#library:not(.hidden)');await page.setViewportSize({width:390,height:844});assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
   console.log('PASS: 4096 real copy receipts, 1536 actual distinct mixed originals/previews, 2560 duplicates, restart persistence, all source/original SHA256s, aliases, bounded gallery completeness; '+JSON.stringify({bytes_per_batch:bytes,timings:saved.timings,verify_ms:Date.now()-started,types}));
  }else throw Error('Unknown phase');assert.deepEqual(errors,[]);
 }finally{await browser.close()}
})().catch(e=>{console.error(e);process.exit(1)});
