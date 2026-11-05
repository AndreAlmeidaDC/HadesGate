import fs from 'node:fs';
import { fileURLToPath } from 'node:url';

export function readInput() {
  let raw;
  try {
    raw = fs.readFileSync(0, 'utf8');
  } catch {
    return {};
  }
  if (raw.trim() === '') return {};
  try {
    const value = JSON.parse(raw);
    return value !== null && typeof value === 'object' && !Array.isArray(value) ? value : {};
  } catch {
    return {};
  }
}

function emit(value) {
  process.stdout.write(`${JSON.stringify(value)}\n`);
}

export function emitSessionContext(text) {
  emit({ hookSpecificOutput: { hookEventName: 'SessionStart', additionalContext: text } });
}

export function emitBlock(reason) {
  emit({ decision: 'block', reason });
}

export function emitSystemMessage(text) {
  emit({ systemMessage: text });
}

export function isMain(metaUrl) {
  if (!process.argv[1]) return false;
  try {
    return fs.realpathSync.native(process.argv[1]) === fs.realpathSync.native(fileURLToPath(metaUrl));
  } catch {
    return false;
  }
}

