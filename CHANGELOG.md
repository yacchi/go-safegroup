# Changelog

## [0.1.1](https://github.com/yacchi/go-safegroup/compare/v0.1.0...v0.1.1) (2026-02-07)


### Features

* add context-aware hooks and run hooks before cancel ([52b0800](https://github.com/yacchi/go-safegroup/commit/52b0800883575a5a1fdac7837d713714b60f87d5))
* add DefaultGroup helper for DefaultPreset ([d7c5a30](https://github.com/yacchi/go-safegroup/commit/d7c5a30455b4f622b0987227098793cf6f5efc95))
* add GroupPreset-based detached helpers and default preset ([deb12ae](https://github.com/yacchi/go-safegroup/commit/deb12ae30226405b70cdeb97dd70482e7401cdbe))
* add OnPanicStderr option for recovered panic logging ([20d2266](https://github.com/yacchi/go-safegroup/commit/20d2266e9b4dbe41c390c82078d7ce0492b793e6))
* add slog stack trace support for PanicError ([20c61e8](https://github.com/yacchi/go-safegroup/commit/20c61e82ec072de9fdeeb427f10c70f260492cab))
* split detached task APIs into context and non-context variants ([307d531](https://github.com/yacchi/go-safegroup/commit/307d5310a1640eaa8869476cc641e9ff450dc5b4))

## 0.1.0 (2026-02-06)


### Features

* cancel GoLabel when context is done while waiting for semaphore ([a37b794](https://github.com/yacchi/go-safegroup/commit/a37b79462ca5547392e7b648bec16bcfd215cd57))
* improve PanicError nil safety and add GoDoc details ([83db6b2](https://github.com/yacchi/go-safegroup/commit/83db6b207d0fd2c16f6b8797abd7fe5302d4e004))
* initial implementation of safegroup package ([6e7162f](https://github.com/yacchi/go-safegroup/commit/6e7162f0cbcc1a0b7201d1b636cb385c7d4ec2e1))
* reject new tasks after Wait returns ([189f253](https://github.com/yacchi/go-safegroup/commit/189f25374ccc4d220e21e919ceed6cab99e17dee))

## Changelog

All notable changes to this project are documented in this file.
