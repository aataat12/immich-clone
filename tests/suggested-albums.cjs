const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const { chromium } = require(process.env.IMMICH_BROWSER_MODULE);

const origin = process.env.IMMICH_BROWSER_ORIGIN || 'http://127.0.0.1:2283';
const albumTitle = 'Reviewed mixed-media trip';

async function fixtures(page) {
  const png = Buffer.from(await page.evaluate(() => {
    const canvas = document.createElement('canvas');
    canvas.width = 160;
    canvas.height = 120;
    const context = canvas.getContext('2d');
    context.fillStyle = '#2a7088';
    context.fillRect(0, 0, canvas.width, canvas.height);
    return canvas.toDataURL().split(',')[1];
  }), 'base64');
  const mp4 = process.env.IMMICH_VIEWER_VIDEO ? fs.readFileSync(process.env.IMMICH_VIEWER_VIDEO)
    : Buffer.from(await page.evaluate(async () => {
      const canvas = document.createElement('canvas');
      canvas.width = 160;
      canvas.height = 120;
      const context = canvas.getContext('2d');
      const stream = canvas.captureStream(15);
      const mimeType = 'video/mp4;codecs=avc1.42001f';
      const recorder = new MediaRecorder(stream, {mimeType});
      const chunks = [];
      recorder.ondataavailable = event => chunks.push(event.data);
      const stopped = new Promise(resolve => recorder.onstop = resolve);
      recorder.start();
      for (let frame = 0; frame < 15; frame++) {
        context.fillStyle = frame % 2 ? 'navy' : 'orange';
        context.fillRect(0, 0, 160, 120);
        await new Promise(resolve => setTimeout(resolve, 67));
      }
      recorder.stop();
      await stopped;
      stream.getTracks().forEach(track => track.stop());
      const bytes = new Uint8Array(await new Blob(chunks).arrayBuffer());
      return btoa(Array.from(bytes, b => String.fromCharCode(b)).join(''));
    }), 'base64');
  const names = [
    '1987-06-01 08.00.00.jpg', '1987-06-02 14.00.00.jpg', 'VID_19870602_150000.mp4',
    '1987-06-02 16.00.00.jpg', '1987-06-03 09.30.00.jpg', '1987-06-03 10.00.00.jpg',
    '1987-06-03 11.00.00.jpg', '1987-06-04 18.00.00.jpg', 'UndatedSuggestionFixture.png'
  ];
  const assets = [];
  for (const name of names) {
    const data = name.endsWith('.mp4') ? mp4 : Buffer.concat([png, Buffer.from(name)]);
    const response = await page.request.post(origin + '/api/upload?name=' + encodeURIComponent(name),
      {headers:{'X-Immich-Request':'1'}, data});
    assert(response.ok(), 'upload returned ' + response.status());
    assets.push((await response.json()).asset);
  }
  return assets;
}

async function main() {
  const browser = await chromium.launch({channel:'msedge', headless:true});
  try {
    const page = await browser.newPage({viewport:{width:1120,height:820}});
    page.setDefaultTimeout(15_000);
    const errors = [];
    page.on('pageerror', error => errors.push(error.message));
    assert.equal((await page.request.get(origin + '/api/album-suggestions?kind=event')).status(), 401);
    await page.goto(origin);
    await page.fill('#code', fs.readFileSync(path.join(process.env.IMMICH_TEST_STATE, 'access-code.txt'), 'utf8').trim());
    await page.click('#loginForm button');
    await page.waitForSelector('#library:not(.hidden)');
    const items = await fixtures(page);
    assert.equal(items[2].mime, 'video/mp4');

    await page.getByRole('button', {name:'Suggested albums', exact:true}).click();
    const dialog = page.getByRole('dialog', {name:'Suggested albums'});
    await dialog.getByRole('button', {name:/1987-06-02 14:00.*3 items/}).waitFor();
    await dialog.getByLabel('Group by', {exact:true}).selectOption('trip');
    await dialog.getByRole('button', {name:/1987-06-02 14:00.*1987-06-03 11:00.*6 items/}).click();
    const strip = dialog.getByRole('group', {name:'Media in capture order'});
    const tile = name => strip.getByRole('button', {name:new RegExp(name.replace(/[.]/g, '\\.'))});
    await tile(items[0].name).waitFor();
    assert.match(await tile(items[0].name).getAttribute('aria-label'), /Before$/);
    assert.match(await tile(items[7].name).getAttribute('aria-label'), /After$/);
    assert.equal(await strip.getByRole('button').count(), 8);
    await dialog.getByRole('button', {name:'Review end', exact:true}).click();
    assert.equal(await tile(items[6].name).getAttribute('aria-pressed'), 'true');
    await page.keyboard.press('ArrowRight');
    await dialog.getByRole('button', {name:'Include in album', exact:true}).click();
    assert.match(await tile(items[7].name).getAttribute('aria-label'), /End$/);

    await tile(items[2].name).click();
    await page.waitForFunction(() => document.querySelector('#suggestionPreview video')?.readyState >= 1);
    assert.match(await page.locator('#suggestionMediaDate').textContent(), /From filename · Video/);
    await tile(items[3].name).click();
    await dialog.getByRole('button', {name:'Exclude from album', exact:true}).click();
    assert.match(await tile(items[3].name).getAttribute('aria-label'), /Excluded$/);
    assert.match(await page.locator('#suggestionRange').textContent(), /^6 selected.*1987-06-04 18:00$/);
    await page.setViewportSize({width:390,height:820});
    assert(await dialog.evaluate(el => el.scrollWidth <= el.clientWidth + 1));
    await dialog.getByLabel('Album title').fill(albumTitle);

    // Commit the real write, then simulate losing its response.
    let committed;
    await page.route('**/api/metadata/edit', async route => {
      const response = await route.fetch();
      committed = response.ok() ? await response.json() : null;
      await route.abort('failed');
    }, {times:1});
    await dialog.getByRole('button', {name:'Save as album'}).click();
    await page.getByText(/You can retry the save/).waitFor();
    assert(committed?.id);
    await dialog.getByRole('button', {name:'Save as album'}).click();
    await dialog.waitFor({state:'hidden'});
    await page.waitForFunction(id => new URL(location.href).searchParams.get('album') === id
      && document.querySelectorAll('#grid .card').length === 6, committed.id);
    const result = await page.request.get(origin + '/api/metadata/albums/' + committed.id);
    assert(result.ok());
    assert.deepEqual((await result.json()).members.map(member => member.asset_id),
      [1, 2, 4, 5, 6, 7].map(i => items[i].id));
    assert.deepEqual(errors, []);
    console.log('PASS: events and multi-day groups, photo/video review with adjacent exclusions, edited chronological membership, and replay-safe album save');
  } finally {
    await browser.close();
  }
}

main().catch(error => { console.error(error.stack); process.exitCode = 1; });
