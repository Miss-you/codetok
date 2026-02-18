# codetok (npm)

Install the `codetok` CLI through npm.

## Install

```bash
npm install -g codetok
```

## How it works

- During `postinstall`, this package downloads the matching binary from GitHub Releases.
- The `codetok` command then runs that native binary.

Supported targets:

- macOS: `x64`, `arm64`
- Linux: `x64`, `arm64`
- Windows: `x64`, `arm64`

## Troubleshooting

If install fails due to network restrictions, you can download binaries directly from:

https://github.com/miss-you/codetok/releases
