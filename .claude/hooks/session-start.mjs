import fs from 'node:fs';
import path from 'node:path';
import { git, repoRoot } from './lib/git.mjs';
import { emitSessionContext, isMain, readInput } from './lib/io.mjs';
import { recordSessionBase } from './lib/state.mjs';

export const CANARY = '[harness hadesgate] ativo.';

const GIT_TIMEOUT = 3000;
const BUDGET_MS = 6000;

function gitSlot(deadline) {
  const left = Math.min(GIT_TIMEOUT, deadline - Date.now());
  return left > 0 ? left : null;
}

function formatAge(ms) {
  const minutes = Math.max(0, Math.floor(ms / 60000));
  if (minutes < 60) return `há ${minutes} min`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `há ${hours} h`;
  const days = Math.floor(hours / 24);
  return days === 1 ? 'há 1 dia' : `há ${days} dias`;
}

function lastFetch(root, now, deadline) {
  const timeout = gitSlot(deadline);
  if (timeout === null) return 'sem registro de fetch';
  const fetchHead = git(['rev-parse', '--git-path', 'FETCH_HEAD'], { cwd: root, timeout });
  if (fetchHead.status !== 0) return 'sem registro de fetch';
  try {
    const { mtimeMs } = fs.statSync(path.resolve(root, fetchHead.stdout.trim()));
    return `último fetch ${formatAge(now - mtimeMs)}`;
  } catch {
    return 'sem registro de fetch';
  }
}

export function buildSessionContext({ cwd, now = Date.now(), deadline = Date.now() + BUDGET_MS }) {
  const rootTimeout = gitSlot(deadline);
  if (rootTimeout === null) return CANARY;
  const root = repoRoot(cwd, { timeout: rootTimeout });
  if (root === null) return CANARY;
  const countTimeout = gitSlot(deadline);
  if (countTimeout === null) return CANARY;
  const behind = git(['rev-list', '--count', 'HEAD..refs/remotes/origin/main'], { cwd: root, timeout: countTimeout });
  const count = behind.status === 0 ? Number.parseInt(behind.stdout.trim(), 10) : 0;
  if (!(count > 0)) return CANARY;
  const commits = count === 1 ? '1 commit' : `${count} commits`;
  return `${CANARY} origin/main local tem ${commits} que o HEAD não tem; ${lastFetch(root, now, deadline)}. Antes de planejar, rode git fetch e traga a main.`;
}

function recordBase(input, cwd, deadline) {
  try {
    if (!input.session_id) return;
    const rootTimeout = gitSlot(deadline);
    if (rootTimeout === null) return;
    const root = repoRoot(cwd, { timeout: rootTimeout });
    const left = deadline - Date.now();
    if (root && left > 0) {
      recordSessionBase(root, input.session_id, { timeout: Math.max(1, Math.min(GIT_TIMEOUT, Math.floor(left / 2))) });
    }
  } catch {
    return;
  }
}

function main() {
  let text = CANARY;
  try {
    const input = readInput();
    const cwd = input.cwd ?? process.env.CLAUDE_PROJECT_DIR ?? process.cwd();
    const deadline = Date.now() + BUDGET_MS;
    text = buildSessionContext({ cwd, deadline });
    recordBase(input, cwd, deadline);
  } catch {
    text = CANARY;
  }
  emitSessionContext(text);
}

if (isMain(import.meta.url)) main();

