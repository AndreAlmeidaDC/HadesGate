import crypto from 'node:crypto';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { redAtBase, resolveBase } from '../scripts/red-at-base.mjs';
import { git, repoRoot } from './lib/git.mjs';
import { goPkgs, newTests, parseGoTestJson } from './lib/gotest.mjs';
import { emitBlock, emitSystemMessage, isMain, readInput } from './lib/io.mjs';
import { run } from './lib/proc.mjs';
import { readState, writeState } from './lib/state.mjs';

export const BUDGET_MS = 240000;
const STEP_CAP_MS = 120000;
const MIN_STEP_MS = 5000;
const MAX_LINES = 40;
const MAX_LINE_CHARS = 200;
const DETAIL_LINES = 6;
const MAX_BLOCKED = 50;
const GO_EXTRA = new Set(['go.mod', 'go.sum']);
const PASSIVE_TASKS = new Set(['shell', 'monitor']);
const LOCATION = /^(?:vet(?:\.exe)?: )?(.+?\.go):\d+:\d+: /;
const TYPE_ERROR = /^vet(?:\.exe)?: /;
const SILENT = { kind: 'silent', text: '' };
const FOOTER = 'Corrija antes de encerrar. Se a falha não veio desta sessão, explique ao usuário e encerre: o mesmo estado não bloqueia duas vezes.';

function isGeneratedDist(file) {
  return file.startsWith('internal/web/dist/') && file !== 'internal/web/dist/.gitkeep';
}

function isBenchmarkInput(file) {
  return (
    file.startsWith('internal/anonymizer/') ||
    file.startsWith('cmd/pii-benchmark/') ||
    file.startsWith('benchmarks/pii-accuracy/')
  );
}

function isGo(file) {
  return file.endsWith('.go') || GO_EXTRA.has(file);
}

function isFrontendTs(file) {
  return file.startsWith('frontend/') && /\.(?:ts|tsx)$/.test(file) && !file.startsWith('frontend/node_modules/');
}

function relevant(file) {
  return isGo(file) || isFrontendTs(file) || isBenchmarkInput(file) || isGeneratedDist(file);
}

function nonEmpty(text) {
  return text.split(/\r?\n/).filter((line) => line.trim() !== '');
}

function listZ(root, args) {
  const r = git(args, { cwd: root, timeout: 30000 });
  if (r.status !== 0) throw new Error(`git ${args.join(' ')} falhou: ${r.stderr.trim()}`);
  return r.stdout.split('\0').filter(Boolean);
}

function changedFiles(root, base) {
  const tracked = listZ(root, ['diff', '--name-only', '--no-renames', '-z', base, '--']);
  const untracked = listZ(root, ['ls-files', '--others', '--exclude-standard', '-z']).filter((file) => !file.endsWith('/'));
  return { all: [...new Set([...tracked, ...untracked])].sort(), untracked: untracked.sort() };
}

function fingerprint(root, base, untracked) {
  const hash = crypto.createHash('sha256');
  hash.update(git(['rev-parse', 'HEAD'], { cwd: root }).stdout);
  hash.update(`\0${base}\0`);
  hash.update(git(['diff', '--binary', '--no-color', '--no-ext-diff', '--no-textconv', base, '--'], { cwd: root, timeout: 30000 }).stdout);
  for (const file of untracked) {
    const full = path.join(root, file);
    if (fs.existsSync(full) && fs.statSync(full).isFile()) {
      hash.update(`\0${file}\0`);
      hash.update(fs.readFileSync(full));
    }
  }
  return hash.digest('hex');
}

function report() {
  const findings = { blocks: [], warnings: [] };
  findings.block = (text, detail = []) => findings.blocks.push({ text, detail });
  findings.warn = (text, detail = []) => findings.warnings.push({ text, detail });
  return findings;
}

function remaining(ctx) {
  return Math.min(STEP_CAP_MS, ctx.deadline - Date.now());
}

