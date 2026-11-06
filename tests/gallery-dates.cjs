// Exercise the actual bundled date helpers without an installed service/browser.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const html = fs.readFileSync(path.join(__dirname, '../internal/web/index.html'), 'utf8');
const helperNames = ['galleryCaptureDate', 'galleryDate', 'galleryDateDetails'];
const helpers = helperNames.map(name => {
  const line = html.split(/\r?\n/).find(line => line.startsWith('function ' + name + '('));
  assert.ok(line, 'bundled helper missing: ' + name);
  return line;
}).join('\n');
const context = vm.createContext({});
vm.runInContext(helpers, context);
const added = '2026-11-05T23:59:58Z';
const captured = '2024-03-02T10:11:12';
const photo = {added, photo: {captured}};
assert.equal(context.galleryDate(photo), 'Date taken: 2024-03-02');
assert.equal(context.galleryDate({added}), 'Date imported: 2026-11-05');
for (const invalid of ['', '2024-02-30T10:11:12', 'not a date', '2024-03-02T10:11:12+05:30', '2024-03-02T10:11:12Z']) {
  assert.equal(context.galleryDate({added, photo: {captured: invalid}}), 'Date imported: 2026-11-05');
}
const details = Array.from(context.galleryDateDetails(photo));
assert.ok(details.includes('Date taken: ' + captured));
assert.ok(details.includes('Capture time zone: Not read from original metadata'));
assert.ok(details.includes('Date imported: ' + added));
assert.ok(details.includes('Import time zone: UTC'));
assert.ok(!details.some(line => /not supplied|zone unknown|camera time/.test(line)));
assert.ok(Array.from(context.galleryDateDetails({added})).includes('Date taken: Not available'));
// A legacy catalog contains no extraction status; it must not imply no offset.
assert.ok(Array.from(context.galleryDateDetails(photo)).some(line => line.includes('Not read')));
console.log('PASS: taken/imported labels, malformed-date fallback, no capture-zone invention, honest unread-zone details, UTC import details.');
