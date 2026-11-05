import { spawn, spawnSync } from 'node:child_process';

const KILL_GRACE_MS = 2000;

function killTree(pid) {
  if (!pid) return;
  if (process.platform === 'win32') {
    spawnSync('taskkill', ['/T', '/F', '/PID', String(pid)], { stdio: 'ignore', windowsHide: true });
    return;
  }
  try {
    process.kill(-pid, 'SIGKILL');
  } catch {
    return;
  }
}

export function run(cmd, args, { cwd, timeoutMs = 120000, env, input, maxOutput = 4 * 1024 * 1024 } = {}) {
  return new Promise((resolve) => {
    const child = spawn(cmd, args, {
      cwd,
      env: env ?? process.env,
      windowsHide: true,
      detached: process.platform !== 'win32',
    });
    const out = { stdout: '', stderr: '' };
    let timedOut = false;
    let missing = false;
    let settled = false;
    let grace;
    const finish = (code) => {
      if (settled) return;
      settled = true;
      clearTimeout(timer);
      clearTimeout(grace);
      resolve({ code, stdout: out.stdout, stderr: out.stderr, timedOut, missing });
    };
    const timer = setTimeout(() => {
      timedOut = true;
      killTree(child.pid);
      grace = setTimeout(() => {
        child.stdout.destroy();
        child.stderr.destroy();
        finish(null);
      }, KILL_GRACE_MS);
    }, timeoutMs);
    for (const name of ['stdout', 'stderr']) {
      child[name].setEncoding('utf8');
      child[name].on('data', (chunk) => {
        if (out[name].length < maxOutput) out[name] += chunk.slice(0, maxOutput - out[name].length);
      });
    }
    child.on('error', (err) => {
      missing = err.code === 'ENOENT';
      out.stderr += err.message;
      finish(null);
    });
    child.on('close', (code) => finish(code));
    child.stdin.on('error', () => {});
    child.stdin.end(input);
  });
}