async function step(ctx, out, label, cmd, args, options = {}) {
  const timeoutMs = remaining(ctx);
  if (timeoutMs < MIN_STEP_MS) {
    out.warn(`NÃO VERIFICADO: sem tempo para ${label}`);
    return null;
  }
  const r = await run(cmd, args, { cwd: ctx.root, timeoutMs, ...options });
  if (r.timedOut) {
    out.warn(`NÃO VERIFICADO: ${label} estourou o prazo de ${Math.round(timeoutMs / 1000)} s`);
    return null;
  }
  return r;
}

function fileOf(ctx, line) {
  const match = LOCATION.exec(line);
  if (!match) return null;
  const file = path.isAbsolute(match[1]) ? path.relative(ctx.root, match[1]) : match[1];
  return file.replace(/\\/g, '/').replace(/^\.\//, '');
}

function checkStatic(ctx, out) {
  const generated = ctx.changed.filter(isGeneratedDist);
  if (generated.length > 0) {
    out.block(`artefato gerado tocado em internal/web/dist/ (edite frontend/ e mantenha apenas .gitkeep): ${generated.join(', ')}`);
  }
}

async function checkGofmt(ctx, out) {
  const files = ctx.changed.filter((file) => file.endsWith('.go') && fs.existsSync(path.join(ctx.root, file)));
  if (files.length === 0) return;
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'hadesgate-gofmt-'));
  try {
    files.forEach((file, index) => {
      fs.writeFileSync(path.join(tmp, `${index}.go`), fs.readFileSync(path.join(ctx.root, file), 'utf8').replace(/\r\n/g, '\n'));
    });
    const r = await step(ctx, out, 'gofmt', 'gofmt', ['-l', ...files.map((_, index) => `${index}.go`)], { cwd: tmp });
    if (r === null) return;
    if (r.missing) {
      out.warn('NÃO VERIFICADO: gofmt não encontrado no PATH');
      return;
    }
    const bad = nonEmpty(r.stdout).map((name) => files[Number.parseInt(path.basename(name), 10)]);
    if (bad.length > 0) out.block(`gofmt -l acusa arquivo fora do formato: ${bad.join(', ')}`);
  } finally {
    fs.rmSync(tmp, { recursive: true, force: true, maxRetries: 5, retryDelay: 100 });
  }
}

function testNames(list) {
  return list.map((f) => `${f.pkg} ${f.test}`).join(', ');
}

async function checkAccuracy(ctx, out) {
  if (!ctx.changed.some(isBenchmarkInput)) return;
  const env = { ...process.env, CGO_ENABLED: '0' };
  const r = await step(
    ctx,
    out,
    'pii-benchmark',
    'go',
    ['run', './cmd/pii-benchmark', '-min-precision', '0.90', '-min-recall', '0.90'],
    { env },
  );
  if (r === null || r.missing) return;
  if (r.code !== 0) {
    out.block('gate de acurácia PII (precisão/recall >= 0.90) falhou', nonEmpty(`${r.stdout}\n${r.stderr}`));
  }
}

