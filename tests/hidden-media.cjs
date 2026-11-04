// Reuse generated public fixtures, never private user media or machine details.
const fs=require('fs'),path=require('path'),crypto=require('crypto'),assert=require('assert/strict');
const {chromium}=require(process.env.IMMICH_BROWSER_MODULE);
(async()=>{
 const browser=await chromium.launch({channel:'msedge',headless:true});const context=await browser.newContext({viewport:{width:1280,height:900},permissions:['clipboard-read','clipboard-write']});const page=await context.newPage();page.setDefaultTimeout(60000);const errors=[];page.on('pageerror',e=>errors.push(e.message));
 const receipt=JSON.parse(fs.readFileSync(path.join(process.env.IMMICH_6A_FIXTURES,'app-receipt.json'))),id=receipt['video.mp4'],saved=path.join(process.env.IMMICH_6A_FIXTURES,'hidden-receipt.json');
 await page.goto('http://127.0.0.1:2283/');await page.fill('#code',fs.readFileSync(path.join(process.env.IMMICH_TEST_STATE,'access-code.txt'),'utf8').trim());await page.click('#loginForm button');await page.waitForSelector('#library:not(.hidden)');
 const api=async(u,b)=>page.evaluate(async({u,b})=>{const r=await fetch(u,b===undefined?{}:{method:'POST',headers:{'X-Immich-Request':'1'},body:JSON.stringify(b)});if(!r.ok)throw Error('Hidden media HTTP '+r.status);return r.json()},{u,b});
 const snapshot=o=>({record:o.assets.find(x=>x.asset.id===id),faces:o.faces.filter(f=>f.asset_id===id)});
 const originalHash=async()=>{const r=await page.request.get('http://127.0.0.1:2283/api/assets/'+id+'/original');assert(r.ok());return crypto.createHash('sha256').update(await r.body()).digest('hex')};
 const hiddenPage=async()=>{await Promise.all([page.waitForResponse(r=>r.url().includes('/api/gallery?')&&r.url().includes('visibility=hidden')),page.click('#hiddenMedia')]);await page.locator('.card[data-asset="'+id+'"]').waitFor();assert(await page.locator('#hiddenHelp').isVisible())};
 const location=async()=>{await page.locator('.card[data-asset="'+id+'"]').getByRole('button',{name:'Managed file location',exact:true}).click();await page.locator('#managedLocation').waitFor();const expected=path.resolve(process.env.IMMICH_TEST_STATE,'originals',id);assert.equal(await page.inputValue('#locationPath'),expected);await page.click('#copyLocation');await page.waitForFunction(()=>document.querySelector('#locationStatus').textContent==='Path copied.');assert.equal(await page.evaluate(()=>navigator.clipboard.readText()),expected);await page.setViewportSize({width:390,height:844});assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);await page.keyboard.press('Escape');assert.equal(await page.locator('#managedLocation').isVisible(),false)};
 const phase=process.argv[2];
 if(phase==='prepare'){
  const baseline=snapshot(await api('/api/organization'));assert(baseline.record);assert.equal(await originalHash(),id);
  await api('/api/assets/'+id+'/hide',{});await hiddenPage();await location();fs.writeFileSync(saved,JSON.stringify(baseline));
 }else if(phase==='finish'){
  await hiddenPage();await location();assert.equal(await originalHash(),id);
  await Promise.all([page.waitForResponse(r=>r.url().endsWith('/api/assets/'+id+'/unhide')&&r.request().method()==='POST'),page.locator('.card[data-asset="'+id+'"]').getByRole('button',{name:'Restore to library',exact:true}).click()]);
  const o=await api('/api/organization');assert.deepEqual(snapshot(o),JSON.parse(fs.readFileSync(saved)));assert.equal(await originalHash(),id);
  const r=await page.request.get('http://127.0.0.1:2283/api/assets/'+id+'/managed-location');assert.equal(r.status(),409);
 }else throw Error('Unknown phase');assert.deepEqual(errors,[]);await browser.close();console.log('PASS: hidden media location, clipboard, narrow dialog and persisted review '+phase);
})().catch(e=>{console.error(e.message);process.exit(1)});
