// Actual service metadata and HTTP backup proof, synthetic catalog only.
const fs=require('fs'),path=require('path'),assert=require('assert/strict'),crypto=require('crypto');
const {chromium}=require(process.env.IMMICH_BROWSER_MODULE);
(async()=>{const browser=await chromium.launch({channel:'msedge',headless:true});try{
 const page=await browser.newPage();await page.goto('http://127.0.0.1:2283');await page.fill('#code',fs.readFileSync(path.join(process.env.IMMICH_TEST_STATE,'access-code.txt'),'utf8').trim());await page.click('#loginForm button');await page.waitForSelector('#library:not(.hidden)');
 const get=async u=>{const r=await page.request.get('http://127.0.0.1:2283'+u);assert(r.ok(),u+' '+r.status());return r.json()};
 const edit=async input=>{const r=await page.request.post('http://127.0.0.1:2283/api/metadata/edit',{headers:{'X-Immich-Request':'1'},data:input});assert(r.ok(),input.action+' '+r.status()+' '+await r.text());return r.json()};
 const p=await get('/api/gallery?sort=name&visibility=library&limit=100&q=ViewerTrip'),assets=p.assets.slice(0,65);assert.equal(assets.length,65);
 let metadata=await get('/api/metadata/asset/'+assets[0].id);const stale={action:'set_asset',asset_id:metadata.asset_id,added:metadata.added,expected:metadata.token,favorite:true,title:'Family <trip>',caption:'Résumé & reunion\n第二行',notes:'Private synthetic note'};metadata=await edit(stale);
 const conflict=await page.request.post('http://127.0.0.1:2283/api/metadata/edit',{headers:{'X-Immich-Request':'1'},data:stale});assert.equal(conflict.status(),409);
 const priorAlbums=Object.keys((await get('/api/export')).curation.albums_v3).length;
 const albumID=crypto.randomBytes(32).toString('hex'),emptyID=crypto.randomBytes(32).toString('hex');let album=await edit({action:'create_album',album_id:albumID,title:'Trip',description:'Synthetic recovery album'});await edit({action:'create_album',album_id:emptyID,title:'Empty'});
 const repeated=await edit({action:'create_album',album_id:albumID,title:'Trip',description:'Synthetic recovery album'});assert.equal(repeated.id,albumID);
 for(const a of assets)album=await edit({action:'add_member',album_id:albumID,asset_id:a.id,added:a.added,expected:album.token});
 album=await edit({action:'move_member',album_id:albumID,asset_id:assets[64].id,other_id:assets[0].id,expected:album.token});album=await edit({action:'set_cover',album_id:albumID,asset_id:assets[0].id,expected:album.token});
 const first=await get('/api/metadata/albums/'+albumID+'?limit=60'),second=await get('/api/metadata/albums/'+albumID+'?limit=60&after='+encodeURIComponent(first.next));assert.equal(first.members[0].asset_id,assets[64].id);assert.equal(new Set([...first.members,...second.members].map(m=>m.asset_id)).size,65);assert.equal((await get('/api/metadata/albums/'+emptyID)).count,0);
 const exported=await get('/api/export');assert.equal(exported.catalog_schema,'3');assert.equal(exported.curation.version,1);assert.equal(Object.keys(exported.curation.albums_v3).length,priorAlbums+2);
 const root=path.dirname(process.env.IMMICH_TEST_STATE);fs.writeFileSync(path.join(root,'expected-curation.json'),JSON.stringify(exported.curation));const backup=await page.request.get('http://127.0.0.1:2283/api/backup');assert(backup.ok());fs.writeFileSync(path.join(root,'metadata-backup.zip'),await backup.body());
 for(const a of assets.slice(0,3)){const original=await page.request.get('http://127.0.0.1:2283/api/assets/'+a.id+'/original');assert.equal(crypto.createHash('sha256').update(await original.body()).digest('hex'),a.id);}
 console.log('PASS: actual service favorite/text/version-3 storage, stale edit refusal, stable create identity, empty album, ordered 65-member paging, full organization export, HTTP backup and original hashes');
}finally{await browser.close()}})().catch(e=>{console.error(e.stack);process.exit(1)});
