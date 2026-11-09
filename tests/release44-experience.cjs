// Run against a disposable LocalService instance, with an actual restart between stages.
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const fs = require('node:fs');
const path = require('node:path');
const { chromium } = require(process.env.IMMICH_BROWSER_MODULE);

const origin = 'http://127.0.0.1:2283';
const stage = process.env.IMMICH_EXPERIENCE_STAGE;
const fixturePath = process.env.IMMICH_EXPERIENCE_FIXTURE;
const output = process.env.IMMICH_BROWSER_OUTPUT;
const marker = 'Combined44';
const batchTag = 'Together44';
const resumedTag = 'Recovered44';
const title = 'Deux sessions <ensemble>';
const notes = 'Retained second-client note';
const count = 63;

async function api(page, endpoint, data) {
  const response = data === undefined
    ? await page.request.get(origin + endpoint)
    : await page.request.post(origin + endpoint, { headers: { 'X-Immich-Request': '1' }, data });
  assert(response.ok(), endpoint + ' returned ' + response.status());
  return response.json();
}

async function openSession(browser, timezoneId, errors) {
  const context = await browser.newContext({ timezoneId, viewport: { width: 1280, height: 1000 } });
  const page = await context.newPage();
  page.setDefaultTimeout(30000);
  page.on('pageerror', error => errors.push(error.message));
  await page.goto(origin);
  await page.fill('#code', fs.readFileSync(path.join(process.env.IMMICH_TEST_STATE, 'access-code.txt'), 'utf8').trim());
  await page.click('#loginForm button');
  await page.waitForSelector('#library:not(.hidden)');
  return page;
}

async function waitGallery(page, items, number = 1) {
  await page.waitForFunction(({items, number}) => {
    const shown = Math.min(60, items - (number - 1) * 60);
    return !document.querySelector('#status').textContent.includes('Loading')
      && document.querySelector('#pageLabel').textContent === `Page ${number} · ${shown} shown / ${items}`
      && document.querySelectorAll('#grid .card').length === shown;
  }, {items, number});
}

function card(page, id) {
  return page.locator('#grid [data-asset="' + id + '"]');
}

function sortIDs(ids) {
  return [...ids].sort();
}

async function seed(page) {
  const png = Buffer.from(await page.evaluate(() => {
    const canvas = document.createElement('canvas');
    canvas.width = 200;
    canvas.height = 150;
    canvas.getContext('2d').fillRect(0, 0, 200, 150);
    return canvas.toDataURL('image/png').split(',')[1];
  }), 'base64');
  const bytes = Array.from({ length: 65 }, (_, i) => Buffer.concat([png, Buffer.from(marker + i)]));
  const files = bytes.map((data, i) => {
    const name = marker + '-' + String(i).padStart(2, '0') + '.png';
    return { name, relative: (i === 64 ? 'Other44' : 'Trip44') + '/Day1/' + name, size: data.length, modified: 1 };
  });
  const batch = await api(page, '/api/imports', { files });
  await api(page, '/api/imports/' + batch.id + '/start', {});
  const ids = [];
  for (const [i, data] of bytes.entries()) {
    const result = await api(page, '/api/imports/' + batch.id + '/files/' + i, data);
    assert.equal(result.asset_id, crypto.createHash('sha256').update(data).digest('hex'));
    ids.push(result.asset_id);
  }
  await api(page, '/api/imports/' + batch.id + '/finish', {});
  await api(page, '/api/assets/' + ids[63] + '/manual-face', { box: [.1, .1, .3, .3], name: 'Excluded44' });
  await page.waitForFunction(async marker => {
    const response = await fetch('/api/gallery?sort=name&visibility=library&limit=100&q=' + marker);
    if (!response.ok) return false;
    const results = await response.json();
    return results.count === 65 && results.assets.every(a => a.preview === 'ready');
  }, marker);
  return { ids, batch: batch.id };
}

async function selectAndReview(page, ids, action, { album, tag } = {}) {
  await waitGallery(page, ids.length);
  await page.click('#selectOrganizationPage');
  await page.click('#nextPage');
  await waitGallery(page, ids.length, 2);
  await page.click('#selectOrganizationPage');
  assert.match(await page.locator('#organizationSelected').textContent(), new RegExp('^' + ids.length + ' selected'));
  await page.selectOption('#batchAction', action);
  if (album) await page.selectOption('#batchAlbum', album);
  if (tag) await page.fill('#batchTag', tag);
  await page.click('#reviewOrganization');
  await page.waitForSelector('#organizationPreview[open]');
  assert.match(await page.locator('#organizationPreviewTitle').textContent(), new RegExp(ids.length + ' exact targets'));
  const receipt = await page.evaluate(() => organizationBatch);
  assert.deepEqual(sortIDs(receipt.targets.map(target => target.id)), sortIDs(ids));
  assert(receipt.targets.every(target => target.outcome === 'pending'));
  return receipt;
}

