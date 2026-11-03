// Every deletion target is generated inside disposable CI state.
const fs=require('fs'),os=require('os'),path=require('path'),assert=require('assert/strict'),crypto=require('crypto'),{spawn}=require('child_process');
const {chromium}=require(process.env.IMMICH_BROWSER_MODULE);
(async()=>{
 const root=fs.mkdtempSync(path.join(os.tmpdir(),'immich-batch-')),state=process.env.IMMICH_NAV_ATTACH?process.env.IMMICH_TEST_STATE:path.join(root,'state');
 const app=process.env.IMMICH_NAV_ATTACH?null:spawn(process.env.IMMICH_NAV_BINARY,['-mode','serve','-data',state,'-logs',path.join(root,'logs')],{stdio:'ignore'});let browser;
 try{
  for(let i=0;i<100;i++){try{if((await fetch('http://127.0.0.1:2283/health')).ok)break}catch{}await new Promise(r=>setTimeout(r,100))}
  browser=await chromium.launch({channel:'msedge',headless:true});const page=await browser.newPage();page.setDefaultTimeout(60000);const errors=[];page.on('pageerror',e=>errors.push(e.message));
  await page.goto('http://127.0.0.1:2283/');await page.fill('#code',fs.readFileSync(path.join(state,'access-code.txt'),'utf8').trim());await page.click('#loginForm button');await page.waitForSelector('#library:not(.hidden)');
  const post=async(url,body)=>{for(let attempt=0;;attempt++){const r=await page.request.post('http://127.0.0.1:2283'+url,{headers:{'X-Immich-Request':'1'},...(body?{data:body}:{})});if(r.status()===409&&(await r.text()).includes('deletion busy;')&&attempt<4){await page.waitForTimeout(150);continue}assert(r.ok(),url+' '+r.status());return r.json()}};
  const records=[];
  for(let i=0;i<62;i++){
   const data=Buffer.from('disposable multi-page batch original '+i),name=String(i).padStart(4,'0')+'-batch.txt',id=crypto.createHash('sha256').update(data).digest('hex');
   const r=await page.request.post('http://127.0.0.1:2283/api/upload?name='+name,{headers:{'X-Immich-Request':'1'},data});assert(r.ok());const a=(await r.json()).asset;records.push({...a,data});assert.equal(a.id,id);await post('/api/assets/'+id+'/hide');
  }
  const card=i=>page.locator('.card[data-asset="'+records[i].id+'"]');
  const choose=async i=>{await card(i).locator('.hiddenPick').check()};
  const loaded=async action=>{await Promise.all([page.waitForResponse(r=>r.url().includes('/api/gallery?')),action()]);await page.waitForFunction(()=>!busy)};
  await loaded(()=>page.selectOption('#sort','name'));await loaded(()=>page.click('#hiddenMedia'));
  assert.equal(await page.locator('#hiddenMedia').textContent(),'Hide Hidden Media');assert.equal(await page.locator('#libraryHome').count(),0);
  await page.click('#selectHiddenPage');assert.equal(await page.locator('#hiddenSelected').textContent(),'60 selected');await page.click('#clearHiddenSelection');assert.equal(await page.locator('#hiddenSelected').textContent(),'0 selected');
  await choose(0);await choose(1);await loaded(()=>page.click('#nextPage'));await choose(61);assert.equal(await page.locator('#hiddenSelected').textContent(),'3 selected');
  await page.click('#restoreHiddenSelection');await page.locator('#bulkResults').filter({hasText:'3 restored'}).waitFor();await page.waitForFunction(()=>!busy);
  let all=await(await page.request.get('http://127.0.0.1:2283/api/assets')).json();for(const i of [0,1,61])assert(!all.find(a=>a.id===records[i].id).hidden);
  await choose(2);await choose(3);await choose(4);await page.click('#deleteHiddenSelection');assert.equal(await page.locator('#confirmBulkDelete').isDisabled(),true);await page.fill('#bulkDeleteConfirm','wrong');assert(await page.locator('#confirmBulkDelete').isDisabled());await page.click('#cancelBulkDelete');for(const i of [2,3,4])assert(fs.existsSync(path.join(state,'originals',records[i].id)));
  await page.click('#deleteHiddenSelection');await page.fill('#bulkDeleteConfirm','DELETE');
  // Replace one selected disposable item after confirmation was prepared. Its
  // old Added guard must reject the new publication after two earlier successes.
  await post('/api/assets/'+records[4].id+'/delete',{confirm:records[4].id,added:records[4].added});
  const replaced=await page.request.post('http://127.0.0.1:2283/api/upload?name=replaced-batch.txt',{headers:{'X-Immich-Request':'1'},data:records[4].data});assert(replaced.ok());await post('/api/assets/'+records[4].id+'/hide');
  await page.click('#confirmBulkDelete');await page.locator('#bulkResults').filter({hasText:'2 deleted · 0 cleanup pending · 1 failed'}).waitFor();await page.waitForFunction(()=>!busy);for(const i of [2,3])assert(!fs.existsSync(path.join(state,'originals',records[i].id)));assert.deepEqual(fs.readFileSync(path.join(state,'originals',records[4].id)),records[4].data);
  await page.click('#clearHiddenSelection');await choose(5);await choose(6);const block=path.join(state,'previews',records[5].id+'.png');fs.mkdirSync(block);
  await page.click('#deleteHiddenSelection');await page.fill('#bulkDeleteConfirm','DELETE');await page.click('#confirmBulkDelete');await page.locator('#bulkResults').filter({hasText:'1 deleted · 1 cleanup pending · 0 failed'}).waitFor();await page.waitForFunction(()=>!busy);assert(fs.existsSync(path.join(state,'originals',records[5].id)));assert(!fs.existsSync(path.join(state,'originals',records[6].id)));fs.rmdirSync(block);await post('/api/deletions/'+records[5].id+'/retry',{confirm:records[5].id});assert(!fs.existsSync(path.join(state,'originals',records[5].id)));
  await choose(7);await loaded(()=>page.click('#hiddenMedia'));await loaded(()=>page.click('#hiddenMedia'));assert.equal(await page.locator('#hiddenSelected').textContent(),'0 selected');
  await page.setViewportSize({width:390,height:844});assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);assert.deepEqual(errors,[]);
  console.log('PASS: multi-page batch restore/delete, typed confirmation/cancel, stale-publication preservation, truthful partial/pending results, selection reset and narrow layout');
 }finally{if(browser)await browser.close();if(app){app.kill();await new Promise(r=>app.exitCode!==null?r():app.once('exit',r))}fs.rmSync(root,{recursive:true,force:true})}
})().catch(e=>{console.error(e);process.exit(1)});
