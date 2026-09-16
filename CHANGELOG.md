# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/2.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2026-09-16

### Changed

- The picker cycles its grouping with `tab`, and back with `shift-tab`, in place of `ctrl-f`, `ctrl-d` and `ctrl-r`.

## [0.1.0] - 2026-09-16

### Added

- `zmx-nav pick` attaches to a running zmx session, grouped flat, by working directory, or by repository with worktrees under the repository they belong to. `ctrl-f`, `ctrl-d` and `ctrl-r` swap the grouping without leaving the picker.
- `zmx-nav new` starts a session in any repository below `~/Forge` or in any worktree git records for one, named `org.repo` and `org.repo@label`. A name already running is attached rather than created.
- `--root` and `ZMX_NAV_ROOT` set where repositories are looked for.

[Unreleased]: https://github.com/mikluko/zmx-nav/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/mikluko/zmx-nav/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/mikluko/zmx-nav/releases/tag/v0.1.0
