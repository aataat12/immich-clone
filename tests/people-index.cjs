// Standalone disposable Windows/Edge review acceptance; no native AI required.
const fs=require('fs'),os=require('os'),path=require('path'),assert=require('assert/strict'),crypto=require('crypto');
const {spawn}=require('child_process');const {chromium}=require(process.env.IMMICH_BROWSER_MODULE);
(async()=>{
 const root=fs.mkdtempSync(path.join(os.tmpdir(),'immich-people-index-')),state=process.env.IMMICH_PEOPLE_ATTACH?process.env.IMMICH_TEST_STATE:path.join(root,'state');let app,browser;
 try{
  let occupied=false;try{await fetch('http://127.0.0.1:2283/health');occupied=true}catch{}assert(!occupied||process.env.IMMICH_PEOPLE_ATTACH,'Disposable runner port occupied');
  if(!process.env.IMMICH_PEOPLE_ATTACH)app=spawn(process.env.IMMICH_PEOPLE_BINARY,['-mode','serve','-data',state,'-logs',path.join(root,'logs')],{stdio:'ignore'});
  for(let i=0;i<100;i++){if(app&&app.exitCode!==null)throw Error('App exited');try{if((await fetch('http://127.0.0.1:2283/health')).ok)break}catch{}await new Promise(r=>setTimeout(r,100))}
  browser=await chromium.launch({channel:'msedge',headless:true});const page=await browser.newPage({viewport:{width:1280,height:900}});page.setDefaultTimeout(30000);const errors=[];page.on('pageerror',e=>errors.push(e.message));
  await page.goto('http://127.0.0.1:2283');await page.fill('#code',fs.readFileSync(path.join(state,'access-code.txt'),'utf8').trim());await page.click('#loginForm button');await page.waitForSelector('#library:not(.hidden)');
  const png=Buffer.from(await page.evaluate(()=>{const c=document.createElement('canvas');c.width=c.height=32;const x=c.getContext('2d');x.fillStyle='orange';x.fillRect(0,0,32,32);return c.toDataURL('image/png').split(',')[1]}),'base64');
  const post=async(u,data)=>{const r=await page.request.post('http://127.0.0.1:2283'+u,{headers:{'X-Immich-Request':'1'},data});assert(r.ok(),'HTTP '+r.status()+' '+u);return r.json()};
  const originals=[];
  for(let i=0;i<55;i++){
   const bytes=Buffer.concat([png,Buffer.from('synthetic-review-'+i)]),result=await post('/api/upload?name=review-'+i+'.png',bytes);originals.push({id:result.asset.id,bytes});
   if(i<51)for(let j=0;j<2;j++)await post('/api/assets/'+result.asset.id+'/manual-face',{box:[.1+j*.2,.1,.2,.2],name:''});
  }
  await page.goto('http://127.0.0.1:2283/people');await page.waitForFunction(()=>document.querySelectorAll('.faceCard').length===100&&document.querySelectorAll('.assetCard').length===50);
  const ids=kind=>page.locator('.'+kind+'Card').evaluateAll((els,k)=>els.map(e=>e.dataset[k]),kind);
  const navigate=async id=>{await page.click('#'+id);await page.waitForFunction(()=>!paging&&!busy)};
  const firstFaces=await ids('face'),firstAssets=await ids('asset');
  await navigate('moreFaces');assert.equal((await ids('face')).length,2);assert.deepEqual(await ids('asset'),firstAssets);
  const lastFaces=await ids('face');await navigate('morePhotos');assert.equal((await ids('asset')).length,5);assert.deepEqual(await ids('face'),lastFaces);
  const lastAssets=await ids('asset');await navigate('prevFaces');assert.deepEqual(await ids('face'),firstFaces);assert.deepEqual(await ids('asset'),lastAssets);await navigate('prevPhotos');assert.deepEqual(await ids('asset'),firstAssets);
  const face=page.locator('.faceCard').first(),faceID=await face.getAttribute('data-face');await face.getByLabel('Person name',{exact:true}).fill('Index person');await face.getByRole('button',{name:'Save name',exact:true}).click();await page.waitForFunction(()=>!busy&&Array.from(document.querySelectorAll('.faceCard .name')).some(e=>e.textContent==='Index person'));
  const search=async q=>{await page.fill('#query',q);await page.locator('#searchForm button').first().click();await page.waitForFunction(()=>!busy&&!paging)};
  await search('INDEX PERSON');await page.waitForFunction(()=>document.querySelectorAll('.faceCard').length===1&&document.querySelectorAll('.assetCard').length===1);assert.equal(await page.locator('.faceCard').getAttribute('data-face'),faceID);
  await page.locator('.assetCard').getByLabel('Custom tags',{exact:true}).fill('ReviewTag');await page.locator('.assetCard').getByRole('button',{name:'Save tags',exact:true}).click();await page.waitForFunction(()=>!busy);
  await search('reviewtag');await page.waitForFunction(()=>document.querySelectorAll('.faceCard').length===0&&document.querySelectorAll('.assetCard').length===1);
  await page.click('#clear');await page.waitForFunction(()=>document.querySelectorAll('.faceCard').length===100);
  const background=await page.evaluate(()=>getComputedStyle(document.documentElement).backgroundColor);
  await page.check('#unnamed');await page.waitForFunction(()=>Array.from(document.querySelectorAll('.faceCard .name')).every(e=>e.textContent==='Unnamed'));
  assert.equal(await page.evaluate(()=>getComputedStyle(document.documentElement).backgroundColor),background);await navigate('moreFaces');assert.equal((await ids('face')).length,1);await navigate('prevFaces');
  await page.locator('.faceCard').first().getByRole('button',{name:'Not a face',exact:true}).click();await page.waitForFunction(()=>!busy&&document.querySelector('#moreFaces').classList.contains('hidden'));
  await page.check('#dismissedFaces');await page.waitForFunction(()=>document.querySelectorAll('.faceCard').length===1);await page.locator('.faceCard').getByRole('button',{name:'Restore face',exact:true}).click();await page.waitForFunction(()=>!busy&&document.querySelectorAll('.faceCard').length===0);
  await page.uncheck('#dismissedFaces');await page.waitForFunction(()=>document.querySelectorAll('.faceCard').length===100);
  await page.reload();await page.waitForFunction(()=>document.querySelectorAll('.faceCard').length===100);await search('Index person');await page.waitForFunction(()=>document.querySelectorAll('.faceCard').length===1);
  for(const width of [1280,390]){await page.setViewportSize({width,height:900});assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false)}
  for(const item of originals){const r=await page.request.get('http://127.0.0.1:2283/api/assets/'+item.id+'/original');assert(r.ok());assert.equal(crypto.createHash('sha256').update(await r.body()).digest('hex'),item.id)}
  assert.deepEqual(errors,[]);console.log('PASS: indexed People review independent previous/next pages, search/name/tag correction invalidation, unnamed and dismissed transitions, restore/reload, stable background, desktop/narrow layout and all55 original hashes');
 }finally{if(browser)await browser.close();if(app&&app.exitCode===null){app.kill();await new Promise(r=>app.once('exit',r))}fs.rmSync(root,{recursive:true,force:true})}
})().catch(e=>{console.error(e.message);process.exit(1)});