async function finishReceipt(page, ids) {
  await page.click('#continueOrganization');
  await page.waitForFunction(() => !organizationRunning && organizationBatch?.status === 'done');
  const receipt = await page.evaluate(() => organizationBatch);
  assert.deepEqual(sortIDs(receipt.targets.map(target => target.id)), sortIDs(ids));
  assert(receipt.targets.every(target => target.outcome === 'saved'));
  await page.click('#closeOrganization');
  await waitGallery(page, ids.length);
  assert.match(await page.locator('#organizationSelected').textContent(), /^0 selected/);
}

async function chooseCombinedQuery(page, batch, album, ids) {
  // Reorder the album first, then retain its manual order in the more selective query.
  await page.goto(origin + '/?album=' + album + '&sort=manual');
  await waitGallery(page, 65);
  const before = await api(page, '/api/metadata/albums/' + album + '?limit=100');
  const moved = before.members[0].asset_id;
  await card(page, moved).locator('.moveAlbumnext').click();
  await page.waitForFunction(moved => document.querySelectorAll('#grid .card')[1]?.dataset.asset === moved
    && !document.querySelector('#status').textContent.includes('Loading'), moved);
  await card(page, moved).locator('.albumCoverAction').click();
  await page.waitForFunction(moved => currentAlbum?.cover === moved && !document.querySelector('#status').textContent.includes('Loading'), moved);
  const ordered = await api(page, '/api/metadata/albums/' + album + '?limit=100');
  assert.equal(ordered.members[1].asset_id, moved);
  assert.equal(ordered.album.cover, moved);

  await page.fill('#galleryQuery', marker);
  await page.locator('#gallerySearch button[type=submit]').click();
  await waitGallery(page, 65);
  await page.selectOption('#favoriteScope', 'true');
  await waitGallery(page, 65);
  await page.selectOption('#mediaFilter', 'photo');
  await waitGallery(page, 65);
  await page.locator('#combinedFilters > summary').click();
  await page.fill('#person_exclude', 'Excluded44');
  await page.fill('#tag_all', batchTag);
  await page.locator('#filterForm button[type=submit]').click();
  await waitGallery(page, 64);
  await page.click('#browseSources');
  await page.locator('#sourceFolders button').filter({ hasText: 'Trip44 ·' }).click();
  await waitGallery(page, count);
  await page.click('#browseSources');
  await page.locator('#sourceBatches button').filter({ hasText: batch }).click();
  await waitGallery(page, count);

  await page.click('#browseCalendar');
  await page.selectOption('#calendarBasis', 'upload');
  await page.waitForFunction(() => calendarResult?.basis === 'upload');
  const calendar = await page.evaluate(() => calendarResult);
  assert.equal(calendar.count, count);
  await page.selectOption('#calendarYear', String(calendar.years[0].year));
  await page.waitForFunction(() => calendarResult?.months.length === 1);
  await page.selectOption('#calendarMonth', await page.evaluate(() => String(calendarResult.months[0].month)));
  await page.waitForFunction(() => calendarResult?.days.length === 1);
  await page.selectOption('#calendarDay', await page.evaluate(() => String(calendarResult.days[0].day)));
  await page.click('#calendarGo');
  await waitGallery(page, count);

  const address = new URL(page.url());
  assert.equal(address.searchParams.get('album'), album);
  assert.equal(address.searchParams.get('sort'), 'manual');
  assert.equal(address.searchParams.get('q'), marker);
  assert.equal(address.searchParams.get('favorite'), 'true');
  assert.equal(address.searchParams.get('media'), 'photo');
  assert.equal(address.searchParams.get('person_exclude'), 'Excluded44');
  assert.equal(address.searchParams.get('tag_all'), batchTag);
  assert.equal(address.searchParams.get('source_folder'), 'Trip44');
  assert.equal(address.searchParams.get('import_batch'), batch);
  assert.equal(address.searchParams.get('date_basis'), 'upload');
  assert(address.searchParams.has('from') && address.searchParams.has('until'));

  const expected = ordered.members.map(member => member.asset_id).filter(id => ids.includes(id));
  const gallery = await api(page, '/api/gallery?visibility=library&limit=100&' + address.searchParams);
  assert.equal(gallery.count, count);
  assert.deepEqual(gallery.assets.map(asset => asset.id), expected);
  return { query: address.search, expected, moved };
}

async function waitViewer(page, id) {
  await page.waitForFunction(id => galleryContext.viewer?.id === id && galleryContext.viewer.neighbors
    && !galleryContext.viewer.loading && document.querySelector('#viewerFavorite').getAttribute('aria-pressed') === 'true', id);
}

