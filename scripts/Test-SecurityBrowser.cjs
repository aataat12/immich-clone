const {chromium}=require(process.env.IMMICH_BROWSER_MODULE || 'playwright');
const fs=require('fs'),os=require('os'),path=require('path'),{spawn}=require('child_process'),assert=require('assert/strict');
(async()=>{const root=fs.mkdtempSync(path.join(os.tmpdir(),'immich-security-'));const child=spawn(process.env.IMMICH_SECURITY_APP,['-data',path.join(root,'state'),'-logs',path.join(root,'logs'),'-listen','127.0.0.1:24831']);let browser;
try{for(let i=0;i<100;i++){try{if((await fetch('http://127.0.0.1:24831/health')).ok)break}catch{}await new Promise(r=>setTimeout(r,50));}
browser=await chromium.launch({executablePath:process.env.IMMICH_BROWSER_EXE,headless:true,args:['--no-sandbox']});const context=await browser.newContext();
const code=fs.readFileSync(path.join(root,'state','access-code.txt'),'utf8').trim();assert.equal((await context.request.post('http://127.0.0.1:24831/api/login',{headers:{'X-Immich-Request':'1'},data:{code}})).status(),200);
for(const width of [1280,390]){const page=await context.newPage();await page.setViewportSize({width,height:900});const errors=[];page.on('pageerror',e=>errors.push(e.message));
for(const route of ['/','/people','/import','/backups','/build-notes']){await page.goto('http://127.0.0.1:24831'+route);await page.waitForFunction(()=>document.querySelector('#version')?.textContent.includes('prototype'));await page.waitForTimeout(150);assert.deepEqual(errors,[]);assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
await page.evaluate(()=>{window.violations=[];document.addEventListener('securitypolicyviolation',e=>window.violations.push(e.effectiveDirective));const s=document.createElement('script');s.textContent='window.injectedSecurityCanary = true';document.body.append(s);const b=document.createElement('button');b.setAttribute('onclick','window.injectedSecurityCanary = true');document.body.append(b);b.click();});
await page.waitForFunction(()=>window.violations.length>=2);assert.equal(await page.evaluate(()=>window.injectedSecurityCanary),undefined);
}await page.close();}
const page=await context.newPage();const prior=(await context.cookies('http://127.0.0.1:24831')).find(c=>c.name==='session');
await page.goto('http://127.0.0.1:24831/logout');await page.locator('#logoutConfirmation:not(.hidden)').waitFor();
assert.equal((await context.request.get('http://127.0.0.1:24831/api/assets')).status(),200);
assert.equal((await context.request.post('http://127.0.0.1:24831/api/logout',{headers:{'X-Immich-Request':'1',Origin:'https://attacker.invalid'}})).status(),403);
await page.click('#confirmLogout');await page.waitForURL('**/login');await page.locator('#login:not(.hidden)').waitFor();
assert.equal((await context.request.get('http://127.0.0.1:24831/api/assets')).status(),401);
const replay=await browser.newContext();await replay.addCookies([prior]);assert.equal((await replay.request.get('http://127.0.0.1:24831/api/assets')).status(),401);await replay.close();
await page.fill('#code','0'.repeat(64));const denied=page.waitForResponse(r=>r.url().endsWith('/api/login'));await page.click('#loginForm button');assert.equal((await denied).status(),401);await page.waitForFunction(()=>document.querySelector('#loginError').textContent.includes('incorrect access code'));
await page.fill('#code',code);await page.click('#loginForm button');await page.locator('#library:not(.hidden)').waitFor();assert.equal((await context.request.get('http://127.0.0.1:24831/api/assets')).status(),200);await page.reload();await page.locator('#library:not(.hidden)').waitFor();
await page.goto('http://127.0.0.1:24831/login');await page.locator('#login:not(.hidden)').waitFor();assert.equal((await context.request.get('http://127.0.0.1:24831/api/assets')).status(),200);await page.close();
console.log('PASS: explicit login/logout confirmation, foreign-origin rejection, server-side revocation and old-cookie rejection, incorrect-code rejection, re-login and reload persistence');
console.log('PASS: all five pages, desktop/narrow, bundled scripts run; injected scripts and event attributes blocked');
}finally{if(browser)await browser.close();child.kill('SIGTERM');await new Promise(r=>child.once('exit',r));fs.rmSync(root,{recursive:true,force:true});}})().catch(e=>{console.error(e);process.exitCode=1;});
