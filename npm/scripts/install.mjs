import crypto from 'node:crypto';
import fs from 'node:fs';
import { promises as fsp } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import extract from 'extract-zip';
import { x as extractTar } from 'tar';
import { downloadToFile } from './download.mjs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const pkgRoot = path.resolve(__dirname, '..');
const pkgJSONPath = path.join(pkgRoot, 'package.json');
const vendorDir = path.join(pkgRoot, 'vendor');

const project = 'codetok';
const owner = 'miss-you';
const repo = 'codetok';

const isWindows = process.platform === 'win32';
const binaryName = isWindows ? 'codetok.exe' : 'codetok';

if (process.env.CODETOK_SKIP_INSTALL === '1') {
  console.log('[codetok] CODETOK_SKIP_INSTALL=1, skip binary install');
  process.exit(0);
}

main().catch((err) => {
  console.error(`[codetok] install failed: ${err.message}`);
  process.exit(1);
});

async function main() {
  const pkg = JSON.parse(await fsp.readFile(pkgJSONPath, 'utf8'));
  const version = pkg.version;

  if (!version || version === '0.0.0-dev') {
    console.log('[codetok] development package version detected, skip binary download');
    return;
  }

  const target = resolveTarget(process.platform, process.arch);
  const ext = target.goos === 'windows' ? 'zip' : 'tar.gz';
  const archiveName = `${project}_${version}_${target.goos}_${target.goarch}.${ext}`;
  const releaseTag = `v${version}`;
  const baseURL = `https://github.com/${owner}/${repo}/releases/download/${releaseTag}`;

  const tmpDir = await fsp.mkdtemp(path.join(os.tmpdir(), `${project}-npm-`));
  const archivePath = path.join(tmpDir, archiveName);
  const checksumsPath = path.join(tmpDir, 'checksums.txt');
  const extractDir = path.join(tmpDir, 'extract');

  try {
    await fsp.mkdir(extractDir, { recursive: true });

    await downloadToFile(`${baseURL}/${archiveName}`, archivePath);
    await downloadToFile(`${baseURL}/checksums.txt`, checksumsPath);
    await verifySHA256(checksumsPath, archivePath, archiveName);

    if (ext === 'tar.gz') {
      await extractTar({
        cwd: extractDir,
        file: archivePath,
        strict: true,
      });
    } else {
      await extract(archivePath, { dir: extractDir });
    }

    const extractedBinary = await findFileByName(extractDir, binaryName);
    if (!extractedBinary) {
      throw new Error(`binary ${binaryName} not found in ${archiveName}`);
    }

    await fsp.mkdir(vendorDir, { recursive: true });
    const destPath = path.join(vendorDir, binaryName);
    await fsp.copyFile(extractedBinary, destPath);
    if (!isWindows) {
      await fsp.chmod(destPath, 0o755);
    }

    console.log(`[codetok] installed ${binaryName} (${target.goos}/${target.goarch})`);
  } finally {
    await fsp.rm(tmpDir, { recursive: true, force: true });
  }
}

function resolveTarget(platform, arch) {
  const goos =
    platform === 'darwin'
      ? 'darwin'
      : platform === 'linux'
        ? 'linux'
        : platform === 'win32'
          ? 'windows'
          : null;

  const goarch =
    arch === 'x64'
      ? 'amd64'
      : arch === 'arm64'
        ? 'arm64'
        : null;

  if (!goos || !goarch) {
    throw new Error(`unsupported platform/arch: ${platform}/${arch}`);
  }

  return { goos, goarch };
}

async function verifySHA256(checksumsPath, filePath, archiveName) {
  const checksums = await fsp.readFile(checksumsPath, 'utf8');
  const expected = parseChecksum(checksums, archiveName);
  if (!expected) {
    throw new Error(`checksum not found for ${archiveName}`);
  }

  const actual = await sha256(filePath);
  if (actual !== expected) {
    throw new Error(`checksum mismatch for ${archiveName}: expected ${expected}, got ${actual}`);
  }
}

function parseChecksum(checksumsText, archiveName) {
  const lines = checksumsText.split(/\r?\n/);
  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed) {
      continue;
    }
    const parts = trimmed.split(/\s+/);
    if (parts.length < 2) {
      continue;
    }

    const hash = parts[0].toLowerCase();
    const fileName = parts[parts.length - 1].replace(/^\*/, '');
    if (fileName === archiveName) {
      return hash;
    }
  }
  return '';
}

async function sha256(filePath) {
  return new Promise((resolve, reject) => {
    const hash = crypto.createHash('sha256');
    const rs = fs.createReadStream(filePath);
    rs.on('error', reject);
    rs.on('data', (chunk) => hash.update(chunk));
    rs.on('end', () => resolve(hash.digest('hex')));
  });
}

async function findFileByName(rootDir, name) {
  const entries = await fsp.readdir(rootDir, { withFileTypes: true });
  for (const entry of entries) {
    const full = path.join(rootDir, entry.name);
    if (entry.isFile() && entry.name === name) {
      return full;
    }
    if (entry.isDirectory()) {
      const found = await findFileByName(full, name);
      if (found) {
        return found;
      }
    }
  }
  return '';
}