async function concurrentCaption(page, other, query, ids) {
  const boundary = ids[59];
  const target = ids[60];
  await page.check('#selectionMode');
  await card(page, boundary).locator('.organizationPick').check();
  await card(page, boundary).locator('.openMedia').click();
  await waitViewer(page, boundary);
  await page.keyboard.press('ArrowRight');
  await waitViewer(page, target);
  await page.click('#closeViewer');
  await waitGallery(page, count, 2);
  await page.waitForFunction(id => document.activeElement.closest('.card')?.dataset.asset === id && !galleryContext.returning, target);
  assert.match(await page.locator('#organizationSelected').textContent(), /^1 selected/);

  await other.goto(origin + '/' + query);
  await waitGallery(other, count);
  await other.click('#nextPage');
  await waitGallery(other, count, 2);
  for (const session of [page, other]) {
    await card(session, target).locator('.openMedia').click();
    await waitViewer(session, target);
    await session.click('#viewerCaption');
    await session.waitForFunction(() => !document.querySelector('#saveCaption').disabled);
  }
  await page.fill('#captionNotes', notes);
  await other.fill('#captionTitle', title);
  await other.fill('#captionText', 'Résumé de voyage');
  await other.click('#saveCaption');
  await other.waitForSelector('#captionEditor', { state: 'hidden' });
  await page.click('#saveCaption');
  await page.waitForFunction(() => document.querySelector('#captionError').textContent.includes('Your draft is kept'));
  assert.equal(await page.inputValue('#captionNotes'), notes);
  await page.click('#reloadCaption');
  await page.waitForFunction(title => document.querySelector('#captionTitle').value === title
    && !document.querySelector('#saveCaption').disabled, title);
  await page.fill('#captionNotes', notes);
  await page.click('#saveCaption');
  await page.waitForSelector('#captionEditor', { state: 'hidden' });
  await page.click('#viewerRefresh');
  await page.waitForFunction(notes => document.querySelector('#viewerText').textContent.includes(notes)
    && !galleryContext.viewer.stale && !galleryContext.viewer.loading, notes);
  assert.match(await page.locator('#viewerText').textContent(), /Résumé de voyage/);
  assert.equal(await page.locator('#viewerFavorite').getAttribute('aria-pressed'), 'true');
  if (output) await page.screenshot({ path: path.join(output, 'combined44-viewer.png') });
  await page.click('#closeViewer');
  await page.waitForFunction(() => !galleryContext.returning);
  assert.match(await page.locator('#organizationSelected').textContent(), /^0 selected/);
  await page.goto(origin + '/' + query);
  await waitGallery(page, count);
  return target;
}

async function prepare(page, other, timezoneId) {
  const { ids, batch } = await seed(page);
  await page.click('#newAlbum');
  await page.fill('#albumTitleInput', 'Together in 0.1.44');
  await page.click('#saveAlbum');
  await page.waitForFunction(() => currentAlbum?.title === 'Together in 0.1.44');
  const album = await page.evaluate(() => currentAlbum.id);
  await page.goto(origin + '/?sort=name&q=' + marker);
  for (const [action, inputs] of [['favorite_on', {}], ['add_tag', { tag: batchTag }], ['add_album', { album }]]) {
    await selectAndReview(page, ids, action, inputs);
    await finishReceipt(page, ids);
  }
  const { query, expected, moved } = await chooseCombinedQuery(page, batch, album, ids.slice(0, count));
  const target = await concurrentCaption(page, other, query, expected);
  const interrupted = await selectAndReview(page, expected, 'add_tag', { tag: resumedTag });
  await page.route('**/api/organization/batches/' + interrupted.id, async route => {
    const response = await route.fetch();
    assert(response.ok());
    await route.abort('failed');
  }, { times: 1 });
  await page.click('#continueOrganization');
  await page.waitForFunction(() => !organizationRunning
    && document.querySelector('#organizationProgress').textContent.includes('last response may have been lost'));
  const receipt = await api(other, '/api/organization/batches/' + interrupted.id);
  assert.equal(receipt.status, 'paused');
  assert.equal(receipt.targets.filter(target => target.outcome === 'saved').length, 25);
  assert.equal(receipt.targets.filter(target => target.outcome === 'pending').length, count - 25);
  fs.writeFileSync(fixturePath, JSON.stringify({ timezoneId, ids, batch, album, query, expected, moved, target, receipt }));
  console.log('PASS: integrated browser workflow, two isolated sessions, and lost HTTP response after 25 committed changes');
}

