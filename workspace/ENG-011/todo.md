# ENG-011 Follow-ups

- Consider CODEOWNERS or a ruleset requiring maintainer review for `.github/workflows/`, `Makefile`, `scripts/` (reviewer suggestion; out of #56 scope).
- Reboot test: the maintainer reboots, logs in, and confirms the runner returns to Idle (cannot be done from an agent session).
- Fork-PR test with a real fork (needs a second GitHub account); covered today by configuration (approval policy readback) and workflow triggers.
- After merge, dispatch `runner-smoke.yml` from `main` once (`gh workflow run runner-smoke.yml -R Miss-you/codetok`).
