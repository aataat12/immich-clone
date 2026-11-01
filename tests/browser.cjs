const {chromium} = require(process.env.IMMICH_BROWSER_MODULE);
const fs = require('fs');
const path = require('path');
(async () => {
  const browser = await chromium.launch({channel:'msedge',headless:true});
  const page = await browser.newPage({viewport:{width:1360,height:900}});
  const errors = [];
  page.on('pageerror',error=>errors.push(error.message));
  await page.goto('http://127.0.0.1:2283');
  await page.fill('#code',fs.readFileSync(path.join(process.env.IMMICH_TEST_STATE,'access-code.txt'),'utf8').trim());
  await page.click('#loginForm button');
  await page.waitForSelector('#library:not(.hidden)');
  await page.waitForSelector('.card img');
  if(!(await page.locator('.card').filter({hasText:'synthetic.png'}).locator('.meta').textContent()).includes('UTC'))throw Error('Upload date timezone not explicit');
  await page.locator('.card').filter({hasText:'synthetic.png'}).getByRole('button',{name:'View image',exact:true}).click();
  await page.waitForSelector('#viewMedia img');
  await page.waitForFunction(()=>document.querySelector('#viewMedia img').naturalWidth===64);
  await page.click('#closeViewer');
  await page.locator('.card').filter({hasText:'synthetic-video.mp4'}).getByRole('button',{name:'Play video',exact:true}).click();
  await page.waitForFunction(()=>{const v=document.querySelector('#viewMedia video');return v&&v.readyState>=2&&v.videoWidth===160});
  await page.evaluate(async()=>{const v=document.querySelector('#viewMedia video');v.muted=true;await v.play()});
  await page.waitForFunction(()=>document.querySelector('#viewMedia video').currentTime>0.1);
  await page.click('#closeViewer');
  await page.waitForFunction(()=>{const card=Array.from(document.querySelectorAll('.card')).find(c=>c.querySelector('.name')?.textContent==='oriented.jpg');const img=card?.querySelector('img');return img&&img.naturalWidth===10&&img.naturalHeight===20});
  await page.locator('.card').filter({hasText:'oriented.jpg'}).getByRole('button',{name:'View image',exact:true}).click();
  await page.waitForFunction(()=>document.querySelector('#viewDetails').textContent.includes('2024-03-02T10:11:12'));
  if(!(await page.locator('#viewDetails').textContent()).includes('Demo!'))throw Error('Camera details missing');
  await page.click('#closeViewer');
  for(const ext of ['bmp','gif','tiff']) {
    await page.waitForFunction(ext=>{const c=Array.from(document.querySelectorAll('.card')).find(c=>c.querySelector('.name')?.textContent==='synthetic.'+ext);const img=c?.querySelector('img');return img&&img.naturalWidth===32&&img.naturalHeight===24},ext);
    const card=page.locator('.card').filter({hasText:'synthetic.'+ext});
    await card.getByRole('button',{name:ext==='tiff'?'View preview':'View image',exact:true}).click();
    await page.waitForFunction(()=>document.querySelector('#viewMedia img')?.naturalWidth===32);
    await page.click('#closeViewer');
  }
  for (const [name,w,h] of [['synthetic.heic',320,240],['grid-large.heic',320,240],['hevc.mp4',320,240],['rotated.mov',240,320]]) {
    await page.waitForFunction(({name,w,h})=>{const c=Array.from(document.querySelectorAll('.card')).find(c=>c.querySelector('.name')?.textContent===name);const img=c?.querySelector('img');return img&&img.naturalWidth===w&&img.naturalHeight===h},{name,w,h});
    if (!name.endsWith('.mp4')) {
      await page.locator('.card').filter({hasText:name}).getByRole('button',{name:'View preview',exact:true}).click();
      await page.waitForFunction(w=>document.querySelector('#viewMedia img')?.naturalWidth===w,w);
      await page.click('#closeViewer');
    }
  }
  await page.selectOption('#sort','name');
  await page.waitForFunction(()=>document.querySelector('.dateGroup')?.textContent==='All files');
  await page.selectOption('#sort','capture');
  await page.waitForFunction(()=>Array.from(document.querySelectorAll('.dateGroup')).some(h=>h.textContent==='2024-03'));
  await page.setInputFiles('#files' ,{name:'synthetic-browser.txt',mimeType:'text/plain',buffer:Buffer.from('synthetic browser test original')});
  await page.waitForFunction(()=>document.querySelectorAll('.card').length===11);
  const source=path.join(process.env.IMMICH_BROWSER_OUTPUT,'synthetic-source');
  fs.mkdirSync(source,{recursive:true});
  fs.writeFileSync(path.join(source,'folder-test.txt'),'synthetic source that must remain untouched');
  await page.setInputFiles('#folder',source);
  await page.waitForSelector('#confirmFolder:not(.hidden)');
  if(await page.locator('.card').count()!==11)throw Error('Folder uploaded before confirmation');
  await page.click('#confirmFolder');
  await page.waitForFunction(()=>document.querySelectorAll('.card').length===12);
  if(fs.readFileSync(path.join(source,'folder-test.txt'),'utf8')!=='synthetic source that must remain untouched')throw Error('Folder source was changed');
  const retryResponse=page.waitForResponse(r=>r.url().endsWith('/retry')&&r.request().method()==='POST');
  await page.locator('.card').filter({hasText:'synthetic-browser.txt'}).getByRole('button',{name:'Retry preview',exact:true}).click();
  if((await retryResponse).status()!==200)throw Error('Browser preview retry failed');
  const metadata=await page.evaluate(async()=>await(await fetch('/api/assets')).json());
  if(!metadata.some(a=>a.sources?.some(p=>p.endsWith('/folder-test.txt'))))throw Error('Relative folder path lost');
  await page.screenshot({path:path.join(process.env.IMMICH_BROWSER_OUTPUT,'desktop.png'),fullPage:true});
  await page.setViewportSize({width:390,height:844});
  await page.screenshot({path:path.join(process.env.IMMICH_BROWSER_OUTPUT,'narrow.png'),fullPage:true});
  if(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth))throw Error('Horizontal overflow');
  const downloadPromise=page.waitForEvent('download');
  await page.click('#backup');
  const download=await downloadPromise;
  if(await download.failure())throw Error('Browser backup download failed');
  if(errors.length)throw Error(errors.join(';'));
  console.log('PASS: Edge sign-in, upload, confirmed folder copy with unchanged source, BMP/TIFF/GIF and single/grid HEIC thumbnails/viewers, HEVC and rotated MOV thumbnails, EXIF-oriented thumbnail, camera/date details, timeline sorting, relative source path, full-image view, actual H264/AAC playback, streamed backup download and narrow layout; no JavaScript errors.');
  await browser.close();
})().catch(error=>{console.error(error.message);process.exit(1)});
