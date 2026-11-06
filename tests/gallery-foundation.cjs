// Run after gallery-search.cjs against the same disposable service/catalog.
const fs=require('fs'),path=require('path'),assert=require('assert/strict');
const {chromium}=require(process.env.IMMICH_BROWSER_MODULE);
(async()=>{const browser=await chromium.launch({channel:'msedge',headless:true});try{
 const page=await browser.newPage({viewport:{width:1280,height:900}}),errors=[];page.on('pageerror',e=>errors.push(e.message));page.setDefaultTimeout(30000);
 await page.goto('http://127.0.0.1:2283');await page.fill('#code',fs.readFileSync(path.join(process.env.IMMICH_TEST_STATE,'access-code.txt'),'utf8').trim());await page.click('#loginForm button');await page.waitForSelector('#library:not(.hidden)');
 const search=async(q)=>{await page.fill('#filenameFilter','');await page.fill('#galleryQuery',q);await page.locator('#gallerySearch button[type=submit]').click();await page.waitForFunction(q=>document.querySelector('#gallerySearchState').textContent.includes('matching')&&document.querySelector('#galleryQuery').value===q,q);};
 await search('SearchTrip');await page.click('#nextPage');await page.waitForFunction(()=>document.querySelector('#pageLabel').textContent.startsWith('Page 2'));
 const second=await page.locator('.card').evaluateAll(cs=>cs.map(c=>c.dataset.asset));assert(second.length>0);
 // Viewer shares the exact query and snapshot, and its metadata request is bounded.
 await page.locator('.openMedia').first().click();await page.waitForFunction(()=>galleryContext.viewer?.neighbors);
 assert.equal(await page.evaluate(()=>galleryContext.viewer.neighbors.current.id),second[0]);
 let refreshes=0;page.on('request',r=>{if(new URL(r.url()).pathname==='/api/gallery')refreshes++});
 await page.waitForTimeout(5500);assert.equal(refreshes,0);assert.equal(await page.locator('#viewer').evaluate(e=>e.open),true);
 await page.click('#closeViewer');assert.match(await page.locator('#pageLabel').textContent(),/^Page 2/);assert.equal(await page.locator('.openMedia').first().evaluate(e=>e===document.activeElement),true);
 await search('Alice');await page.goBack();await page.waitForFunction(()=>document.querySelector('#galleryQuery').value==='SearchTrip'&&document.querySelector('#pageLabel').textContent.startsWith('Page 2'));assert.deepEqual(await page.locator('.card').evaluateAll(cs=>cs.map(c=>c.dataset.asset)),second);
 await page.goForward();await page.waitForFunction(()=>document.querySelector('#galleryQuery').value==='Alice'&&document.querySelectorAll('.card').length===1);
 await page.fill('#filenameFilter','search-000');await page.locator('#gallerySearch button[type=submit]').click();await page.waitForFunction(()=>document.querySelectorAll('.card').length===1&&document.querySelector('.name').textContent==='search-000.png');
 await page.selectOption('#mediaFilter','video');await page.waitForFunction(()=>document.querySelectorAll('.card').length===0&&document.querySelector('#pageLabel').textContent.endsWith('/ 0'));
 await page.reload();await page.waitForFunction(()=>document.querySelector('#library').classList.contains('hidden')===false);assert.equal(await page.inputValue('#mediaFilter'),'video');assert.equal(await page.inputValue('#filenameFilter'),'search-000');
 await page.selectOption('#mediaFilter','photo');await page.waitForSelector('.card');
 // An edit from another client invalidates both continuation and viewer context.
 const id=await page.locator('.card').getAttribute('data-asset');await page.request.post('http://127.0.0.1:2283/api/assets/'+id+'/tags',{headers:{'X-Immich-Request':'1'},data:{tags:['SearchTrip','concurrent']}});
 await page.locator('.openMedia').click();await page.waitForFunction(()=>document.querySelector('#viewerState').textContent.includes('results changed'));await page.click('#closeViewer');
 await search('SearchTrip');await page.selectOption('#visibility','hidden');await page.waitForSelector('.hiddenPick');await page.locator('.hiddenPick').check();assert.equal(await page.locator('#hiddenSelected').textContent(),'1 selected');
 await search('Alice');await page.waitForFunction(()=>document.querySelector('#hiddenSelected').textContent==='0 selected');
 for(const width of [1280,390]) {await page.setViewportSize({width,height:900});assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);if(process.env.IMMICH_BROWSER_OUTPUT)await page.screenshot({path:path.join(process.env.IMMICH_BROWSER_OUTPUT,'gallery-foundation-'+width+'.png'),fullPage:true});}
 assert.deepEqual(errors,[]);console.log('PASS: shared gallery/viewer context, suspended refresh, focus/page return, browser back/forward, combined filename/media, bookmark/reload, concurrent edit and selection reset');
}finally{await browser.close()}})().catch(e=>{console.error(e.stack);process.exit(1)});