async function checkRestoredGraph(page, fixture) {
  await page.goto(origin + '/' + fixture.query);
  await waitGallery(page, count);
  assert.equal(new URL(page.url()).search, fixture.query);
  const first = await page.locator('#grid .card').evaluateAll(cards => cards.map(card => card.dataset.asset));
  await page.click('#nextPage');
  await waitGallery(page, count, 2);
  const second = await page.locator('#grid .card').evaluateAll(cards => cards.map(card => card.dataset.asset));
  assert.deepEqual([...first, ...second], fixture.expected);

  const query = new URLSearchParams(fixture.query);
  query.set('visibility', 'library');
  query.append('tag_all', resumedTag);
  const resumed = await api(page, '/api/gallery?limit=100&' + query);
  assert.deepEqual(resumed.assets.map(asset => asset.id), fixture.expected);
  const exactText = new URLSearchParams(query);
  exactText.set('q', notes);
  assert.deepEqual((await api(page, '/api/gallery?' + exactText)).assets.map(asset => asset.id), [fixture.target]);
  const metadata = await api(page, '/api/metadata/asset/' + fixture.target);
  assert.equal(metadata.title, title);
  assert.equal(metadata.caption, 'Résumé de voyage');
  assert.equal(metadata.notes, notes);
  assert.equal(metadata.favorite, true);
  const album = await api(page, '/api/metadata/albums/' + fixture.album + '?limit=100');
  assert.equal(album.album.cover, fixture.moved);
  assert.equal(album.members[1].asset_id, fixture.moved);
  assert.deepEqual(sortIDs(album.members.map(member => member.asset_id)), sortIDs(fixture.ids));
  const receipt = await api(page, '/api/organization/batches/' + fixture.receipt.id);
  assert.equal(receipt.status, 'done');
  assert.deepEqual(sortIDs(receipt.targets.filter(target => target.outcome === 'saved').map(target => target.id)),
    sortIDs(fixture.expected));
  const exported = await api(page, '/api/export');
  assert.equal(exported.import_history.sessions[fixture.batch].session.accepted_files, 65);
  for (const id of fixture.ids) {
    const response = await page.request.get(origin + '/api/assets/' + id + '/original');
    assert(response.ok());
    assert.equal(crypto.createHash('sha256').update(await response.body()).digest('hex'), id);
  }
  await card(page, fixture.target).locator('.openMedia').click();
  await waitViewer(page, fixture.target);
  assert.match(await page.locator('#viewerText').textContent(), new RegExp(notes));
  await page.click('#closeViewer');
  await page.waitForFunction(() => !galleryContext.returning);
}

async function recover(page, fixture) {
  await page.goto(origin + '/' + fixture.query);
  await waitGallery(page, count);
  assert.match(await page.locator('#organizationSelected').textContent(), /^0 selected/);
  const paused = await api(page, '/api/organization/batches/' + fixture.receipt.id);
  assert.deepEqual(paused.targets, fixture.receipt.targets);
  assert.equal(paused.status, 'paused');
  await page.click('#operationHistory');
  await page.locator('#operationList button').filter({ hasText: 'Add custom tag · paused' }).click();
  await page.waitForSelector('#organizationPreview[open]');
  assert.match(await page.locator('#organizationProgress').textContent(), /25 saved/);
  await finishReceipt(page, fixture.expected);
  await checkRestoredGraph(page, fixture);
  console.log('PASS: restart retains the paused receipt, explicit resume completes it, and combined results and 65 originals remain exact');
}

async function main() {
  assert(fixturePath && ['prepare', 'recover', 'verify'].includes(stage), 'Set the experience fixture path and stage');
  const fixture = stage === 'prepare' ? null : JSON.parse(fs.readFileSync(fixturePath, 'utf8'));
  // Place this short fixture away from midnight in the browser calendar.
  const hour = new Date().getUTCHours();
  const timezoneId = fixture?.timezoneId || (hour < 4 || hour >= 20 ? 'America/Chicago' : 'UTC');
  const browser = await chromium.launch({ channel: 'msedge', headless: true });
  try {
    const errors = [];
    const page = await openSession(browser, timezoneId, errors);
    if (stage === 'prepare') {
      const other = await openSession(browser, timezoneId, errors);
      const session = async p => (await p.context().cookies(origin)).find(cookie => cookie.name === 'session')?.value;
      const firstSession = await session(page);
      const secondSession = await session(other);
      assert(firstSession && secondSession && firstSession !== secondSession, 'Each browser context must have a distinct session');
      await prepare(page, other, timezoneId);
    } else if (stage === 'recover') {
      await recover(page, fixture);
    } else {
      await checkRestoredGraph(page, fixture);
      console.log('PASS: independent restored catalog opens the same browser query, metadata, receipt and 65 original hashes');
    }
    assert.deepEqual(errors, []);
  } finally {
    await browser.close();
  }
}

main().catch(error => { console.error(error.stack); process.exitCode = 1; });
