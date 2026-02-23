import fs from 'node:fs';
import { promises as fsp } from 'node:fs';
import https from 'node:https';

export const maxRedirects = 5;
export const requestTimeoutMs = 180_000;
export const maxDownloadRetries = 3;
export const retryBaseDelayMs = 1_000;
export const retriableStatusCodes = new Set([408, 425, 429, 500, 502, 503, 504]);
export const retriableErrorCodes = new Set([
  'ETIMEDOUT',
  'ECONNRESET',
  'ECONNREFUSED',
  'EPIPE',
  'EHOSTUNREACH',
  'ENETUNREACH',
  'ENOTFOUND',
  'EAI_AGAIN',
]);

export async function downloadToFile(url, destPath, options = {}) {
  const {
    maxRetries = maxDownloadRetries,
    baseDelayMs = retryBaseDelayMs,
    logger = (message) => console.warn(message),
    sleepFn = sleep,
    removeFile = (targetPath) => fsp.rm(targetPath, { force: true }),
    downloadOnce = (downloadURL, targetPath) => downloadToFileOnce(downloadURL, targetPath, 0),
    isRetriable = isRetriableDownloadError,
  } = options;

  let lastErr;
  let attemptsMade = 0;
  for (let attempt = 0; attempt <= maxRetries; attempt += 1) {
    attemptsMade = attempt + 1;
    if (attempt > 0) {
      const delayMs = baseDelayMs * 2 ** (attempt - 1);
      logger(`[codetok] retry ${attempt}/${maxRetries} in ${delayMs}ms: ${url}`);
      await sleepFn(delayMs);
    }

    await removeFile(destPath).catch(() => {});

    try {
      await downloadOnce(url, destPath);
      return;
    } catch (err) {
      lastErr = err;
      if (!isRetriable(err) || attempt === maxRetries) {
        break;
      }
    }
  }

  const attemptWord = attemptsMade === 1 ? 'attempt' : 'attempts';
  const message = lastErr ? lastErr.message : 'unknown download error';
  throw new Error(`download failed after ${attemptsMade} ${attemptWord} for ${url}: ${message}`);
}

export async function downloadToFileOnce(url, destPath, redirects = 0) {
  if (redirects > maxRedirects) {
    throw new Error(`too many redirects while downloading ${url}`);
  }

  return new Promise((resolve, reject) => {
    let done = false;
    const finish = (err) => {
      if (done) {
        return;
      }
      done = true;
      if (err) {
        reject(err);
        return;
      }
      resolve();
    };

    const req = https.get(
      url,
      {
        headers: {
          'User-Agent': 'codetok-npm-installer',
        },
      },
      (res) => {
        if (
          res.statusCode &&
          [301, 302, 303, 307, 308].includes(res.statusCode) &&
          res.headers.location
        ) {
          const nextURL = new URL(res.headers.location, url).toString();
          res.resume();
          downloadToFileOnce(nextURL, destPath, redirects + 1)
            .then(() => finish())
            .catch(finish);
          return;
        }

        if (res.statusCode !== 200) {
          res.resume();
          const statusErr = new Error(`download failed (${res.statusCode}) for ${url}`);
          statusErr.statusCode = res.statusCode;
          finish(statusErr);
          return;
        }

        const ws = fs.createWriteStream(destPath);
        ws.on('error', (err) => {
          res.destroy();
          finish(err);
        });
        ws.on('finish', () => finish());
        res.on('error', (err) => {
          ws.destroy();
          finish(err);
        });
        res.pipe(ws);
      }
    );

    req.setTimeout(requestTimeoutMs, () => {
      const timeoutErr = new Error(`download timeout after ${requestTimeoutMs}ms for ${url}`);
      timeoutErr.code = 'ETIMEDOUT';
      req.destroy(timeoutErr);
    });
    req.on('error', finish);
  });
}

export function isRetriableDownloadError(err) {
  if (!err) {
    return false;
  }

  if (typeof err.statusCode === 'number') {
    return retriableStatusCodes.has(err.statusCode);
  }

  if (typeof err.code === 'string' && retriableErrorCodes.has(err.code)) {
    return true;
  }

  return typeof err.message === 'string' && err.message.toLowerCase().includes('timeout');
}

export function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
