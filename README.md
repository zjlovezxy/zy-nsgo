# zy-nsgo

NSGo refactoring skeleton for zyfun (T1/T2 Apple CMS sources only, without IPTV, web sniffer, or T3 local script runtime).

## Modules

- `internal/cms`: T1 (XML) and T2 (JSON) Apple CMS client (`Categories`, `List`, `Detail`, `Search`, `AggregateSearch`, `ParsePlaySources`).
- `internal/store`: Pure-Go SQLite persistence (`modernc.org/sqlite`) for `sites`, `history`, `favorites`, and `settings`, including zyfun JSON site import.
- `internal/player`: Reactive NSGo `core.Video` controller with resume-seek, playback rate, progress persistence, and optional proxy URL wrapping.
- `internal/proxy`: Optional local `127.0.0.1` HTTP proxy for `#EXT-X-DISCONTINUITY` ad-block stripping, fake PNG/JPEG/GIF header removal on TS segments, and upstream header injection.

## Commands

```bash
go test -v ./...
go build ./...
```
