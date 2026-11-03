// Public fixtures only. These checks deliberately fail providers/storage and
// require persisted checkpoints and user corrections to survive retries.
const fs=require('fs'),path=require('path'),crypto=require('crypto');
const {chromium}=require(process.env.IMMICH_BROWSER_MODULE);
(async()=>{
 const browser=await chromium.launch({channel:'msedge',headless:true});const page=await browser.newPage();
 await page.goto('http://127.0.0.1:2283/');await page.fill('#code',fs.readFileSync(path.join(process.env.IMMICH_TEST_STATE,'access-code.txt'),'utf8').trim());await page.click('#loginForm button');await page.waitForSelector('#library:not(.hidden)');
 const api=async(u,b)=>page.evaluate(async({u,b})=>{const r=await fetch(u,b===undefined?{}:{method:'POST',headers:{'X-Immich-Request':'1'},body:JSON.stringify(b)});if(!r.ok)throw Error('Recovery request '+u+' HTTP '+r.status);return r.json()},{u,b});
 const wait=async predicate=>{const end=Date.now()+90000;while(Date.now()<end){const o=await api('/api/organization');if(predicate(o))return o;await new Promise(r=>setTimeout(r,100))}throw Error('Recovery state timeout')};
 const file=path.join(process.env.IMMICH_6A_FIXTURES,'recovery-receipt.json'),phase=process.argv[2];let r=phase==='seed'?{}:JSON.parse(fs.readFileSync(file));
 const record=o=>o.assets.find(a=>a.asset.id===r.video);
 const assertBaseline=o=>{const a=record(o);if(JSON.stringify(a.analysis.video)!==JSON.stringify(r.baseline))throw Error('Failed sample advanced or changed checkpoint');if(!o.faces.some(f=>f.id===r.face&&f.name==='Recovery Person')||!a.tags.includes('Recovery Review'))throw Error('Manual correction lost')};
 if(phase==='seed'){
  for(const [key,name,mime] of [['video','rotated-face.mov','video/quicktime'],['heic','recovery.heic','image/heic']]){const bytes=fs.readFileSync(path.join(process.env.IMMICH_6A_FIXTURES,name));r[key]=crypto.createHash('sha256').update(bytes).digest('hex');await page.setInputFiles('#files',{name,mimeType:mime,buffer:bytes});await page.waitForFunction(name=>Array.from(document.querySelectorAll('.card .name')).some(n=>n.textContent===name),name)}
  await api('/api/assets/'+r.video+'/analyze',{});await wait(o=>record(o)?.analysis.video?.next>=1);await api('/api/assets/'+r.video+'/video-analysis',{action:'pause'});
 }else if(phase==='baseline'){
  const o=await api('/api/organization'),a=record(o);if(a.analysis.status!=='paused'||a.analysis.video.next<1||a.analysis.video.next>=5)throw Error('Controlled partial face-bearing checkpoint missing');
  r.baseline=a.analysis.video;r.faces=o.faces.filter(f=>f.asset_id===r.video).map(f=>f.id);if(!r.faces.length)throw Error('Partial video faces missing');r.face=r.faces[0];
  await api('/api/faces/label',{ids:[r.face],name:'Recovery Person'});await api('/api/assets/'+r.video+'/tags',{tags:['Recovery Review']});
 }else if(phase==='decoder-failure'){
  await api('/api/assets/'+r.heic+'/analyze',{});const o=await wait(o=>o.assets.find(a=>a.asset.id===r.heic)?.analysis.status==='failed');assertBaseline(o);
 }else if(phase==='decoder-retry'){
  await api('/api/assets/'+r.heic+'/analyze',{});const o=await wait(o=>o.assets.find(a=>a.asset.id===r.heic)?.analysis.status==='ready');if(!o.faces.some(f=>f.asset_id===r.heic))throw Error('Recovered HEIC face missing');assertBaseline(o);
 }else if(phase==='storage-failure'||phase==='model-failure'){
  await api('/api/assets/'+r.video+'/video-analysis',{action:'resume'});const o=await wait(o=>record(o)?.analysis.status==='failed');assertBaseline(o);
 }else if(phase==='resume-crash'){
  const o=await api('/api/organization');assertBaseline(o);if(record(o).analysis.status!=='failed')throw Error('Failure not persisted after restart');await api('/api/assets/'+r.video+'/video-analysis',{action:'resume'});
 }else if(phase==='recovered'){
  const o=await wait(o=>record(o)?.analysis.status==='ready'),a=record(o),v=a.analysis.video;
  if(v.next!==12||new Set(v.samples.map(s=>s.position_us)).size!==v.samples.length||JSON.stringify(v.samples.slice(0,r.baseline.samples.length))!==JSON.stringify(r.baseline.samples))throw Error('Crash recovery duplicated or replaced completed samples');
  const faces=o.faces.filter(f=>f.asset_id===r.video);for(const id of r.faces)if(!faces.some(f=>f.id===id))throw Error('Existing face identity lost');if(!faces.some(f=>f.id===r.face&&f.name==='Recovery Person')||!a.tags.includes('Recovery Review'))throw Error('Crash recovery lost corrections');
  for(const id of [r.video,r.heic]){const response=await page.request.get('http://127.0.0.1:2283/api/assets/'+id+'/original');if(!response.ok()||crypto.createHash('sha256').update(await response.body()).digest('hex')!==id)throw Error('Recovery original changed')}
 }else throw Error('Unknown recovery phase');
 fs.writeFileSync(file,JSON.stringify(r));await browser.close();console.log('PASS: 6A recovery '+phase);
})().catch(e=>{console.error(e.stack||e.message);process.exit(1)});
