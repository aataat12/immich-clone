// Disposable installed-service acceptance only; fixture identities are public/synthetic.
const fs=require('fs'),path=require('path'),crypto=require('crypto'),assert=require('assert/strict');
const {chromium}=require(process.env.IMMICH_BROWSER_MODULE);
(async()=>{
 assert.equal(process.env.IMMICH_COMPAT_INSTALLED,'1','Explicit disposable installed fixture opt-in required');
 const root=process.env.IMMICH_COMPAT_FIXTURES,state=process.env.IMMICH_TEST_STATE,restored=process.argv.includes('--restored');
 assert(root&&state,'Compatibility fixtures and installed state required');
 const fixtures=JSON.parse(fs.readFileSync(path.join(root,'expected.json'),'utf8'));assert.equal(fixtures.length,24);
 const browser=await chromium.launch({channel:'msedge',headless:true});
 try{
  const page=await browser.newPage();page.setDefaultTimeout(45000);const errors=[];page.on('pageerror',e=>errors.push(e.message));
  await page.goto('http://127.0.0.1:2283/');await page.fill('#code',fs.readFileSync(path.join(state,'access-code.txt'),'utf8').trim());await page.click('#loginForm button');await page.waitForSelector('#library:not(.hidden)');
  const ids=fixtures.map(f=>f.sha256),writes=[];if(restored)page.on('request',r=>{if(!['GET','HEAD'].includes(r.method()))writes.push(r.method())});
  if(restored){const existing=await(await page.request.get('http://127.0.0.1:2283/api/assets')).json();assert(ids.every(id=>existing.some(a=>a.id===id)),'Restored matrix IDs must already exist; repair uploads are forbidden')}
  for(const f of fixtures){
   const data=fs.readFileSync(path.join(root,f.name));assert.equal(crypto.createHash('sha256').update(data).digest('hex'),f.sha256);
   if(!restored){const r=await page.request.post('http://127.0.0.1:2283/api/upload?name='+encodeURIComponent(f.name),{headers:{'X-Immich-Request':'1'},data});assert(r.ok(),'Fixture import refused');const a=(await r.json()).asset;assert.equal(a.id,f.sha256)}
  }
  if(process.env.IMMICH_COMPAT_ASYNC_CONTROL==='1'){
   const before=await(await page.request.get('http://127.0.0.1:2283/api/assets')).json();const queued=ids.filter(id=>before.some(a=>a.id===id&&a.preview==='queued')).length;assert(queued>0,'Old-wait control requires actual queued previews');
   const started=Date.now(),old=await page.waitForFunction(async ids=>{const r=await fetch('/api/assets');if(!r.ok)return false;const assets=await r.json();return ids.every(id=>assets.some(a=>a.id===id&&a.preview!=='queued'))},ids,{timeout:5000});
   assert.equal(await old.jsonValue(),false,'Pinned async-predicate control must demonstrate premature false completion');await old.dispose();
   const after=await(await page.request.get('http://127.0.0.1:2283/api/assets')).json();assert(ids.some(id=>after.some(a=>a.id===id&&a.preview==='queued')),'Old async wait must return while previews remain queued');
   console.log('CONTROL: old async wait returned false before readiness; '+queued+' actual previews queued before the call; '+(Date.now()-started)+'ms');
  }
  const deadline=Date.now()+90000,started=Date.now();let assets=null;
  while(Date.now()<deadline){
   const response=await page.request.get('http://127.0.0.1:2283/api/assets',{timeout:Math.max(1,Math.min(5000,deadline-Date.now()))});
   if(response.status()!==429){
    assert(response.ok(),'Catalog polling failed');const snapshot=await response.json();
    for(const f of fixtures){const a=snapshot.find(a=>a.id===f.sha256);if(a&&a.preview==='failed')throw Error('Installed preview failed for public fixture '+f.name+': '+JSON.stringify({preview:a.preview,mime:a.mime,error:a.error}))}
    if(ids.every(id=>snapshot.some(a=>a.id===id&&a.preview==='ready'))){assets=snapshot;break}
   }
   await new Promise(resolve=>setTimeout(resolve,250));
  }
  assert(assets,'Timed out waiting for all24 public-fixture previews to become ready');
  if(process.env.IMMICH_COMPAT_ASYNC_CONTROL==='1')console.log('PASS: explicit catalog polling waited '+(Date.now()-started)+'ms for all24 actual ready previews');
  for(const f of fixtures){
   const a=assets.find(a=>a.id===f.sha256);assert(a&&a.preview==='ready','Installed preview not ready for public fixture '+f.name+': '+JSON.stringify(a?{preview:a.preview,mime:a.mime,error:a.error}:null));assert.equal(a.width,f.width);assert.equal(a.height,f.height);
   if(f.photo){assert.equal(a.photo.orientation,f.photo.orientation);for(const key of ['make','model','captured'])assert.equal(a.photo[key]||'',f.photo[key]);assert(!a.photo.warning)}
   const pixels=await page.evaluate(async id=>{
    const img=new Image();img.src='/api/assets/'+id+'/preview';await new Promise((resolve,reject)=>{img.onload=resolve;img.onerror=()=>reject(Error('Preview unavailable'))});
    const canvas=document.createElement('canvas');canvas.width=img.naturalWidth;canvas.height=img.naturalHeight;const ctx=canvas.getContext('2d');ctx.drawImage(img,0,0);
    return [[1,1],[3,1],[1,3],[3,3]].map(([x,y])=>Array.from(ctx.getImageData(Math.floor(canvas.width*x/4),Math.floor(canvas.height*y/4),1,1).data));
   },f.sha256);
   for(let i=0;i<4;i++)for(let c=0;c<4;c++)assert(Math.abs(pixels[i][c]-f.pixels[i][c])<=f.tolerance,'Installed independent first-page/frame/orientation pixel mismatch');
   const r=await page.request.get('http://127.0.0.1:2283/api/assets/'+f.sha256+'/original');assert(r.ok());assert.equal(crypto.createHash('sha256').update(await r.body()).digest('hex'),f.sha256);
  }
  assert.deepEqual(errors,[]);if(restored)assert.deepEqual(writes,[],'Restored checks must not issue mutations or repair uploads');
  console.log('PASS: '+(restored?'independently restored read-only matrix without repair uploads':(process.env.IMMICH_COMPAT_BINARY?'standalone production serve import matrix':'actual installed LocalService import matrix'))+'; previews/original downloads for24 independent image fixtures; three public reduced camera EXIF samples, eight orientations/date priority, progressive/CMYK JPEG, PNG variants, BMP, GIF first frame, four compressed/multipage TIFF first pages,12MP JPEG; exact geometry/metadata/pixels/original SHA256');
 }finally{await browser.close()}
})().catch(e=>{console.error(e.message);process.exit(1)});
