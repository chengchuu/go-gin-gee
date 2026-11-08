import { spawn } from 'node:child_process';
import { createInterface } from 'node:readline';
import { setTimeout as delay } from 'node:timers/promises';
import { parseArgs } from 'node:util';

const usage = 'Usage: fetch-complete-branches --path PATH [--batch-size 500] [--retries 3]';
const refspec = '+refs/heads/*:refs/remotes/origin/*';
const fetchFlags = ['--no-filter', '--no-tags', '--no-prune', '--no-prune-tags',
  '--no-recurse-submodules', '--no-auto-maintenance', '--refmap='];
const transient = /connection (?:reset|timed out|closed)|could not resolve host|operation timed out|early eof|unexpected disconnect|(?:http |returned error: )(?:408|429|5\d\d)\b/i;

class GitError extends Error {
  readonly status: number | null;
  readonly stderr: string;

  constructor(args: string[], status: number | null, stderr: string) {
    super(`git ${args[0]} failed (${status ?? 'terminated'}).${stderr.trim() ? `\n${stderr.trim()}` : ''}`);
    this.status = status;
    this.stderr = stderr;
  }
}

function integer(name: string, value: string, min: number, max: number): number {
  const number = Number(value);
  if (!/^\d+$/.test(value) || !Number.isSafeInteger(number) || number < min || number > max) {
    throw new Error(`--${name} must be an integer between ${min} and ${max}.`);
  }
  return number;
}

async function run(path: string, batchSize: number, retries: number, signal: AbortSignal) {
  async function git(args: string[], options: {
    input?: string;
    line?: (line: string) => void;
    showProgress?: boolean;
  } = {}): Promise<string> {
    signal.throwIfAborted();
    const group = process.platform !== 'win32';
    const child = spawn('git', ['-C', path, ...args], {
      env: { ...process.env, GIT_NO_LAZY_FETCH: '1', LC_ALL: 'C' },
      stdio: ['pipe', 'pipe', 'pipe'],
      detached: group,
    });
    return new Promise((resolve, reject) => {
      let stdout = '';
      let stderr = '';
      let processError: Error | undefined;
      let inputError: Error | undefined;
      const cancel = () => {
        if (!child.pid) return;
        try {
          // Git's transport helpers can hold its output pipes open after Git exits.
          if (group) process.kill(-child.pid, 'SIGTERM');
          else child.kill();
        } catch (error) {
          if ((error as NodeJS.ErrnoException).code !== 'ESRCH') processError = error as Error;
        }
      };
      signal.addEventListener('abort', cancel, { once: true });
      child.on('error', error => { processError = error; });
      child.stdin.on('error', error => { inputError = error; });
      if (options.line) {
        createInterface({ input: child.stdout, crlfDelay: Infinity }).on('line', options.line);
      } else {
        child.stdout.setEncoding('utf8').on('data', chunk => { stdout += chunk; });
      }
      child.stderr.setEncoding('utf8').on('data', (chunk: string) => {
        stderr = (stderr + chunk).slice(-8192);
        if (options.showProgress) process.stderr.write(chunk);
      });
      child.on('close', status => {
        signal.removeEventListener('abort', cancel);
        if (signal.aborted) reject(signal.reason);
        else if (processError) reject(processError);
        else if (status !== 0) reject(new GitError(args, status, stderr));
        else if (inputError) reject(inputError);
        else resolve(stdout.trim());
      });
      child.stdin.end(options.input);
    });
  }

  async function fetch(args: string[] | (() => Promise<string[]>), input?: string) {
    for (let attempt = 0; ; attempt++) {
      const fetchArgs = typeof args === 'function' ? await args() : args;
      try {
        await git(['fetch', '--progress', ...fetchFlags, ...fetchArgs], { input, showProgress: true });
        return;
      } catch (error) {
        if (signal.aborted || !(error instanceof GitError) || !transient.test(error.stderr) || attempt >= retries) throw error;
        const seconds = Math.min(2 ** attempt, 30);
        console.error(`Retry ${attempt + 1}/${retries} in ${seconds}s.`);
        await delay(seconds * 1000, undefined, { signal });
      }
    }
  }

  async function missingObjects() {
    const missing = new Set<string>();
    await git(['rev-list', '--objects', '--all', '--no-object-names', '--missing=print'], {
      line(line) {
        if (/^\?(?:[a-f0-9]{40}|[a-f0-9]{64})$/.test(line)) missing.add(line.slice(1));
      },
    });
    return missing;
  }

  await git(['rev-parse', '--git-dir']);
  await git(['remote', 'get-url', 'origin']);
  console.log('Refreshing origin branch refs...');
  await fetch(async () => {
    const shallow = await git(['rev-parse', '--is-shallow-repository']);
    return [...(shallow === 'true' ? ['--unshallow'] : []), 'origin', refspec];
  });
  if (await git(['rev-parse', '--is-shallow-repository']) === 'true') {
    throw new Error('Origin did not provide complete commit history.');
  }

  let missing = await missingObjects();
  while (missing.size > 0) {
    const ids = [...missing].sort();
    const count = Math.ceil(ids.length / batchSize);
    console.log(`${ids.length} missing objects across ${count} batches.`);
    for (let offset = 0; offset < ids.length; offset += batchSize) {
      const batch = ids.slice(offset, offset + batchSize);
      console.log(`Fetching batch ${offset / batchSize + 1}/${count} (${batch.length} objects)...`);
      await fetch(['--no-write-fetch-head', 'origin', '--stdin'], batch.join('\n') + '\n');
    }
    const next = await missingObjects();
    if (ids.every(id => next.has(id))) throw new Error('Object download made no progress.');
    missing = next;
  }

  // Keep the promisor configuration until the entire local ref graph is complete.
  await git(['config', '--local', '--replace-all', 'remote.origin.fetch', refspec]);
  for (const key of ['remote.origin.partialclonefilter', 'remote.origin.promisor']) {
    try {
      await git(['config', '--local', '--unset-all', key]);
    } catch (error) {
      if (!(error instanceof GitError) || error.status !== 5) throw error;
    }
  }
  console.log('Complete: all origin branches fetched and no reachable Git objects are missing.');
}

async function main() {
  let options: { path: string; batchSize: number; retries: number };
  try {
    const { values } = parseArgs({ options: {
      path: { type: 'string' },
      'batch-size': { type: 'string', default: '500' },
      retries: { type: 'string', default: '3' },
      help: { type: 'boolean', short: 'h' },
    } });
    if (values.help) { console.log(usage); return; }
    if (!values.path) throw new Error('--path is required.');
    options = { path: values.path, batchSize: integer('batch-size', values['batch-size'], 1, 10000),
      retries: integer('retries', values.retries, 0, 10) };
  } catch (error) {
    console.error(`${(error as Error).message}\n${usage}`);
    process.exitCode = 2;
    return;
  }

  const stop = new AbortController();
  const interrupt = () => stop.abort(130);
  const terminate = () => stop.abort(143);
  process.on('SIGINT', interrupt);
  process.on('SIGTERM', terminate);
  try {
    await run(options.path, options.batchSize, options.retries, stop.signal);
  } catch (error) {
    if (stop.signal.aborted) {
      console.error('Interrupted. Rerun the same command to resume.');
      process.exitCode = stop.signal.reason;
    } else {
      console.error((error as Error).message);
      process.exitCode = 1;
    }
  } finally {
    process.off('SIGINT', interrupt);
    process.off('SIGTERM', terminate);
  }
}

await main();
