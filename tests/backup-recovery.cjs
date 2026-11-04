// Disposable standalone Windows binary + Edge; never attaches to a user library.
const fs=require('fs'),os=require('os'),path=require('path'),assert=require('assert/strict'),crypto=require('crypto');
const {spawn,spawnSync}=require('child_process');
const {chromium}=require(process.env.IMMICH_BROWSER_MODULE);
const hash=b=>crypto.createHash('sha256').update(b).digest('hex');
(async()=>{
 const root=fs.mkdtempSync(path.join(os.tmpdir(),'immich-recovery-ui-')),state=path.join(root,'state');
 let app,browser;
 const stop=async()=>{if(app&&app.exitCode===null){app.kill();await new Promise((resolve,reject)=>{const timer=setTimeout(()=>reject(Error('App did not terminate')),10000);app.once('exit',()=>{clearTimeout(timer);resolve()})})}app=null};
 try{
  let occupied=false;try{await fetch('http://127.0.0.1:2283/health');occupied=true}catch{}assert(!occupied,'Disposable runner port must be unused');
  app=spawn(process.env.IMMICH_RECOVERY_BINARY,['-mode','serve','-data',state,'-logs',path.join(root,'logs')],{stdio:'ignore'});
  for(let i=0;i<100;i++){if(app.exitCode!==null)throw Error('Disposable app exited');try{if((await fetch('http://127.0.0.1:2283/health')).ok)break}catch{}await new Promise(r=>setTimeout(r,100))}
  browser=await chromium.launch({channel:'msedge',headless:true});
  const page=await browser.newPage({viewport:{width:1280,height:900},acceptDownloads:true});page.setDefaultTimeout(30000);
  const errors=[];page.on('pageerror',e=>errors.push(e.message));
  await page.goto('http://127.0.0.1:2283/');const code=fs.readFileSync(path.join(state,'access-code.txt'),'utf8').trim();
  await page.fill('#code',code);await page.click('#loginForm button');await page.waitForSelector('#library:not(.hidden)');
  const data=Buffer.from('synthetic recovery instructions original'),id=hash(data);
  assert((await page.request.post('http://127.0.0.1:2283/api/upload?name=recovery-fixture.bin',{headers:{'X-Immich-Request':'1'},data})).ok());
  await page.goto('http://127.0.0.1:2283/backups');await page.waitForFunction(()=>document.querySelector('#status').textContent==='idle');
  assert(!await page.locator('#recoveryDownload').isVisible());assert(await page.locator('#recoveryDownload').isDisabled());
  const summary=page.locator('#recoveryHelp summary');await summary.focus();await page.keyboard.press('Enter');assert(await page.locator('#recoveryHelp').evaluate(e=>e.open));
  assert((await page.locator('#recoveryHelp').innerText()).includes('does not replace or switch'));
  await page.keyboard.press('Enter');assert(!await page.locator('#recoveryHelp').evaluate(e=>e.open));
  await page.waitForFunction(()=>!document.querySelector('#start').disabled);await page.click('#start');await page.waitForFunction(()=>document.querySelector('#status').textContent==='ready');
  const job=await(await page.request.get('http://127.0.0.1:2283/api/backup-job')).json();
  assert(await page.locator('#recoveryDownload').isVisible());assert(!await page.locator('#recoveryDownload').isDisabled());
  const archive=path.join(root,'download.zip'),instructions=path.join(root,'recovery.txt');
  const zipWait=page.waitForEvent('download');await page.click('#download');const zip=await zipWait;await zip.saveAs(archive);assert.equal(await zip.failure(),null);assert.equal(hash(fs.readFileSync(archive)),job.sha256);
  const mutations=[];const watch=req=>{if(req.method()!=='GET')mutations.push(req.url())};page.on('request',watch);
  const guideWait=page.waitForEvent('download');await page.click('#recoveryDownload');const guide=await guideWait;await guide.saveAs(instructions);assert.equal(await guide.failure(),null);
  page.off('request',watch);assert.deepEqual(mutations,[],'Instructions download must not mutate server state');
  assert.equal(guide.suggestedFilename(),'immich-backup-recovery-'+job.sha256.slice(0,12)+'.txt');
  const text=fs.readFileSync(instructions,'utf8');
  for(const required of [job.sha256,job.archive_bytes+' bytes',"C:\\Program Files\\ImmichClone\\immichclone.exe",'-mode restore','-mode verify','$LASTEXITCODE','does NOT activate','Ctrl+C'])assert(text.includes(required),'Missing recovery instruction: '+required);
  assert(!text.includes(code));assert(!text.includes(state));
  const after=await(await page.request.get('http://127.0.0.1:2283/api/backup-job')).json();assert.equal(after.id,job.id);assert.equal(after.status,'ready');
  for(const width of [1280,390]){await page.setViewportSize({width,height:900});await page.locator('#recoveryHelp').evaluate(e=>e.open=true);assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,'Recovery help overflow');}
  await page.reload();await page.waitForFunction(()=>document.querySelector('#status').textContent==='ready');assert(await page.locator('#recoveryDownload').isVisible());
  page.once('dialog',d=>d.accept());await page.click('#discard');await page.waitForFunction(()=>document.querySelector('#status').textContent==='discarded');assert(!await page.locator('#recoveryDownload').isVisible());assert(await page.locator('#recoveryDownload').isDisabled());
  assert.equal(hash(fs.readFileSync(archive)),job.sha256);assert.deepEqual(errors,[]);
  await browser.close();browser=null;await stop();fs.renameSync(state,state+'.offline');
  const target=path.join(root,'independent-restored');
  for(const args of [['-mode','restore','-archive',archive,'-data',target],['-mode','verify','-data',target]]){
   const result=spawnSync(process.env.IMMICH_RECOVERY_BINARY,[...args,'-logs',path.join(root,'restore-logs')],{encoding:'utf8',timeout:60000});assert.equal(result.status,0,'Offline recovery command failed');
  }
  assert.deepEqual(fs.readFileSync(path.join(target,'originals',id)),data);assert.equal(hash(fs.readFileSync(archive)),job.sha256);
  console.log('PASS: keyboard restore help, ready-only per-archive instructions, real ZIP/text downloads, exact SHA256/size and Windows command paths, no server mutation or access-code disclosure, narrow layout, reload/discard behavior, independent actual CLI restore/verify with source offline');
 }finally{if(browser)await browser.close();await stop();fs.rmSync(root,{recursive:true,force:true})}
})().catch(e=>{console.error(e.message);process.exit(1)});
