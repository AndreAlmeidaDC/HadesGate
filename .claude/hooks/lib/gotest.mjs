import fs from 'node:fs';
import path from 'node:path';
import { git } from './git.mjs';

const TEST_FUNC = /^func\s+(Test(?!\p{Ll})[\p{L}\p{Nd}_]*)\s*\(\s*(?:[\p{L}\p{Nd}_]+\s+)?\*testing\.T\s*\)/gmu;
const NOISE = /^\s*(=== (RUN|PAUSE|CONT|NAME)|--- (PASS|SKIP))/;

export function goPkgs(root) {
  const makefile = path.join(root, 'Makefile');
  const content = fs.readFileSync(makefile, 'utf8').replace(/\\\r?\n/g, ' ');
  const match = /^GO_PKGS\s*[?:]?=(.*)$/m.exec(content);
  if (!match) throw new Error(`GO_PKGS não encontrado em ${makefile}`);
  return match[1].trim().split(/\s+/);
}

function append(map, key, value) {
  if (!map.has(key)) map.set(key, []);
  map.get(key).push(value);
}

export function parseGoTestJson(stdout) {
  const tests = new Map();
  const testOutput = new Map();
  const pkgOutput = new Map();
  const buildOutput = [];
  const passed = new Set();
  const failedPkg = new Set();
  const buildFailed = new Set();
  for (const line of stdout.split('\n')) {
    if (!line.startsWith('{')) continue;
    let e;
    try {
      e = JSON.parse(line);
    } catch {
      continue;
    }
    if (e.Action === 'build-output') {
      buildOutput.push(e.Output);
    } else if (e.Package && e.Test) {
      const top = e.Test.split('/')[0];
      const key = `${e.Package}\0${top}`;
      if (e.Action === 'output') append(testOutput, key, e.Output);
      else if (e.Test === top && ['run', 'pass', 'fail', 'skip'].includes(e.Action)) tests.set(key, e.Action);
    } else if (e.Package) {
      if (e.Action === 'output') append(pkgOutput, e.Package, e.Output);
      else if (e.Action === 'pass') passed.add(e.Package);
      else if (e.Action === 'fail') (e.FailedBuild ? buildFailed : failedPkg).add(e.Package);
    }
  }
  const failed = [];
  for (const [key, action] of tests) {
    const [pkg, test] = key.split('\0');
    if (action === 'fail' || (action === 'run' && failedPkg.has(pkg))) failed.push({ pkg, test });
  }
  failed.sort((a, b) => (a.pkg === b.pkg ? (a.test < b.test ? -1 : 1) : a.pkg < b.pkg ? -1 : 1));
  const failedPkgs = [...failedPkg].filter((pkg) => !failed.some((f) => f.pkg === pkg)).sort();
  const output = [
    ...failed.flatMap((f) => testOutput.get(`${f.pkg}\0${f.test}`) ?? []),
    ...buildOutput,
    ...failedPkgs.flatMap((pkg) => pkgOutput.get(pkg) ?? []),
  ]
    .map((text) => text.replace(/\r?\n$/, ''))
    .filter((text) => !NOISE.test(text));
  return { failed, buildFailed: [...buildFailed].sort(), passedPkgs: [...passed].sort(), failedPkgs, output };
}

export function testNamesIn(content) {
  return new Set([...content.matchAll(TEST_FUNC)].map((m) => m[1]));
}

function modulePath(root) {
  const goMod = path.join(root, 'go.mod');
  const match = /^module\s+(\S+)/m.exec(fs.readFileSync(goMod, 'utf8'));
  if (!match) throw new Error(`module não encontrado em ${goMod}`);
  return match[1];
}

function packageTestNames(root, base, dir) {
  const listed = git(['ls-tree', '--name-only', '-z', base, ...(dir === '.' ? [] : ['--', `${dir}/`])], { cwd: root });
  const names = new Set();
  if (listed.status !== 0) return names;
  for (const file of listed.stdout.split('\0')) {
    if (!file.endsWith('_test.go') || path.posix.dirname(file) !== dir) continue;
    const old = git(['show', `${base}:${file}`], { cwd: root });
    if (old.status === 0) testNamesIn(old.stdout).forEach((name) => names.add(name));
  }
  return names;
}

export function newTests(root, base, testFiles) {
  const module = modulePath(root);
  const known = new Map();
  const fresh = new Set();
  for (const file of testFiles) {
    const current = path.join(root, file);
    if (!fs.existsSync(current)) continue;
    const dir = path.posix.dirname(file);
    if (!known.has(dir)) known.set(dir, packageTestNames(root, base, dir));
    const pkg = dir === '.' ? module : `${module}/${dir}`;
    for (const name of testNamesIn(fs.readFileSync(current, 'utf8'))) {
      if (!known.get(dir).has(name)) fresh.add(`${pkg} ${name}`);
    }
  }
  return fresh;
}

