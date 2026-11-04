// Disposable installed-service acceptance only; fixture identities are public/synthetic.
const fs=require('fs'),path=require('path'),crypto=require('crypto'),assert=require('assert/strict');
const {chromium}=require(process.env.IMMICH_BROWSER_MODULE);
(async()=>{
 assert.equal(process.env.IMMICH_COMPAT_INSTALLED,'1','Explicit disposable installed fixture opt-in required');
 const root=process.env.IMMICH_COMPAT_FIXTURES,state=process.env.IMMICH_TEST_STATE;
 assert(root&&state,'Compatibility fixtures and installed state required');
 const fixtures=JSON.parse(fs.readFileSync(path.join(root,'expected.json'),'utf8'));assert.equal(fixtures.length,24);
 const browser=await chromium.launch({channel:'msedge',headless:true});
 try{
  const page=await browser.newPage();page.setDefaultTimeout(45000);const errors=[];page.on('pageerror',e=>errors.push(e.message));
  await page.goto('http://127.0.0.1:2283/');await page.fill('#code',fs.readFileSync(path.join(state,'access-code.txt'),'utf8').trim());await page.click('#loginForm button');await page.waitForSelector('#library:not(.hidden)');
  const ids=[];
  for(const f of fixtures){
   const data=fs.readFileSync(path.join(root,f.name));assert.equal(crypto.createHash('sha256').update(data).digest('hex'),f.sha256);
   const r=await page.request.post('http://127.0.0.1:2283/api/upload?name='+encodeURIComponent(f.name),{headers:{'X-Immich-Request':'1'},data});assert(r.ok(),'Fixture import refused');const a=(await r.json()).asset;assert.equal(a.id,f.sha256);ids.push(a.id);
  }
  await page.waitForFunction(async ids=>{const r=await fetch('/api/assets');if(!r.ok)return false;const assets=await r.json();return ids.every(id=>assets.some(a=>a.id===id&&a.preview!=='queued'))},ids,{timeout:90000});
  const assets=await(await page.request.get('http://127.0.0.1:2283/api/assets')).json();
  for(const f of fixtures){
   const a=assets.find(a=>a.id===f.sha256);assert(a&&a.preview==='ready','Installed preview not ready');assert.equal(a.width,f.width);assert.equal(a.height,f.height);
   if(f.photo){assert.equal(a.photo.orientation,f.photo.orientation);for(const key of ['make','model','captured'])assert.equal(a.photo[key]||'',f.photo[key]);assert(!a.photo.warning)}
   const pixels=await page.evaluate(async id=>{
    const img=new Image();img.src='/api/assets/'+id+'/preview';await new Promise((resolve,reject)=>{img.onload=resolve;img.onerror=()=>reject(Error('Preview unavailable'))});
    const canvas=document.createElement('canvas');canvas.width=img.naturalWidth;canvas.height=img.naturalHeight;const ctx=canvas.getContext('2d');ctx.drawImage(img,0,0);
    return [[1,1],[3,1],[1,3],[3,3]].map(([x,y])=>Array.from(ctx.getImageData(Math.floor(canvas.width*x/4),Math.floor(canvas.height*y/4),1,1).data));
   },f.sha256);
   for(let i=0;i<4;i++)for(let c=0;c<4;c++)assert(Math.abs(pixels[i][c]-f.pixels[i][c])<=f.tolerance,'Installed independent first-page/frame/orientation pixel mismatch');
   const r=await page.request.get('http://127.0.0.1:2283/api/assets/'+f.sha256+'/original');assert(r.ok());assert.equal(crypto.createHash('sha256').update(await r.body()).digest('hex'),f.sha256);
  }
  assert.deepEqual(errors,[]);
  console.log('PASS: actual installed LocalService imports/previews/original downloads for24 independent image fixtures; three public reduced camera EXIF samples, eight orientations/date priority, progressive/CMYK JPEG, PNG variants, BMP, GIF first frame, four compressed/multipage TIFF first pages,12MP JPEG; exact geometry/metadata/pixels/original SHA256');
 }finally{await browser.close()}
})().catch(e=>{console.error(e.message);process.exit(1)});
