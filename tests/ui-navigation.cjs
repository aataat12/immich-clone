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
    await page.getByRole('searchbox', { name: 'Search names, filenames or tags', exact: true }).fill('UI-search-no-match');
    await page.locator('.ui-global-search button').click();
    await page.waitForURL('**/people?q=UI-search-no-match');
    assert.equal(await page.locator('#query').inputValue(), 'UI-search-no-match');
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
    console.log('PASS: real authenticated navigation/search, People task visibility/focus, hidden-view initialization, desktop/narrow shell; no mutation or page errors.');
  } finally {
    await browser.close();
  }
})().catch(error => { console.error(error.message); process.exit(1); });
