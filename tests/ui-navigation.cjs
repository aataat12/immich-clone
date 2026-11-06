// Run against the existing disposable application fixture, never the user's library.
// Same environment contract as tests/browser.cjs. No application mutations.
const { chromium } = require(process.env.IMMICH_BROWSER_MODULE);
const fs = require('fs');
const path = require('path');
const assert = require('assert/strict');
(async () => {
  assert(process.env.IMMICH_TEST_STATE, 'Disposable fixture state is required');
  const browser = await chromium.launch({ channel: 'msedge', headless: true });
  try {
    const page = await browser.newPage({ viewport: { width: 1360, height: 900 } });
    const errors = [];
    page.on('pageerror', error => errors.push(error.message));
    await page.goto('http://127.0.0.1:2283/');
    await page.fill('#code', fs.readFileSync(path.join(process.env.IMMICH_TEST_STATE, 'access-code.txt'), 'utf8').trim());
    await page.click('#loginForm button');
    await page.waitForSelector('#library:not(.hidden)');
    const galleryResponse = await page.request.get('http://127.0.0.1:2283/api/gallery?sort=capture&visibility=library&limit=100');
    assert.equal(galleryResponse.status(), 200);
    const gallery = await galleryResponse.json();
    assert(gallery.count > 0, 'Existing synthetic gallery is required');
    await page.getByRole('searchbox', { name: 'Search names, filenames or tags', exact: true }).fill('UI-search-no-match');
    await page.locator('#gallerySearch').getByRole('button', { name: 'Find', exact: true }).click();
    await page.waitForFunction(() => document.querySelector('#gallerySearchState').textContent.includes('0 matching items') && document.querySelector('#status').textContent.includes('No matching media'));
    assert.equal(new URL(page.url()).pathname, '/');
    assert.equal(new URL(page.url()).searchParams.get('q'), 'UI-search-no-match');
    assert.equal(await page.locator('#galleryQuery').inputValue(), 'UI-search-no-match');
    assert.equal(await page.locator('.card').count(), 0);
    assert.equal(await page.locator('#clearGallerySearch').isVisible(), true);
    await page.locator('#galleryQuery').fill('synthetic.png');
    await page.locator('#gallerySearch').getByRole('button', { name: 'Find', exact: true }).click();
    await page.waitForFunction(() => document.querySelector('#gallerySearchState').textContent.includes('1 matching item') && document.querySelectorAll('.card').length === 1 && document.querySelector('.card .name').textContent === 'synthetic.png');
    const matchingAsset = await page.locator('.card').getAttribute('data-asset');
    const matchingPage = await page.locator('#pageLabel').textContent();
    await page.locator('.card').getByRole('button', { name: 'View image', exact: true }).click();
    await page.waitForFunction(() => document.querySelector('#viewMedia img')?.naturalWidth === 64);
    await page.locator('#closeViewer').click();
    assert.equal(new URL(page.url()).pathname, '/');
    assert.equal(new URL(page.url()).searchParams.get('q'), 'synthetic.png');
    assert.equal(await page.locator('#galleryQuery').inputValue(), 'synthetic.png');
    assert.match(await page.locator('#gallerySearchState').textContent(), /1 matching item/);
    assert.equal(await page.locator('#pageLabel').textContent(), matchingPage);
    assert.equal(await page.locator('.card').getAttribute('data-asset'), matchingAsset);
    await page.locator('#clearGallerySearch').click();
    await page.waitForFunction(count => document.querySelector('#galleryQuery').value === '' && document.querySelector('#gallerySearchState').textContent === '' && document.querySelectorAll('.card').length === Math.min(60, count), gallery.count);
    assert.equal(new URL(page.url()).searchParams.has('q'), false);

    // Name/tag editing remains a separate workspace, reached explicitly.
    await page.getByRole('link', { name: 'People & tags', exact: true }).click();
    await page.waitForURL('**/people');
    assert.equal(await page.locator('#query').inputValue(), '');
    await page.waitForFunction(() => !busy && !paging);
    assert.equal(await page.locator('#uiFaces').isVisible(), true);
    assert.equal(await page.locator('#uiPhotos').isVisible(), true);
    await page.getByRole('button', { name: 'Faces & names', exact: true }).click();
    assert.equal(await page.locator('#uiFaces').isVisible(), true);
    assert.equal(await page.locator('#uiPhotos').isVisible(), false);
    assert.equal(await page.locator('#uiAnalysis').isVisible(), false);
    assert.equal(await page.evaluate(() => document.activeElement.textContent), 'Face review');
    await page.getByRole('button', { name: 'Local analysis', exact: true }).click();
    assert.equal(await page.locator('#analyzeAll').isVisible(), true);
    assert.equal(await page.locator('#uiFaces').isVisible(), false);
    await page.getByRole('button', { name: 'Overview', exact: true }).click();
    assert.equal(await page.locator('#uiPhotos').isVisible(), true);
    assert.equal(await page.locator('#uiFaces').isVisible(), true);
    await page.getByRole('link', { name: 'Hidden Media', exact: true }).click();
    await page.waitForFunction(() => document.querySelector('#visibility').value === 'hidden' && document.querySelector('#libraryTitle').textContent === 'Hidden Media');
    assert.equal(await page.locator('.ui-sidebar a[aria-current="page"]').getAttribute('href'), '/?view=hidden');
    await page.getByRole('link', { name: 'Library', exact: true }).click();
    await page.waitForSelector('#library:not(.hidden)');
    for (const url of ['/', '/people', '/import', '/backups', '/build-notes']) {
      await page.goto('http://127.0.0.1:2283' + url);
      for (const width of [1360, 390]) {
        await page.setViewportSize({ width, height: 900 });
        assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false, 'Overflow: ' + url + ' at ' + width);
        if (process.env.IMMICH_BROWSER_OUTPUT) {
          const label = url === '/' ? 'library' : url.slice(1);
          await page.screenshot({ path: path.join(process.env.IMMICH_BROWSER_OUTPUT, 'redesign-' + label + '-' + width + '.png'), fullPage: true });
        }
      }
      assert.equal(await page.locator('.ui-sidebar').isVisible(), true);
    }
    assert.deepEqual(errors, []);
    console.log('PASS: authenticated main-gallery search/query/count/cards/empty/clear/viewer context, separate People editing task visibility/focus, hidden-view initialization and desktop/narrow shell; no mutation or page errors.');
  } finally {
    await browser.close();
  }
})().catch(error => { console.error(error.message); process.exit(1); });
