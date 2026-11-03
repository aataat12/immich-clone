// Disposable app state only. Supports local Chromium and Windows Edge.
const fs=require('fs'),os=require('os'),path=require('path'),assert=require('assert/strict'),crypto=require('crypto');
const {spawn}=require('child_process');
const {chromium}=require(process.env.IMMICH_BROWSER_MODULE);
(async()=>{
 const root=fs.mkdtempSync(path.join(os.tmpdir(),'immich-navigation-')),state=path.join(root,'state');
 const app=spawn(process.env.IMMICH_NAV_BINARY,['-mode','serve','-data',state,'-logs',path.join(root,'logs')],{stdio:'ignore'});
 let browser;
 try{
  for(let i=0;i<100;i++){try{if((await fetch('http://127.0.0.1:2283/health')).ok)break}catch{}await new Promise(r=>setTimeout(r,100))}
  browser=await chromium.launch(process.env.IMMICH_CHROME_BIN?{executablePath:process.env.IMMICH_CHROME_BIN,headless:true,args:['--no-sandbox']}:{channel:'msedge',headless:true});
  const page=await browser.newPage({viewport:{width:1280,height:900}});const errors=[];page.on('pageerror',e=>errors.push(e.message));
  await page.goto('http://127.0.0.1:2283/');await page.fill('#code',fs.readFileSync(path.join(state,'access-code.txt'),'utf8').trim());await page.click('#loginForm button');await page.waitForSelector('#library:not(.hidden)');
  const change=async(action,visibility)=>{await Promise.all([page.waitForResponse(r=>r.url().includes('/api/gallery?')&&r.url().includes('visibility='+visibility)),action()]);await page.waitForFunction(v=>document.querySelector('#visibility').value===v&&document.querySelector('#libraryTitle').textContent===(v==='hidden'?'Hidden Media':'Photos & videos'),visibility)};
  const check=async hidden=>{assert.equal(await page.locator('#libraryHome').isVisible(),hidden);assert.equal(await page.getAttribute('#hiddenMedia','aria-pressed'),String(hidden));assert.equal(await page.locator('#hiddenHelp').isVisible(),hidden)};
  await check(false);
  await change(()=>page.click('#hiddenMedia'),'hidden');await check(true);await page.getByText('No hidden items.',{exact:true}).waitFor();
  await change(()=>page.click('#libraryHome'),'library');await check(false);assert(!(await page.locator('#status').textContent()).includes('No hidden items.'));
  const data=Buffer.from('disposable hidden navigation original'),id=crypto.createHash('sha256').update(data).digest('hex');
  const uploaded=await page.request.post('http://127.0.0.1:2283/api/upload?name=navigation.txt',{headers:{'X-Immich-Request':'1'},data});assert(uploaded.ok());
  assert((await page.request.post('http://127.0.0.1:2283/api/assets/'+id+'/hide',{headers:{'X-Immich-Request':'1'}})).ok());
  await change(()=>page.click('#hiddenMedia'),'hidden');await check(true);await page.locator('.card[data-asset="'+id+'"]').waitFor();
  await page.setViewportSize({width:390,height:844});assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
  await change(()=>page.click('#hiddenMedia'),'library');await check(false);
  await change(()=>page.selectOption('#visibility','hidden'),'hidden');await check(true);
  await change(()=>page.selectOption('#visibility','library'),'library');await check(false);
  const assets=await(await page.request.get('http://127.0.0.1:2283/api/assets')).json();assert(assets.some(a=>a.id===id&&a.hidden),'Navigation must not unhide or delete the item');
  const original=await page.request.get('http://127.0.0.1:2283/api/assets/'+id+'/original');assert(original.ok());assert.deepEqual(await original.body(),data);assert.deepEqual(errors,[]);
  console.log('PASS: empty/populated Hidden Media Home, toggle-off, selector sync, narrow layout and original preservation');
 }finally{if(browser)await browser.close();app.kill();await new Promise(r=>app.exitCode!==null?r():app.once('exit',r));fs.rmSync(root,{recursive:true,force:true})}
})().catch(e=>{console.error(e);process.exit(1)});
