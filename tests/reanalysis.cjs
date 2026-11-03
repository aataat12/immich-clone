// Public generated fixtures only. Reuse the real-service media acceptance library.
const fs=require('fs'),path=require('path'),assert=require('assert/strict');
const {chromium}=require(process.env.IMMICH_BROWSER_MODULE);
(async()=>{
 const browser=await chromium.launch({channel:'msedge',headless:true});
 const page=await browser.newPage({viewport:{width:1280,height:900}});page.setDefaultTimeout(90000);
 const errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.goto('http://127.0.0.1:2283/');await page.fill('#code',fs.readFileSync(path.join(process.env.IMMICH_TEST_STATE,'access-code.txt'),'utf8').trim());await page.click('#loginForm button');await page.waitForSelector('#library:not(.hidden)');
 const api=async(u,b)=>page.evaluate(async({u,b})=>{const r=await fetch(u,b===undefined?{}:{method:'POST',headers:{'X-Immich-Request':'1'},body:JSON.stringify(b)});if(!r.ok)throw Error('Reanalysis HTTP '+r.status+' '+u);return r.json()},{u,b});
 const receipt=JSON.parse(fs.readFileSync(path.join(process.env.IMMICH_6A_FIXTURES,'app-receipt.json'))),saved=path.join(process.env.IMMICH_6A_FIXTURES,'reanalysis-receipt.json');
 const photo=receipt['face.heic'],video=receipt['video.mp4'];
 const record=(o,id)=>o.assets.find(x=>x.asset.id===id);
 const snapshot=(o,id)=>({analysis:record(o,id).analysis,tags:record(o,id).tags,dismissed:record(o,id).dismissed,faces:o.faces.filter(f=>f.asset_id===id)});
 const command=(id,action)=>api('/api/assets/'+id+'/reanalysis',{action});
 const wait=async(id,status)=>{const end=Date.now()+120000;while(Date.now()<end){const o=await api('/api/organization');const j=record(o,id).reanalysis;if(j?.status===status)return o;if(j?.status==='failed'&&status!=='failed')throw Error('Proposal failed '+j.error);await new Promise(r=>setTimeout(r,250))}throw Error('Proposal timeout '+status)};
 const refresh=async()=>{await page.goto('http://127.0.0.1:2283/people');await page.waitForSelector('.assetCard')};
 const card=id=>page.locator('.assetCard[data-asset="'+id+'"]');
 const click=async(id,label)=>{await card(id).getByRole('button',{name:label,exact:true}).click()};
 const phase=process.argv[2];
 if(phase==='model-failure'){
  const before=snapshot(await api('/api/organization'),photo);
  await command(photo,'start');const o=await wait(photo,'failed');assert.deepEqual(snapshot(o,photo),before);
  assert(!JSON.stringify(o).includes('embedding'));await command(photo,'discard');
 }else if(phase==='prepare'){
  await refresh();const before=snapshot(await api('/api/organization'),photo);
  await click(photo,'Prepare reanalysis proposal');let o=await wait(photo,'ready');assert.deepEqual(snapshot(o,photo),before);assert(!JSON.stringify(o).includes('embedding'));
  await refresh();await click(photo,'Discard reanalysis proposal');assert.deepEqual(snapshot(await api('/api/organization'),photo),before);
  await refresh();await click(photo,'Prepare reanalysis proposal');await wait(photo,'ready');await refresh();page.once('dialog',d=>d.accept());await click(photo,'Apply reanalysis proposal');
  o=await api('/api/organization');assert.equal(record(o,photo).reanalysis.status,'applied');assert.equal(record(o,photo).reanalysis.added,0);assert.deepEqual(snapshot(o,photo).faces,before.faces);assert.deepEqual(snapshot(o,photo).tags,before.tags);
  const lease=await api('/api/imports',{files:[{name:'reanalysis-hold.txt',relative:'reanalysis-hold.txt',size:1,modified:0}]});await api('/api/imports/'+lease.id+'/start',{});
  const baseline=snapshot(o,video);await refresh();await click(video,'Prepare reanalysis proposal');await click(video,'Pause reanalysis');
  assert.equal(record(await api('/api/organization'),video).reanalysis.status,'paused');fs.writeFileSync(saved,JSON.stringify({baseline,lease:lease.id}));
 }else if(phase==='finish'){
  const savedState=JSON.parse(fs.readFileSync(saved));let o=await api('/api/organization');assert.equal(record(o,video).reanalysis.status,'paused');assert.deepEqual(snapshot(o,video),savedState.baseline);
  await refresh();await click(video,'Resume reanalysis');await click(video,'Discard reanalysis proposal');assert.deepEqual(snapshot(await api('/api/organization'),video),savedState.baseline);
  await refresh();await click(video,'Prepare reanalysis proposal');await api('/api/imports/'+savedState.lease+'/cancel',{});o=await wait(video,'ready');assert.equal(record(o,video).reanalysis.next,12);assert.equal(record(o,video).reanalysis.total,12);assert.deepEqual(snapshot(o,video),savedState.baseline);
  await refresh();page.once('dialog',d=>d.accept());await click(video,'Apply reanalysis proposal');o=await api('/api/organization');assert.equal(record(o,video).reanalysis.status,'applied');assert.equal(record(o,video).reanalysis.added,0);assert.deepEqual(snapshot(o,video).faces,savedState.baseline.faces);assert.deepEqual(snapshot(o,video).tags,savedState.baseline.tags);
  await page.setViewportSize({width:390,height:844});await page.reload();await page.waitForSelector('.assetCard');assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
 }else throw Error('Unknown phase');
 assert.deepEqual(errors,[]);await browser.close();console.log('PASS: review-first reanalysis actual LocalService '+phase);
})().catch(e=>{console.error(e.stack||e.message);process.exit(1)});
