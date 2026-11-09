const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const { chromium } = require(process.env.IMMICH_BROWSER_MODULE);

const origin = process.env.IMMICH_BROWSER_ORIGIN || 'http://127.0.0.1:2283';
const marker = 'PhotoComparisonFixture';

async function post(page, endpoint, data) {
  const response = await page.request.post(origin + endpoint, {
    headers: { 'X-Immich-Request': '1' }, data
  });
  assert(response.ok(), endpoint + ' returned ' + response.status());
  return response.json();
}

async function metadata(page, id) {
  const response = await page.request.get(origin + '/api/metadata/asset/' + id);
  assert(response.ok(), 'Photo metadata returned ' + response.status());
  return response.json();
}

function card(page, id) {
  return page.locator('#grid [data-asset="' + id + '"]');
}

function waitGallery(page, total, number = 1) {
  return page.waitForFunction(({total, number}) => {
    const shown = Math.min(60, total - (number - 1) * 60);
    return document.querySelector('#status').textContent !== 'Loading media…'
      && document.querySelector('#pageLabel').textContent === 'Page ' + number + ' · ' + shown + ' shown / ' + total
      && document.querySelectorAll('#grid .card').length === shown;
  }, {total, number});
}

async function main() {
  const browser = await chromium.launch({channel:'msedge', headless:true});
  try {
    const page = await browser.newPage({viewport:{width:1280,height:850}});
    page.setDefaultTimeout(30_000);
    const errors = [];
    page.on('pageerror', error => errors.push(error.message));
    await page.goto(origin);
    await page.fill('#code', fs.readFileSync(path.join(process.env.IMMICH_TEST_STATE, 'access-code.txt'), 'utf8').trim());
    await page.click('#loginForm button');
    await page.waitForSelector('#library:not(.hidden)');

    const png = Buffer.from(await page.evaluate(() => {
      const canvas = document.createElement('canvas');
      canvas.width = 180;
      canvas.height = 120;
      const context = canvas.getContext('2d');
      context.fillStyle = '#cb6633';
      context.fillRect(0, 0, 180, 120);
      context.fillStyle = '#2266ab';
      context.fillRect(80, 20, 50, 60);
      return canvas.toDataURL('image/png').split(',')[1];
    }), 'base64');
    const photos = [];
    for (let i = 0; i < 62; i++) {
      const name = marker + '-' + String(i).padStart(2, '0') + '.png';
      const result = await post(page, '/api/upload?name=' + name, Buffer.concat([png, Buffer.from(name)]));
      photos.push(result.asset);
    }

    await page.goto(origin + '/?' + new URLSearchParams({sort:'name', q:marker}));
    const compare = page.getByRole('button', {name:'Compare two photos'});
    const dialog = page.getByRole('dialog', {name:'Compare photos'});
    const select = photo => card(page, photo.id).locator('.organizationPick').check();
    const transforms = () => dialog.locator('img').evaluateAll(images => images.map(img => img.style.transform));
    const [first, second, third, fourth] = photos;
    const last = photos.at(-1);
    await waitGallery(page, 62);
    await page.getByRole('checkbox', {name:'Select items'}).check();
    await select(first);
    assert(await compare.isDisabled());
    await page.getByRole('button', {name:'Next page'}).click();
    await waitGallery(page, 62, 2);
    await select(last);
    await compare.click();
    await page.waitForFunction(() => [...document.querySelectorAll('#comparePanes img')].filter(i => i.naturalWidth).length === 2);
    assert.deepEqual(await dialog.locator('h3').allTextContents(), [first.name, last.name]);
    assert.deepEqual(await dialog.locator('a').evaluateAll(links => links.map(link => new URL(link.href).pathname)),
      [first, last].map(photo => '/api/assets/' + photo.id + '/original'));
    const zoomIn = dialog.getByRole('button', {name:'Zoom in both photos'});
    await zoomIn.click();
    await zoomIn.click();
    const centered = await transforms();
    assert(centered.every(transform => transform.includes('scale(1.5625)')));
    const box = await dialog.locator('.compareImage').first().boundingBox();
    const x = box.x + box.width / 2, y = box.y + box.height / 2;
    await page.mouse.move(x, y);
    await page.mouse.down();
    await page.mouse.move(x + 38, y + 25, {steps:5});
    await page.mouse.up();
    assert((await transforms()).every((transform, i) => transform !== centered[i]));
    await page.keyboard.press('0');
    assert((await transforms()).every(transform => transform === 'translate(0px, 0px) scale(1)'));
    await page.setViewportSize({width:390,height:760});
    assert(await dialog.evaluate(element => element.scrollWidth <= element.clientWidth + 1));
    await page.keyboard.press('Escape');
    await dialog.waitFor({state:'hidden'});
    await waitGallery(page, 62, 2);
    assert.match(await page.locator('#organizationSelected').textContent(), /^2 selected/);
    await page.setViewportSize({width:1280,height:850});

    await page.selectOption('#favoriteScope', 'false');
    await waitGallery(page, 62);
    await select(first);
    await select(second);
    await compare.click();
    const favorite = dialog.getByRole('button', {name:'Favorite ' + first.name, exact:true});
    await page.waitForFunction(() => !document.querySelector('#comparePanes button')?.disabled
      && document.querySelector('#comparePanes button')?.getAttribute('aria-pressed') === 'false');
    const before = await metadata(page, first.id);
    await post(page, '/api/metadata/edit', {
      action:'set_asset', asset_id:first.id, added:before.added, expected:before.token,
      favorite:false, title:'Concurrent title edit', caption:'Keep this caption', notes:''
    });
    await favorite.click();
    await page.getByText('Organization changed. Reload before saving.').waitFor();
    assert.equal((await metadata(page, first.id)).favorite, false);

    // Close while the retry is still pending. Reconciliation must use its result.
    let releaseSave;
    const pendingSave = new Promise(resolve => { releaseSave = resolve; });
    await page.route('**/api/metadata/edit', async route => {
      await pendingSave;
      await route.continue();
    }, {times:1});
    try {
      await Promise.all([
        page.waitForRequest(request => new URL(request.url()).pathname === '/api/metadata/edit'
          && request.method() === 'POST'),
        favorite.click()
      ]);
      await page.keyboard.press('Escape');
      await dialog.waitFor({state:'hidden'});
    } finally {
      releaseSave();
    }
    await waitGallery(page, 61);
    const saved = await metadata(page, first.id);
    assert.equal(saved.favorite, true);
    assert.equal(saved.title, 'Concurrent title edit');
    assert.equal(saved.caption, 'Keep this caption');
    assert.match(await page.locator('#organizationSelected').textContent(), /^1 selected/);
    assert(await compare.isDisabled());
    assert.equal(await card(page, first.id).count(), 0);
    assert(await card(page, second.id).locator('.organizationPick').isChecked());

    // An edit outside this view can hide a selected photo.
    await page.selectOption('#favoriteScope', '');
    await waitGallery(page, 62);
    await select(third);
    await select(fourth);
    await post(page, '/api/assets/' + third.id + '/hide', {});
    await compare.click();
    await page.getByText('The gallery changed. Close comparison and select the photos again.').waitFor();
    await dialog.getByRole('button', {name:'Close', exact:true}).click();
    await waitGallery(page, 61);
    assert.match(await page.locator('#organizationSelected').textContent(), /^0 selected/);
    assert(await compare.isDisabled());
    assert.deepEqual(errors, []);
    console.log('PASS: cross-page photo comparison, linked zoom and drag, narrow layout, Escape during a save, filtered selection, concurrent caption and stale gallery recovery');
  } finally {
    await browser.close();
  }
}

main().catch(error => { console.error(error.stack); process.exitCode = 1; });