async function checkGo(ctx, out) {
  let pkgs;
  try {
    pkgs = goPkgs(ctx.root);
  } catch (err) {
    out.warn(`NÃO VERIFICADO: ${err.message}`);
    return;
  }
  const env = { ...process.env, CGO_ENABLED: '0' };
  const testFiles = ctx.changed.filter((file) => file.endsWith('_test.go'));
  const perFile = testFiles.map((file) => [file, newTests(ctx.root, ctx.base, [file])]);
  const fresh = new Set(perFile.flatMap(([, keys]) => [...keys]));
  const reproFiles = new Set(perFile.filter(([, keys]) => keys.size > 0).map(([file]) => file));
  const isFresh = (f) => fresh.has(`${f.pkg} ${f.test}`);

  const [vet, test] = await Promise.all([
    step(ctx, out, 'go vet', 'go', ['vet', ...pkgs], { env }),
    step(ctx, out, 'go test', 'go', ['test', '-json', '-count=1', '-timeout', '90s', ...pkgs], { env, maxOutput: 64 * 1024 * 1024 }),
    checkGofmt(ctx, out),
  ]);
  if (vet?.missing || test?.missing) {
    out.warn('NÃO VERIFICADO: go não encontrado no PATH; vet, testes e pii-benchmark não rodaram');
    return;
  }
  if (vet && vet.code !== 0) {
    const lines = nonEmpty(`${vet.stderr}\n${vet.stdout}`).filter((line) => !line.startsWith('#'));
    const real = lines.filter((line) => !(TYPE_ERROR.test(line) && reproFiles.has(fileOf(ctx, line))));
    if (real.length > 0 || lines.length === 0) out.block(`go vet falhou (código ${vet.code})`, real);
  }
  if (test === null) return;
  const parsed = parseGoTestJson(test.stdout);
  const regressions = parsed.failed.filter((f) => !isFresh(f));
  const reproductions = parsed.failed.filter(isFresh);
  let detail = parsed.output;
  const attach = () => {
    const shown = detail;
    detail = [];
    return shown;
  };
  if (parsed.buildFailed.length > 0) {
    const files = parsed.output.map((line) => fileOf(ctx, line)).filter(Boolean);
    if (files.length > 0 && files.every((file) => reproFiles.has(file))) {
      out.warn(`vermelho de reprodução: intencional? o teste novo não compila: ${parsed.buildFailed.join(', ')}`, attach());
    } else {
      out.block(`build quebrado: ${parsed.buildFailed.join(', ')}`, attach());
    }
  }
  if (regressions.length > 0) out.block(`teste que já existia na base falha agora: ${testNames(regressions)}`, attach());
  if (parsed.failedPkgs.length > 0) out.block(`pacote falhou sem teste culpado: ${parsed.failedPkgs.join(', ')}`, attach());
  if (reproductions.length > 0) out.warn(`vermelho de reprodução: intencional? ${testNames(reproductions)}`, attach());
  const parsedAnything = parsed.failed.length + parsed.buildFailed.length + parsed.failedPkgs.length > 0;
  if (test.code !== 0 && !parsedAnything) out.block(`go test falhou (código ${test.code})`, nonEmpty(test.stderr));
  if (test.code !== 0 || out.blocks.length > 0) return;

  await checkAccuracy(ctx, out);
  if (out.blocks.length > 0) return;

  const passing = [...new Set([...fresh].map((key) => key.slice(key.lastIndexOf(' ') + 1)))].sort();
  const sourceChanged = ctx.changed.some((file) => file.endsWith('.go') && !file.endsWith('_test.go'));
  if (passing.length === 0 || !sourceChanged) return;
  const timeoutMs = ctx.deadline - Date.now();
  if (timeoutMs < MIN_STEP_MS) {
    out.warn('NÃO VERIFICADO: sem tempo para o red-at-base');
    return;
  }
  try {
    const result = await redAtBase({ root: ctx.root, base: ctx.base, tests: passing, timeoutMs });
    if (result.atBase === 'pass') out.warn(`teste novo passa no código da base, não discrimina o fix: ${passing.join(', ')}`);
  } catch (err) {
    out.warn(`NÃO VERIFICADO: red-at-base: ${err.message}`);
  }
}

async function checkFrontend(ctx, out) {
  const frontend = path.join(ctx.root, 'frontend');
  const tsc = path.join(frontend, 'node_modules', 'typescript', 'bin', 'tsc');
  if (!fs.existsSync(tsc)) {
    out.warn('NÃO VERIFICADO: frontend/node_modules ausente; tsc --noEmit não rodou');
    return;
  }
  const r = await step(ctx, out, 'tsc', process.execPath, [tsc, '--noEmit', '-p', frontend, '--pretty', 'false']);
  if (r === null || r.code === 0) return;
  const lines = nonEmpty(`${r.stdout}\n${r.stderr}`).filter((line) => !line.replace(/\\/g, '/').startsWith('frontend/.next/'));
  if (lines.length > 0) out.block('tsc --noEmit acusou erro em frontend/', lines);
}

