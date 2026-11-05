import fs from 'node:fs';
import path from 'node:path';
import { git } from './git.mjs';

const GIT_TIMEOUT = 3000;

export function statePath(root, sessionId, { timeout = GIT_TIMEOUT } = {}) {
  const id = String(sessionId).replace(/[^A-Za-z0-9_-]/g, '_');
  const r = git(['rev-parse', '--git-path', `claude-hadesgate/${id}.json`], { cwd: root, timeout });
  if (r.status !== 0) throw new Error(`git rev-parse --git-path falhou em ${root}: ${r.stderr.trim()}`);
  return path.resolve(root, r.stdout.replace(/\r?\n$/, ''));
}

export function readState(root, sessionId) {
  try {
    const parsed = JSON.parse(fs.readFileSync(statePath(root, sessionId), 'utf8'));
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? parsed : {};
  } catch {
    return {};
  }
}

function writeFileAtomic(file, state) {
  fs.mkdirSync(path.dirname(file), { recursive: true });
  const tmp = `${file}.${process.pid}.tmp`;
  fs.writeFileSync(tmp, JSON.stringify(state));
  fs.renameSync(tmp, file);
}

export function writeState(root, sessionId, state) {
  writeFileAtomic(statePath(root, sessionId), state);
}

export function recordSessionBase(root, sessionId, { timeout = GIT_TIMEOUT } = {}) {
  const file = statePath(root, sessionId, { timeout });
  if (fs.existsSync(file)) return false;
  const head = git(['rev-parse', '--verify', '--quiet', 'HEAD'], { cwd: root, timeout });
  if (head.status !== 0) return false;
  writeFileAtomic(file, { baseSha: head.stdout.trim() });
  return true;
}

