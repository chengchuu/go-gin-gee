import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { test } from 'node:test';
import { fileURLToPath } from 'node:url';

const cli = fileURLToPath(new URL('../src/calc-mutuals.ts', import.meta.url));

function fixture(t) {
  const cwd = mkdtempSync(join(tmpdir(), 'calc-mutuals-'));
  t.after(() => rmSync(cwd, { recursive: true, force: true }));
  return cwd;
}

function run(cwd, ...args) {
  const result = spawnSync(process.execPath, [cli, ...args], { cwd, encoding: 'utf8' });
  if (result.error) throw result.error;
  return result;
}

test('preserves counts and lists non-mutual follows in input order', t => {
  const cwd = fixture(t);
  writeFileSync(join(cwd, 'following export.json'), JSON.stringify([
    { userId: 3, username: 'third', displayName: 'Third' },
    { userId: 1, username: 'mutual', displayName: 'Mutual' },
    { userId: 2, username: 'second', displayName: 'Second' },
  ]));
  writeFileSync(join(cwd, 'followers.json'), JSON.stringify([
    { userId: 1, username: 'renamed', displayName: 'Renamed' },
    { userId: 4, username: 'second', displayName: 'Different user' },
  ]));
  const result = run(cwd, '--following', 'following export.json', '--followers', 'followers.json');
  assert.equal(result.status, 0, result.stderr);
  assert.equal(result.stdout, [
    'Number of people I follow: 3',
    'Number of followers: 2',
    'Number of non-mutual follows: 2',
    'Number of mutual follows: 1',
    'Username: third, Displayname: Third',
    'Username: second, Displayname: Second',
    '',
  ].join('\n'));
});

test('requires both export paths', t => {
  const result = run(fixture(t), '--following', 'following.json');
  assert.equal(result.status, 2);
  assert.match(result.stderr, /--following and --followers are required/);
});

test('rejects malformed JSON and invalid records without printing counts', t => {
  const cwd = fixture(t);
  writeFileSync(join(cwd, 'following.json'), '[]');
  for (const [contents, message] of [
    ['[', /Invalid JSON/],
    ['{}', /Expected an array/],
    ['[{"userId":"1","username":"one","displayName":"One"}]', /Invalid user at index 0/],
  ]) {
    writeFileSync(join(cwd, 'followers.json'), contents);
    const result = run(cwd, '--following', 'following.json', '--followers', 'followers.json');
    assert.equal(result.status, 1);
    assert.equal(result.stdout, '');
    assert.match(result.stderr, message);
    assert.match(result.stderr, /followers\.json/);
  }
});