function render(header, findings, footer) {
  const body = [];
  for (const finding of findings) {
    body.push(`- ${finding.text}`);
    const shown = finding.detail.slice(0, DETAIL_LINES);
    body.push(...shown.map((line) => `    ${line}`));
    if (finding.detail.length > shown.length) body.push(`    ... e mais ${finding.detail.length - shown.length} linhas`);
  }
  const room = MAX_LINES - 1 - (footer ? 1 : 0);
  const kept = body.length > room ? [...body.slice(0, room - 1), `... mais ${body.length - room + 1} linhas omitidas`] : body;
  return [header, ...kept, ...(footer ? [footer] : [])]
    .map((line) => (line.length > MAX_LINE_CHARS ? `${line.slice(0, MAX_LINE_CHARS - 3)}...` : line))
    .join('\n');
}

export async function verify({ input, now = Date.now() }) {
  const tasks = Array.isArray(input.background_tasks) ? input.background_tasks : [];
  if (tasks.some((task) => !PASSIVE_TASKS.has(task?.type))) return SILENT;
  const root = repoRoot(input.cwd ?? process.env.CLAUDE_PROJECT_DIR ?? process.cwd());
  if (root === null) return SILENT;
  const sessionId = input.session_id ?? 'sem-sessao';
  const base = resolveBase(root, sessionId);
  if (base === null) return SILENT;
  const { all, untracked } = changedFiles(root, base);
  if (!all.some(relevant)) return SILENT;
  const print = fingerprint(root, base, untracked);
  const state = readState(root, sessionId);
  if (print === state.lastGreen || print === state.lastMessage) return SILENT;
  const ctx = { root, base, changed: all, deadline: now + BUDGET_MS };
  const outs = { local: report(), go: report(), frontend: report() };
  checkStatic(ctx, outs.local);
  const needsGo = all.some(isGo);
  const needsAccuracyOnly = !needsGo && all.some(isBenchmarkInput);
  await Promise.all([
    needsGo ? checkGo(ctx, outs.go) : needsAccuracyOnly ? checkAccuracy(ctx, outs.go) : null,
    all.some(isFrontendTs) ? checkFrontend(ctx, outs.frontend) : null,
  ]);
  const lists = Object.values(outs);
  const blocks = lists.flatMap((out) => out.blocks);
  const findings = [...blocks, ...lists.flatMap((out) => out.warnings)];
  const short = base.slice(0, 12);
  const blocked = state.blocked ?? [];
  if (blocks.length > 0 && !blocked.includes(print)) {
    writeState(root, sessionId, { ...state, blocked: [...blocked, print].slice(-MAX_BLOCKED) });
    return { kind: 'block', text: render(`[harness hadesgate] a verificação de fim de turno falhou contra a base ${short}:`, findings, FOOTER) };
  }
  if (findings.length > 0) {
    writeState(root, sessionId, { ...state, lastMessage: print });
    const header = blocks.length > 0
      ? `[harness hadesgate] vermelho de novo no mesmo estado depois de um bloqueio; não bloqueio de novo (base ${short}):`
      : `[harness hadesgate] verificação de fim de turno contra a base ${short}:`;
    return { kind: 'message', text: render(header, findings) };
  }
  writeState(root, sessionId, { ...state, lastGreen: print });
  return SILENT;
}

async function main() {
  try {
    const result = await verify({ input: readInput() });
    if (result.kind === 'block') emitBlock(result.text);
    else if (result.kind === 'message') emitSystemMessage(result.text);
  } catch (err) {
    emitSystemMessage(`NÃO VERIFICADO: erro do hook: ${err.message}`);
  }
}

if (isMain(import.meta.url)) await main();

