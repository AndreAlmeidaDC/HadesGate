import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { git } from './lib/git.mjs';
import { buildSessionContext, CANARY } from './session-start.mjs';

test('buildSessionContext retorna o canário [harness hadesgate] fora de repositório git', () => {
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'hadesgate-session-'));
  try {
    const out = buildSessionContext({ cwd: tmp });
    assert.equal(out, CANARY);
    assert.equal(CANARY, '[harness hadesgate] ativo.');
  } finally {
    fs.rmSync(tmp, { recursive: true, force: true });
  }
});

test('buildSessionContext avisa quando origin/main está à frente do HEAD', () => {
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'hadesgate-behind-'));
  try {
    git(['init', '-b', 'main'], { cwd: tmp });
    git(['config', 'user.email', 'test@hadesgate.local'], { cwd: tmp });
    git(['config', 'user.name', 'HadesGate Test'], { cwd: tmp });
    fs.writeFileSync(path.join(tmp, 'a.txt'), '1\n');
    git(['add', 'a.txt'], { cwd: tmp });
    git(['commit', '-m', 'c1'], { cwd: tmp });
    const c1 = git(['rev-parse', 'HEAD'], { cwd: tmp }).stdout.trim();

    fs.writeFileSync(path.join(tmp, 'a.txt'), '2\n');
    git(['add', 'a.txt'], { cwd: tmp });
    git(['commit', '-m', 'c2'], { cwd: tmp });
    const c2 = git(['rev-parse', 'HEAD'], { cwd: tmp }).stdout.trim();

    git(['update-ref', 'refs/remotes/origin/main', c2], { cwd: tmp });
    git(['checkout', '--quiet', c1], { cwd: tmp });

    const out = buildSessionContext({ cwd: tmp });
    assert.match(out, /\[harness hadesgate\] ativo\. origin\/main local tem 1 commit/);
  } finally {
    fs.rmSync(tmp, { recursive: true, force: true });
  }
});

