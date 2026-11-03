// Real directory-rename outage on disposable state; no request or reader mocks.
const fs=require('fs'),path=require('path'),crypto=require('crypto'),assert=require('assert/strict');
const {chromium}=require(process.env.IMMICH_BROWSER_MODULE),hash=b=>crypto.createHash('sha256').update(b).digest('hex');
(async()=>{
 const root=process.env.IMMICH_DESTINATION_ROOT,state=path.join(root,'state'),source=path.join(root,'source'),receipt=path.join(root,'receipt.json'),phase=process.argv[2];
 const browser=await chromium.launch({channel:'msedge',headless:true});
 try{
  const page=await browser.newPage();page.setDefaultTimeout(45000);const errors=[];page.on('pageerror',e=>errors.push(e.message));
  await page.goto('http://127.0.0.1:2283/');await page.fill('#code',fs.readFileSync(path.join(state,'access-code.txt'),'utf8').trim());await page.click('#loginForm button');await page.waitForSelector('#library:not(.hidden)');
  const get=async u=>{const r=await page.request.get('http://127.0.0.1:2283'+u);assert(r.ok());return r.json()};
  const post=async(u,data)=>{const r=await page.request.post('http://127.0.0.1:2283'+u,{headers:{'X-Immich-Request':'1'},...(data?{data}:{})});assert(r.ok(),u+' '+r.status());return r.json()};
  if(phase==='prepare'){
   fs.mkdirSync(source);const payloads=[Buffer.from('saved original before destination outage'),Buffer.alloc(8<<20,79),Buffer.from('saved original before destination outage')],names=['a-preserved.txt','b-pending.bin','c-duplicate.txt'],files=[];
   for(let i=0;i<3;i++){const file=path.join(source,names[i]);fs.writeFileSync(file,payloads[i]);fs.utimesSync(file,new Date(1700000000000),new Date(1700000000000));files.push({name:names[i],relative:'',size:payloads[i].length,modified:1700000000000})}
   const m=await post('/api/imports',{files});await post('/api/imports/'+m.id+'/start');const accepted=await post('/api/imports/'+m.id+'/files/0',payloads[0]);assert.equal(accepted.asset_id,hash(payloads[0]));
   fs.writeFileSync(receipt,JSON.stringify({id:m.id,files,hashes:payloads.map(hash)}));
  }else{
   const {id,files,hashes}=JSON.parse(fs.readFileSync(receipt)),url='/api/imports/'+id;
   if(phase==='outage')await post(url+'/pause');
   await page.goto('http://127.0.0.1:2283/import');await page.selectOption('#history',id);await page.click('#open');await page.waitForFunction(()=>document.querySelector('#counts').textContent.startsWith('paused'));await page.setInputFiles('#files',files.map(f=>path.join(source,f.name)));
   if(phase==='outage'){
    assert(!fs.existsSync(path.join(state,'originals')));assert.equal(hash(fs.readFileSync(path.join(state,'originals.offline',hashes[0]))),hashes[0]);
    await page.click('#resume');await page.waitForFunction(()=>!busy&&document.querySelector('#counts').textContent.startsWith('paused'));assert.match(await page.locator('#message').textContent(),/Copy failed/i);
    let p=await get(url);assert.equal(p.session.accepted_files,1);assert.equal(p.session.accepted_bytes,files[0].size);assert.equal(p.entries[0].asset_id,hashes[0]);assert(p.entries[0].error);assert(!p.entries[1].asset_id);assert.equal(p.background_paused,false);
    // Also attempt the genuinely pending file against the missing destination.
    await post(url+'/start');const failed=await page.request.post('http://127.0.0.1:2283'+url+'/files/1',{headers:{'X-Immich-Request':'1'},data:fs.readFileSync(path.join(source,files[1].name))});assert.equal(failed.status(),500);await post(url+'/pause');p=await get(url);assert(p.entries[1].error);assert(!p.entries[1].asset_id);assert.equal(p.session.accepted_files,1);assert.equal(p.session.new_files,1);assert.equal(p.session.duplicates,0);assert.equal(p.session.accepted_bytes,files[0].size);assert.equal(p.background_paused,false);
    const catalog=await get('/api/assets');assert.equal(catalog.length,1);assert.equal(catalog[0].id,hashes[0]);assert(!fs.existsSync(path.join(state,'originals')));assert.deepEqual(fs.readdirSync(path.join(state,'staging')),[]);assert.equal(hash(fs.readFileSync(path.join(state,'originals.offline',hashes[0]))),hashes[0]);
   }else if(phase==='resume'){
    let p=await get(url);assert.equal(p.session.accepted_files,1);assert.equal(p.session.status,'paused');assert.equal(p.background_paused,false);
    await page.click('#resume');await page.waitForFunction(()=>!busy&&document.querySelector('#counts').textContent.startsWith('completed')&&document.querySelector('#counts').textContent.includes('3 / 3'));
    p=await get(url);assert.equal(p.session.accepted_files,3);assert.equal(p.session.new_files,2);assert.equal(p.session.duplicates,1);assert.equal(p.session.accepted_bytes,files.reduce((n,f)=>n+f.size,0));assert.equal(p.background_paused,false);
    for(let i=0;i<3;i++){assert.equal(p.entries[i].asset_id,hashes[i]);assert(!p.entries[i].error);assert.equal(hash(fs.readFileSync(path.join(source,files[i].name))),hashes[i]);const r=await page.request.get('http://127.0.0.1:2283/api/assets/'+hashes[i]+'/original');assert(r.ok());assert.equal(hash(await r.body()),hashes[i])}
    assert.equal((await get('/api/assets')).length,2);assert.deepEqual(fs.readdirSync(path.join(state,'staging')),[]);
   }else throw Error('Unknown phase');
  }
  assert.deepEqual(errors,[]);console.log('PASS: actual LocalService destination-directory recovery '+phase);
 }finally{await browser.close()}
})().catch(e=>{console.error(e);process.exit(1)});
