import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import {
  mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync, 
} from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { test } from "node:test";
import { fileURLToPath, pathToFileURL } from "node:url";

const cli = fileURLToPath(new URL("../src/fetch-complete-branches.ts", import.meta.url));

function command(program, args, options = {}) {
  const result = spawnSync(program, args, { encoding: "utf8", timeout: 30000, ...options });
  if (result.error) throw result.error;
  return result;
}

function git(path, ...args) {
  const result = command("git", [ "-C", path, ...args ], {
    env: { ...process.env, GIT_NO_LAZY_FETCH: args[0] === "rev-list" ? "1" : "0" },
  });
  assert.equal(result.status, 0, result.stderr);
  return result.stdout.trim();
}

function missing(path) {
  return git(path, "rev-list", "--objects", "--all", "--no-object-names", "--missing=print")
    .split("\n").filter(line => line.startsWith("?")).map(line => line.slice(1));
}

function fixture(t) {
  const root = mkdtempSync(join(tmpdir(), "fetch-branches-ts-"));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  const source = join(root, "source");
  git(root, "init", "-qb", "main", source);
  git(source, "config", "user.name", "Fixture");
  git(source, "config", "user.email", "fixture@example.com");
  git(source, "config", "commit.gpgsign", "false");
  git(source, "config", "uploadpack.allowFilter", "true");
  for (let index = 0; index < 5; index++) {
    writeFileSync(join(source, "file"), `contents ${index}\n`);
    git(source, "add", "file");
    git(source, "commit", "-qm", `fixture ${index}`);
  }
  git(source, "branch", "feature/nested");
  const target = join(root, "partial clone");
  git(root, "clone", "-q", "--filter=blob:none", "--single-branch", pathToFileURL(source).href, target);
  writeFileSync(join(target, "file"), "local edits\n");
  return { root, source, target };
}

test("keeps completed batches after interruption and resumes without requesting them again", { skip: process.platform === "win32" }, t => {
  const { root, target } = fixture(t);
  const initial = missing(target);
  assert.ok(initial.length >= 4);
  const head = git(target, "rev-parse", "HEAD");
  const before = git(target, "status", "--porcelain");
  const bin = join(root, "bin");
  mkdirSync(bin);
  const realGit = command("which", [ "git" ]).stdout.trim();
  assert.ok(realGit);
  const log = join(root, "batches");
  const helperStopped = join(root, "helper-stopped");
  writeFileSync(join(bin, "git"), `#!/usr/bin/env node
const fs = require('node:fs');
const { spawn, spawnSync } = require('node:child_process');
const args = process.argv.slice(2);
const batch = args.includes('fetch') && args.includes('--stdin');
const input = batch ? fs.readFileSync(0, 'utf8') : undefined;
if (batch) {
  fs.appendFileSync(${JSON.stringify(log)}, JSON.stringify(input.trim().split('\\n')) + '\\n');
  const count = fs.readFileSync(${JSON.stringify(log)}, 'utf8').trim().split('\\n').length;
  if (process.env.FETCH_TEST_INTERRUPT && count === 1) {
    console.error('connection reset by peer');
    process.exit(128);
  }
  if (process.env.FETCH_TEST_INTERRUPT && count === 3) {
    const helper = spawn(process.execPath, ['-e', ${JSON.stringify(`
      process.on('SIGTERM', () => {
        require('node:fs').writeFileSync(${JSON.stringify(helperStopped)}, 'stopped');
        process.exit(0);
      });
      setTimeout(() => process.exit(1), 3000);
      process.send('ready');
    `)}], { stdio: ['ignore', 'inherit', 'inherit', 'ipc'] });
    helper.once('message', () => process.kill(process.ppid, 'SIGINT'));
    return;
  }
}
const result = spawnSync(${JSON.stringify(realGit)}, args, {
  input, stdio: [batch ? 'pipe' : 'inherit', 'inherit', 'inherit'],
});
process.exit(result.status ?? 1);
`, { mode: 0o755 });
  const env = { ...process.env, PATH: `${bin}:${process.env.PATH}` };
  const args = [ cli, "--path", target, "--batch-size", "2", "--retries", "1" ];
  const interrupted = command(process.execPath, args, { env: { ...env, FETCH_TEST_INTERRUPT: "1" } });
  assert.equal(interrupted.status, 130, interrupted.stderr);
  assert.match(interrupted.stderr, /Retry 1\/1/);
  assert.match(interrupted.stderr, /Rerun the same command/);
  assert.equal(readFileSync(helperStopped, "utf8"), "stopped");
  assert.equal(git(target, "config", "--get", "remote.origin.partialclonefilter"), "blob:none");
  assert.equal(git(target, "config", "--get", "remote.origin.promisor"), "true");
  const remaining = new Set(missing(target));
  assert.equal(remaining.size, initial.length - 2);
  const resumed = command(process.execPath, args, { env });
  assert.equal(resumed.status, 0, resumed.stderr);
  const batches = readFileSync(log, "utf8").trim().split("\n").map(line => JSON.parse(line));
  assert.deepEqual(batches[0], batches[1]);
  assert.ok(batches.every(batch => batch.length <= 2));
  assert.deepEqual(new Set(batches.slice(3).flat()), remaining);
  assert.deepEqual(missing(target), []);
  assert.equal(command(realGit, [ "-C", target, "config", "--get", "remote.origin.promisor" ]).status, 1);
  assert.equal(command(realGit, [ "-C", target, "config", "--get", "remote.origin.partialclonefilter" ]).status, 1);
  assert.equal(git(target, "rev-parse", "HEAD"), head);
  assert.equal(git(target, "status", "--porcelain"), before);
  assert.equal(readFileSync(join(target, "file"), "utf8"), "local edits\n");
  git(target, "show-ref", "--verify", "refs/remotes/origin/feature/nested");
});

