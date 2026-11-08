import { readFileSync } from 'node:fs';
import { parseArgs } from 'node:util';

const usage = 'Usage: calc-mutuals --following PATH --followers PATH';

type User = { userId: number; username: string; displayName: string };

function isUser(value: unknown): value is User {
  return typeof value === 'object' && value !== null &&
    'userId' in value && typeof value.userId === 'number' && Number.isSafeInteger(value.userId) &&
    'username' in value && typeof value.username === 'string' &&
    'displayName' in value && typeof value.displayName === 'string';
}

function readUsers(path: string): User[] {
  const contents = readFileSync(path, 'utf8');
  let data: unknown;
  try {
    data = JSON.parse(contents);
  } catch (error) {
    if (!(error instanceof SyntaxError)) throw error;
    throw new Error(`Invalid JSON in ${path}.`);
  }
  if (!Array.isArray(data)) throw new Error(`Expected an array of users in ${path}.`);

  return data.map((user: unknown, index) => {
    if (!isUser(user)) throw new Error(`Invalid user at index ${index} in ${path}.`);
    return user;
  });
}

function main() {
  let followingPath: string;
  let followersPath: string;
  try {
    const { values } = parseArgs({ options: {
      following: { type: 'string' },
      followers: { type: 'string' },
      help: { type: 'boolean', short: 'h' },
    } });
    if (values.help) { console.log(usage); return; }
    if (!values.following || !values.followers) throw new Error('--following and --followers are required.');
    followingPath = values.following;
    followersPath = values.followers;
  } catch (error) {
    console.error(`${(error as Error).message}\n${usage}`);
    process.exitCode = 2;
    return;
  }

  try {
    const following = readUsers(followingPath);
    const followers = readUsers(followersPath);
    const followerIds = new Set(followers.map(user => user.userId));
    const nonMutual = following.filter(user => !followerIds.has(user.userId));

    console.log(`Number of people I follow: ${following.length}`);
    console.log(`Number of followers: ${followers.length}`);
    console.log(`Number of non-mutual follows: ${nonMutual.length}`);
    console.log(`Number of mutual follows: ${following.length - nonMutual.length}`);
    for (const user of nonMutual) {
      console.log(`Username: ${user.username}, Displayname: ${user.displayName}`);
    }
  } catch (error) {
    console.error((error as Error).message);
    process.exitCode = 1;
  }
}

main();
