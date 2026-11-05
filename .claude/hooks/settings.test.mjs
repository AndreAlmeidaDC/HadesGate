import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..');

test('settings.json registra SessionStart e Stop do harness HadesGate sem impor permissões', () => {
  const raw = fs.readFileSync(path.join(ROOT, '.claude', 'settings.json'), 'utf8');
  const settings = JSON.parse(raw);

  assert.equal(settings.permissions, undefined, 'settings.json não deve sobrescrever permissões do usuário');
  assert.ok(Array.isArray(settings.hooks?.SessionStart), 'SessionStart deve estar configurado');
  assert.ok(Array.isArray(settings.hooks?.Stop), 'Stop deve estar configurado');

  const startCmd = settings.hooks.SessionStart[0].hooks[0];
  assert.equal(startCmd.command, 'node');
  assert.deepEqual(startCmd.args, ['${CLAUDE_PROJECT_DIR}/.claude/hooks/session-start.mjs']);

  const stopCmd = settings.hooks.Stop[0].hooks[0];
  assert.equal(stopCmd.command, 'node');
  assert.deepEqual(stopCmd.args, ['${CLAUDE_PROJECT_DIR}/.claude/hooks/verify.mjs']);
});

test('CLAUDE.md referencia @AGENTS.md e o canário [harness hadesgate]', () => {
  const claudeMd = fs.readFileSync(path.join(ROOT, 'CLAUDE.md'), 'utf8');
  assert.match(claudeMd, /^@AGENTS\.md/m);
  assert.match(claudeMd, /\[harness hadesgate\]/);
  assert.ok(fs.existsSync(path.join(ROOT, 'AGENTS.md')));
  assert.ok(fs.existsSync(path.join(ROOT, 'frontend', 'AGENTS.md')));
});