test("retries a branch refresh after the first attempt already unshallowed the clone", { skip: process.platform === "win32" }, t => {
  const { root, source } = fixture(t);
  const target = join(root, "shallow clone");
  git(root, "clone", "-q", "--depth=1", pathToFileURL(source).href, target);
  const bin = join(root, "bin");
  mkdirSync(bin);
  writeFileSync(join(bin, "git"), `#!/bin/sh
for arg do
  if [ "$arg" = --unshallow ]; then
    "$FETCH_TEST_GIT" "$@" || exit $?
    echo 'connection reset by peer' >&2
    exit 128
  fi
done
exec "$FETCH_TEST_GIT" "$@"
`, { mode: 0o755 });
  const result = command(process.execPath, [ cli, "--path", target, "--retries", "1" ], {
    env: { ...process.env, PATH: `${bin}:${process.env.PATH}`, FETCH_TEST_GIT: command("which", [ "git" ]).stdout.trim() },
  });
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stderr, /Retry 1\/1/);
  assert.equal(git(target, "rev-parse", "--is-shallow-repository"), "false");
  assert.equal(git(target, "rev-list", "--count", "origin/main"), "5");
});

test("an unavailable origin fails and keeps the partial-clone configuration", t => {
  const { root, target } = fixture(t);
  git(target, "remote", "set-url", "origin", join(root, "unavailable"));
  const result = command(process.execPath, [ cli, "--path", target ]);
  assert.equal(result.status, 1);
  assert.doesNotMatch(result.stderr, /Retry/);
  assert.equal(git(target, "config", "--get", "remote.origin.promisor"), "true");
  assert.equal(git(target, "config", "--get", "remote.origin.partialclonefilter"), "blob:none");
});

test("rejects invalid batch sizes before opening a repository", () => {
  const result = command(process.execPath, [ cli, "--path", "unused", "--batch-size", "0" ]);
  assert.equal(result.status, 2);
  assert.match(result.stderr, /--batch-size must be an integer/);
});
