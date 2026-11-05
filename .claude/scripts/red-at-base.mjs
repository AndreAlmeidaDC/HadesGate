import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { git, repoRoot } from '../hooks/lib/git.mjs';
import { parseGoTestJson, testNamesIn } from '../hooks/lib/gotest.mjs';
import { isMain } from '../hooks/lib/io.mjs';
import { run } from '../hooks/lib/proc.mjs';
import { readState } from '../hooks/lib/state.mjs';

const GO_RUN_CAP_MS = 120000;
const TEST_NAME = /^Test(?!\p{Ll})[\p{L}\p{Nd}_]*$/u;
const USAGE = 'uso: node .claude/scripts/red-at-base.mjs [--base <commit>] TestA [TestB ...]';

function isAncestor(root, ancestor, descendant) {
  return git(['merge-base', '--is-ancestor', ancestor, descendant], { cwd: root }).status === 0;
}

export function resolveBase(root, sessionId) {
  const head = git(['rev-parse', '--verify', '--quiet', 'HEAD^{commit}'], { cwd: root });
  if (head.status !== 0) return null;
  const recorded = sessionId ? readState(root, sessionId).baseSha : undefined;
  const session = recorded && isAncestor(root, recorded, 'HEAD') ? recorded : null;
  const merged = git(['merge-base', 'HEAD', 'refs/remotes/origin/main'], { cwd: root });
  const main = merged.status === 0 ? merged.stdout.trim() : null;
  if (session && main && !isAncestor(root, main, session)) return main;
  return session ?? main ?? head.stdout.trim();
}

function listFiles(root, args) {
  const r = git(args, { cwd: root, timeout: 30000 });
  if (r.status !== 0) throw new Error(`git ${args.join(' ')} falhou: ${r.stderr.trim()}`);
  return r.stdout.split('\0').filter(Boolean);
}

function testPackages(root, tests) {
  const wanted = new Set(tests);
  const found = new Set();
  const pkgs = new Set();
  for (const file of listFiles(root, ['ls-files', '-z', '--cached', '--others', '--exclude-standard', '--', '*_test.go'])) {
    const absolute = path.join(root, file);
    if (!fs.existsSync(absolute)) continue;
    const declared = [...testNamesIn(fs.readFileSync(absolute, 'utf8'))].filter((name) => wanted.has(name));
    if (declared.length === 0) continue;
    declared.forEach((name) => found.add(name));
    const dir = path.posix.dirname(file);
    pkgs.add(dir === '.' ? '.' : `./${dir}`);
  }
  const missing = tests.filter((name) => !found.has(name));
  if (missing.length > 0) throw new Error(`nenhum _test.go declara ${missing.join(', ')}`);
  return [...pkgs].sort();
}

function changedSources(root, base) {
  const files = [
    ...listFiles(root, ['diff', '--name-only', '--no-renames', '-z', base, '--']),
    ...listFiles(root, ['ls-files', '-z', '--others', '--exclude-standard']),
  ];
  return [...new Set(files)].filter((file) => file.endsWith('.go') && !file.endsWith('_test.go')).sort();
}

function writeOverlay(root, base, sources, tmp) {
  const replace = {};
  sources.forEach((file, index) => {
    const old = git(['show', `${base}:${file}`], { cwd: root });
    let target = '';
    if (old.status === 0) {
      target = path.join(tmp, `${index}.go`);
      fs.writeFileSync(target, old.stdout);
    }
    replace[file] = target;
  });
  const overlay = path.join(tmp, 'overlay.json');
  fs.writeFileSync(overlay, JSON.stringify({ Replace: replace }));
  return overlay;
}

