// Only generated public/synthetic fixtures; no user media or machine inventory.
const fs=require('fs'),path=require('path'),crypto=require('crypto');
const {chromium}=require(process.env.IMMICH_BROWSER_MODULE);
(async()=>{
 const browser=await chromium.launch({channel:'msedge',headless:true});
 const page=await browser.newPage({viewport:{width:1280,height:900}});page.setDefaultTimeout(90000);
 const errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.goto('http://127.0.0.1:2283/');await page.fill('#code',fs.readFileSync(path.join(process.env.IMMICH_TEST_STATE,'access-code.txt'),'utf8').trim());await page.click('#loginForm button');await page.waitForSelector('#library:not(.hidden)');
 const api=async(u,b)=>page.evaluate(async({u,b})=>{const r=await fetch(u,b===undefined?{}:{method:'POST',headers:{'X-Immich-Request':'1'},body:JSON.stringify(b)});if(!r.ok)throw Error('6A request '+u+' HTTP '+r.status);return r.json()},{u,b});
 const phase=process.argv[2];const receiptPath=path.join(process.env.IMMICH_6A_FIXTURES,'app-receipt.json');let receipt;
 if(phase==='start'){
  receipt={};for(const name of ['face.heic','face-grid.heic','video.mp4']){
   const bytes=fs.readFileSync(path.join(process.env.IMMICH_6A_FIXTURES,name));receipt[name]=crypto.createHash('sha256').update(bytes).digest('hex');
   await page.setInputFiles('#files',{name,mimeType:name.endsWith('heic')?'image/heic':'video/mp4',buffer:bytes});await page.waitForFunction(name=>Array.from(document.querySelectorAll('.card .name')).some(n=>n.textContent===name),name);
  }
  fs.writeFileSync(receiptPath,JSON.stringify(receipt));
 }else receipt=JSON.parse(fs.readFileSync(receiptPath));
 await page.goto('http://127.0.0.1:2283/people');await page.waitForSelector('.assetCard');
 const card=name=>page.locator('.assetCard').filter({hasText:name});
 if(phase==='start'){
  for(const name of ['face.heic','face-grid.heic']){
   await card(name).getByRole('button',{name:'Analyze photo locally',exact:true}).click();
   await page.waitForFunction(async id=>{const o=await(await fetch('/api/organization')).json();return o.assets.find(x=>x.asset.id===id)?.analysis.status==='ready'},receipt[name]);
   const o=await api('/api/organization');if(!o.faces.some(f=>f.asset_id===receipt[name]))throw Error('HEIC face missing');await page.click('#refresh');
  }
  // Import priority holds the worker so pause is deterministic, including fast machines.
  const lease=await api('/api/imports',{files:[{name:'hold.txt',relative:'hold.txt',size:1,modified:0}]});await api('/api/imports/'+lease.id+'/start',{});receipt.lease=lease.id;
  await card('video.mp4').getByRole('button',{name:'Analyze sampled video',exact:true}).click();await card('video.mp4').getByRole('button',{name:'Pause video analysis',exact:true}).click();
  const o=await api('/api/organization');if(o.assets.find(x=>x.asset.id===receipt['video.mp4']).analysis.status!=='paused')throw Error('Pause missing');
  fs.writeFileSync(receiptPath,JSON.stringify(receipt));
 }else if(phase==='finish'){
  let o=await api('/api/organization');if(o.assets.find(x=>x.asset.id===receipt['video.mp4']).analysis.status!=='paused')throw Error('Pause lost on service restart');
  await card('video.mp4').getByRole('button',{name:'Resume video analysis',exact:true}).click();await card('video.mp4').getByRole('button',{name:'Cancel remaining video analysis',exact:true}).click();
  o=await api('/api/organization');if(o.assets.find(x=>x.asset.id===receipt['video.mp4']).analysis.status!=='cancelled')throw Error('Cancel missing');
  await api('/api/imports/'+receipt.lease+'/cancel',{});
  await card('video.mp4').getByRole('button',{name:'Resume video analysis',exact:true}).click();
  await page.waitForFunction(async id=>{const o=await(await fetch('/api/organization')).json();return o.assets.find(x=>x.asset.id===id)?.analysis.status==='ready'},receipt['video.mp4']);
  o=await api('/api/organization');const a=o.assets.find(x=>x.asset.id===receipt['video.mp4']).analysis;const v=a.video;
  if(v.next!==12||v.targets.length!==12||new Set(v.samples.map(s=>s.position_us)).size!==v.samples.length||!a.objects.includes('cup'))throw Error('Sampled coverage/object results missing');
  const faces=o.faces.filter(f=>f.asset_id===receipt['video.mp4']);if(!faces.length||!faces.every(f=>Number.isSafeInteger(f.sample_us)))throw Error('Frame provenance missing');
  receipt.face=faces[0].id;await api('/api/faces/label',{ids:[receipt.face],name:'Video Test Person'});await api('/api/assets/'+receipt['video.mp4']+'/tags',{tags:['Video Review']});
  await page.click('#refresh');const face=page.locator('.faceCard').filter({hasText:'Video Test Person'}).first();await face.getByRole('button',{name:'View sampled frame',exact:true}).click();
  await page.waitForFunction(()=>document.querySelector('#contextImage').naturalWidth>0);if(!(await page.locator('#contextCaption').textContent()).includes('s'))throw Error('Timestamp caption missing');await page.keyboard.press('Escape');
  for(const [name,id] of Object.entries(receipt).filter(([n])=>n.endsWith('.heic')||n.endsWith('.mp4'))){const r=await page.request.get('/api/assets/'+id+'/original');if(!r.ok()||crypto.createHash('sha256').update(await r.body()).digest('hex')!==id)throw Error('Original changed '+name)}
  await page.setViewportSize({width:390,height:844});await page.reload();await page.waitForSelector('.assetCard');if(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth))throw Error('6A narrow overflow');
  fs.writeFileSync(receiptPath,JSON.stringify(receipt));
 }else throw Error('Unknown test phase');
 if(errors.length)throw Error(errors.join(';'));await browser.close();console.log('PASS: 6A actual app/browser '+phase);
})().catch(e=>{console.error(e.stack||e.message);process.exit(1)});
