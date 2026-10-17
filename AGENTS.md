@../project/AGENTS.md

# catch_ajfsp

Native Windows Go GUI. Standard library only; Win32 through syscall, no CGO. MIT. German user README; internal comments English.

## Components

- `internal/core`: legacy INI configuration, MD5 password hash, ajfsp parser, HTTP requests with five-second timeout and no redirect following. Legacy query-password authentication supports old Cores. Do not log request URLs or hash values. HTTP is unencrypted.
- `cmd/catch_ajfsp`: DPI-scaled Win32 window, background HTTP requests, per-user protocol registration in HKCU (no elevation). Embedded icon and asInvoker manifest.
- `build.sh`: Docker Go 1.27, format/vet/tests, Windows vet and cross-builds for amd64 and arm64. Generates ignored resource syso files using rsrc 0.10.2.
- GitHub workflow tests/builds on pushes and PRs; tags publish both executables. Dependabot weekly for Actions and Go.

## Observed legacy behavior

Original tools release 1.0.0 tested on Windows ARM64 over SSH in interactive console session. Without argument opens settings. Link argument sends GET /function/processlink with password hash from adjacent catch_ajfsp.ini; shows metadata briefly and exits. Core test calls /xml/information.xml. Registry writes go to HKCU/Software/Classes/ajfsp. Filename spaces produce malformed HTTP because original does not encode URL values.

Replacement keeps window open after delivery, encodes query values, supports web+ajfsp, validates inputs and limits HTTP timeout. It does not implement old JavaGUI selection/Launch4j -link/-path argument generation; use fixajfsp for GUI registration. No clipboard monitoring.

## Verification

Linux unit tests exercise parser, legacy INI roundtrip, request value encoding, Core information and wrong-password redirect. Both Windows architectures cross-built and vetted. ARM64 binary exercised on Windows VM against synthetic Core mock: filename with spaces and ampersand delivered, test button reports reachable, save works, HKCU handler registration verified, wrong-password dialog inspected. No live P2P traffic or real Core delivery tested. amd64 binary not run.

Never commit INI files with password hashes. Do not commit or publish unless user requests it.