async function goTest(root, args, deadline) {
  const timeoutMs = Math.min(GO_RUN_CAP_MS, deadline - Date.now());
  if (timeoutMs <= 0) throw new Error('sem tempo para rodar o go test');
  const env = { ...process.env, CGO_ENABLED: '0' };
  const r = await run('go', ['test', ...args], { cwd: root, env, timeoutMs, maxOutput: 64 * 1024 * 1024 });
  if (r.missing) throw new Error('go não encontrado no PATH');
  if (r.timedOut) throw new Error(`go test estourou o prazo de ${Math.round(timeoutMs / 1000)} s`);
  const parsed = parseGoTestJson(r.stdout);
  const lines = parsed.output.length > 0 ? parsed.output : r.stderr.split(/\r?\n/).filter((line) => line.trim() !== '');
  if (parsed.failed.length > 0 || parsed.failedPkgs.length > 0) return { status: 'fail', lines };
  if (parsed.buildFailed.length > 0 || r.code !== 0) return { status: 'build-fail', lines };
  return { status: 'pass', lines };
}

export async function redAtBase({ root, base, tests, timeoutMs = 2 * GO_RUN_CAP_MS }) {
  const deadline = Date.now() + timeoutMs;
  const pkgs = testPackages(root, tests);
  const sources = changedSources(root, base);
  const args = ['-json', '-count=1', '-timeout', '90s', '-run', `^(${tests.join('|')})$`, ...pkgs];
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'hadesgate-red-at-base-'));
  try {
    const overlay = writeOverlay(root, base, sources, tmp);
    const atBase = await goTest(root, ['-overlay', overlay, ...args], deadline);
    const now = await goTest(root, args, deadline);
    const output = [...atBase.lines.map((line) => `base: ${line}`), ...now.lines.map((line) => `agora: ${line}`)].join('\n');
    return { atBase: atBase.status, now: now.status, pkgs, sources, output };
  } finally {
    fs.rmSync(tmp, { recursive: true, force: true, maxRetries: 5, retryDelay: 100 });
  }
}

function parseArgs(argv) {
  if (argv[0] === '--base') return { base: argv[1], tests: argv.slice(2) };
  if (argv.length > 0 && !TEST_NAME.test(argv[0])) return { base: argv[0], tests: argv.slice(1) };
  return { base: null, tests: argv };
}

function verdict(result) {
  if (result.now !== 'pass') return `O teste não passa agora (agora=${result.now}).`;
  if (result.atBase === 'pass') return 'NÃO DISCRIMINA O FIX: o teste passa no código da base.';
  if (result.atBase === 'build-fail') return 'EVIDÊNCIA FRACA: na base o teste nem compila; prefira um teste que compile na base e falhe pelo sintoma.';
  return 'OK: o teste falha no código da base e passa agora.';
}

async function main(argv) {
  const { base, tests } = parseArgs(argv);
  if (base === undefined || tests.length === 0 || !tests.every((name) => TEST_NAME.test(name))) {
    process.stderr.write(`${USAGE}\n`);
    return 2;
  }
  const root = repoRoot(process.cwd());
  if (root === null) {
    process.stderr.write('red-at-base: rode dentro de um repositório git\n');
    return 2;
  }
  const wanted = base ?? resolveBase(root, process.env.CLAUDE_CODE_SESSION_ID);
  const commit = wanted === null ? null : git(['rev-parse', '--verify', '--quiet', `${wanted}^{commit}`], { cwd: root });
  if (commit === null || commit.status !== 0) {
    process.stderr.write(`red-at-base: base inválida: ${wanted}\n`);
    return 2;
  }
  const sha = commit.stdout.trim();
  try {
    const result = await redAtBase({ root, base: sha, tests });
    const lines = [`red-at-base contra ${sha.slice(0, 12)} em ${result.pkgs.join(' ')}: base=${result.atBase}, agora=${result.now}`, verdict(result)];
    if (result.sources.length === 0) lines.push('Nenhum .go fora de teste mudou desde a base.');
    if (result.output !== '') lines.push(...result.output.split('\n').slice(0, 40));
    process.stdout.write(`${lines.join('\n')}\n`);
    return result.atBase !== 'pass' && result.now === 'pass' ? 0 : 1;
  } catch (err) {
    process.stderr.write(`red-at-base: ${err.message}\n`);
    return 2;
  }
}

if (isMain(import.meta.url)) process.exitCode = await main(process.argv.slice(2));

