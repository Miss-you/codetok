import assert from 'node:assert/strict';
import test from 'node:test';

import { downloadToFile } from '../scripts/download.mjs';

function makeRetriableError(message = 'temporary timeout') {
  const err = new Error(message);
  err.code = 'ETIMEDOUT';
  return err;
}

function makeNonRetriableError(message = 'permission denied') {
  const err = new Error(message);
  err.code = 'EACCES';
  return err;
}

test('download succeeds on first attempt', async () => {
  const sleepDelays = [];
  const cleanupCalls = [];
  const downloadCalls = [];

  await downloadToFile('https://example.com/a.tar.gz', '/tmp/a.tar.gz', {
    downloadOnce: async (url, destPath) => {
      downloadCalls.push({ url, destPath });
    },
    sleepFn: async (delayMs) => {
      sleepDelays.push(delayMs);
    },
    removeFile: async (destPath) => {
      cleanupCalls.push(destPath);
    },
    logger: () => {},
  });

  assert.equal(downloadCalls.length, 1);
  assert.equal(cleanupCalls.length, 1);
  assert.deepEqual(sleepDelays, []);
});

test('download succeeds after retriable failures', async () => {
  const sleepDelays = [];
  const cleanupCalls = [];
  let attempts = 0;

  await downloadToFile('https://example.com/a.tar.gz', '/tmp/a.tar.gz', {
    downloadOnce: async () => {
      attempts += 1;
      if (attempts < 3) {
        throw makeRetriableError(`timeout #${attempts}`);
      }
    },
    sleepFn: async (delayMs) => {
      sleepDelays.push(delayMs);
    },
    removeFile: async (destPath) => {
      cleanupCalls.push(destPath);
    },
    logger: () => {},
  });

  assert.equal(attempts, 3);
  assert.equal(cleanupCalls.length, 3);
  assert.deepEqual(sleepDelays, [1000, 2000]);
});

test('download fails after exhausting all retries', async () => {
  const sleepDelays = [];
  const cleanupCalls = [];
  let attempts = 0;

  await assert.rejects(
    downloadToFile('https://example.com/a.tar.gz', '/tmp/a.tar.gz', {
      downloadOnce: async () => {
        attempts += 1;
        throw makeRetriableError('timeout always');
      },
      sleepFn: async (delayMs) => {
        sleepDelays.push(delayMs);
      },
      removeFile: async (destPath) => {
        cleanupCalls.push(destPath);
      },
      logger: () => {},
    }),
    /download failed after 4 attempts/
  );

  assert.equal(attempts, 4);
  assert.equal(cleanupCalls.length, 4);
  assert.deepEqual(sleepDelays, [1000, 2000, 4000]);
});

test('non-retriable errors fail immediately', async () => {
  const sleepDelays = [];
  let attempts = 0;

  await assert.rejects(
    downloadToFile('https://example.com/a.tar.gz', '/tmp/a.tar.gz', {
      downloadOnce: async () => {
        attempts += 1;
        throw makeNonRetriableError();
      },
      sleepFn: async (delayMs) => {
        sleepDelays.push(delayMs);
      },
      removeFile: async () => {},
      logger: () => {},
    }),
    /permission denied/
  );

  assert.equal(attempts, 1);
  assert.deepEqual(sleepDelays, []);
});

test('cleanup runs before each retry attempt', async () => {
  const events = [];
  let attempts = 0;

  await downloadToFile('https://example.com/a.tar.gz', '/tmp/a.tar.gz', {
    downloadOnce: async () => {
      events.push('download');
      attempts += 1;
      if (attempts < 3) {
        throw makeRetriableError();
      }
    },
    sleepFn: async () => {},
    removeFile: async () => {
      events.push('cleanup');
    },
    logger: () => {},
  });

  assert.deepEqual(events, ['cleanup', 'download', 'cleanup', 'download', 'cleanup', 'download']);
});
