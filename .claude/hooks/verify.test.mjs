import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { git } from './lib/git.mjs';
import { goPkgs, parseGoTestJson, testNamesIn } from './lib/gotest.mjs';
import { recordSessionBase } from './lib/state.mjs';
import { verify } from './verify.mjs';

function initFixtureRepo(tmp) {
  git(['init', '-b', 'main'], { cwd: tmp });
  git(['config', 'user.email', 'test@hadesgate.local'], { cwd: tmp });
  git(['config', 'user.name', 'HadesGate Test'], { cwd: tmp });
  fs.writeFileSync(path.join(tmp, 'go.mod'), 'module example.com/hgfixture\n\ngo 1.26.0\n');
  fs.writeFileSync(path.join(tmp, 'Makefile'), 'GO_PKGS ?= ./...\n');
  fs.mkdirSync(path.join(tmp, 'internal', 'web', 'dist'), { recursive: true });
  fs.writeFileSync(path.join(tmp, 'internal', 'web', 'dist', '.gitkeep'), '');
  git(['add', '.'], { cwd: tmp });
  git(['commit', '-m', 'init'], { cwd: tmp });
}

test('goPkgs extrai pacotes do Makefile e testNamesIn identifica funções Test*', () => {
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'hadesgate-gotest-'));
  try {
    fs.writeFileSync(path.join(tmp, 'Makefile'), 'GO_PKGS ?= ./cmd/... ./internal/...\n');
    assert.deepEqual(goPkgs(tmp), ['./cmd/...', './internal/...']);

    const names = testNamesIn('package p\nfunc TestMaskCPF(t *testing.T) {}\nfunc helper() {}\n');
    assert.deepEqual([...names], ['TestMaskCPF']);
  } finally {
    fs.rmSync(tmp, { recursive: true, force: true });
  }
});

test('parseGoTestJson captura falhas de teste Go', () => {
  const jsonLines = [
    JSON.stringify({ Action: 'run', Package: 'example.com/hg', Test: 'TestBroken' }),
    JSON.stringify({ Action: 'output', Package: 'example.com/hg', Test: 'TestBroken', Output: '    cpf_test.go:12: expected masked\n' }),
    JSON.stringify({ Action: 'fail', Package: 'example.com/hg', Test: 'TestBroken' }),
    JSON.stringify({ Action: 'fail', Package: 'example.com/hg' }),
  ].join('\n');

  const parsed = parseGoTestJson(jsonLines);
  assert.deepEqual(parsed.failed, [{ pkg: 'example.com/hg', test: 'TestBroken' }]);
  assert.ok(parsed.output.some((line) => line.includes('cpf_test.go:12')));
});

test('verify bloqueia modificação direta em internal/web/dist/ (exceto .gitkeep) uma única vez por estado', async () => {
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'hadesgate-verify-'));
  try {
    initFixtureRepo(tmp);
    recordSessionBase(tmp, 'sess-1');

    fs.writeFileSync(path.join(tmp, 'internal', 'web', 'dist', 'index.html'), '<h1>manual edit</h1>');

    const first = await verify({ input: { cwd: tmp, session_id: 'sess-1' } });
    assert.equal(first.kind, 'block');
    assert.match(first.text, /\[harness hadesgate\]/);
    assert.match(first.text, /internal\/web\/dist\/index\.html/);

    const second = await verify({ input: { cwd: tmp, session_id: 'sess-1' } });
    assert.equal(second.kind, 'message', 'mesmo estado não deve bloquear duas vezes seguidas');
  } finally {
    fs.rmSync(tmp, { recursive: true, force: true });
  }
});

