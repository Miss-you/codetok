#!/usr/bin/env node

const { spawn } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');

const binName = process.platform === 'win32' ? 'codetok.exe' : 'codetok';
const binPath = path.join(__dirname, '..', 'vendor', binName);

if (!fs.existsSync(binPath)) {
  console.error(
    '[codetok] native binary is missing. Reinstall with `npm install -g @yousali/codetok`.'
  );
  process.exit(1);
}

const child = spawn(binPath, process.argv.slice(2), {
  stdio: 'inherit',
});

child.on('error', (err) => {
  console.error(`[codetok] failed to start binary: ${err.message}`);
  process.exit(1);
});

child.on('exit', (code, signal) => {
  if (signal) {
    process.kill(process.pid, signal);
    return;
  }
  process.exit(code ?? 1);
});
