import { spawnSync } from 'node:child_process';

export const GIT_ENV = { GIT_OPTIONAL_LOCKS: '0', GIT_TERMINAL_PROMPT: '0' };

export function git(args, { cwd, timeout = 5000, input, env } = {}) {
  const result = spawnSync('git', args, {
    cwd,
    env: { ...process.env, ...GIT_ENV, ...env },
    encoding: 'utf8',
    timeout,
    input,
    windowsHide: true,
    maxBuffer: 64 * 1024 * 1024,
  });
  return {
    status: result.status,
    stdout: result.stdout ?? '',
    stderr: result.error ? result.error.message : result.stderr ?? '',
  };
}

export function repoRoot(cwd, { timeout = 5000 } = {}) {
  const result = git(['rev-parse', '--show-toplevel'], { cwd, timeout });
  if (result.status !== 0) return null;
  const root = result.stdout.replace(/\r?\n$/, '');
  return root === '' ? null : root;
}

