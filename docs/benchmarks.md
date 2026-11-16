# Benchmarks

WaveTrack's cost is measured with Go benchmarks over a realistic office: **100 people with a year of history**, the default retention, two visits per person per working day (about 52,000 visits), everyone in the office at the time of the measurement. The raw results are in [`bench/`](bench); this page explains them.

## Results

Apple M4 (10 cores), macOS, Go 1.27.3. Medians of six runs; `benchstat` reports every difference below as significant (p ≤ 0.002) unless marked.

### What a dashboard costs

| Request | 3.0.0 | 3.1.0 | Change |
| --- | ---: | ---: | ---: |
| `GET /api/now`, the page that refreshes every 10 s | 5.79 ms | 0.42 ms | **−93%** |
| `GET /api/now`, ten dashboards refreshing at once | 6.28 ms | 0.16 ms | **−98%** |
| `GET /api/timeline`, today | 5.70 ms | 0.33 ms | −94% |
| `GET /api/history?days=7` | 6.27 ms | 1.05 ms | −83% |
| `GET /api/history?days=31` | 9.38 ms | 4.98 ms | −47% |
| `GET /api/export.csv?days=366`, a full year | 152 ms | 157 ms | +3% |

Numbers are per request and exclude the network: requests go straight to the HTTP handler. With ten dashboards in parallel, 3.1 serves about 6,400 `/api/now` requests a second on this machine, against 160 before.

### Where the time went

| Operation | 3.0.0 | 3.1.0 | Change |
| --- | ---: | ---: | ---: |
| Today's visits, 30 days of history | 496 µs | 98 µs | −80% |
| Today's visits, 365 days of history | 5,585 µs | 100 µs | **−98%** |
| A month of visits, 365 days of history | 7.21 ms | 2.29 ms | −68% |
| Extending a visit (the tracker's write, once a minute per person) | 8.40 µs | 8.36 µs | no change (p=0.12) |
| A registered phone seen (tracker, includes its writes) | 17.8 µs | 17.2 µs | −3% |
| A stranger's phone seen (memory only) | 41 ns | 43 ns | no change (p=0.70) |
| Decoding a packet and extracting its sender (Ethernet) | 185 ns | 166 ns | −10% |
| Decoding a packet and extracting its sender (802.11 monitor mode) | 243 ns | 219 ns | −10% |

The report package (daily totals, timelines) and the tracker's snapshot did not change in 3.1. Their benchmarks moved by up to 8% between runs on a busy machine, within their run-to-run spread, and are left out of the table.

## What changed, and why

**Today's visits no longer read the whole history.** Every dashboard refresh asks for the visits that overlap today. 3.0 filtered on `started_at < end of today`, which SQLite answered by walking the `started_at` index from the beginning of time: with a year of history, about 52,000 rows read to return 200. The cost grew with every day of history kept.

3.1 asks two narrow questions instead: finished visits that *ended* after the start of the window (a new index on `ended_at`), and visits still open (an existing partial index). The cost now follows the size of the window, not of the history: 98 µs with a month of history, 100 µs with a year.

An index on `COALESCE(ended_at, last_seen)` answered the same query in one step, and was tried first. It made the tracker's once-a-minute write 52% slower, because `last_seen` changes on every write and the index had to follow. `ended_at` is written once per visit, so the chosen index costs nothing on that path; the benchmark confirms it.

**Readers no longer queue behind each other.** 3.0 used one SQLite connection for everything, which serialized every read: ten dashboards took ten times as long as one. 3.1 keeps a single connection for writes (which rules out `SQLITE_BUSY`) and a pool of read-only connections; SQLite's WAL mode lets readers proceed while the writer works. Ten parallel dashboards now cost less per request than one did before.

**Writes skip a sync they don't need.** In WAL mode, `synchronous=NORMAL` is SQLite's recommended setting: a power cut can lose the last commits but never corrupts the database.

**The packet library moved.** `google/gopacket` was archived in 2020; its maintained successor, `gopacket/gopacket`, decodes the same frames about 10% faster.

**The full-year export got slightly slower.** It now writes an audit-log entry (see [security.md](security.md#the-audit-log)), and its 366-day window is answered through the new indexes. At 157 ms for a year of a hundred people's attendance, that's an acceptable price for an occasional action.

## In practice

These numbers come from one fast machine; they were not measured on small hardware. As an estimate, a Raspberry Pi 4 is roughly ten times slower on this kind of work. Scaled accordingly, `/api/now` would take about 4 ms there for a hundred people with a year of history, and decoding a busy office network's traffic (a few thousand packets a second) would need under 1% of one core before libpcap's own overhead.

## Reproduce

```bash
make bench                       # six runs of every benchmark into bench.txt
go run golang.org/x/perf/cmd/benchstat@latest docs/bench/3.1.0.txt bench.txt
```

Close other applications first: the comparisons are only as quiet as the machine. The benchmarks live next to the code they measure:

| File | Measures |
| --- | --- |
| [`internal/server/bench_test.go`](../internal/server/bench_test.go) | API endpoints, sequential and parallel |
| [`internal/store/bench_test.go`](../internal/store/bench_test.go) | The visit queries behind them, and the tracker's write |
| [`internal/presence/bench_test.go`](../internal/presence/bench_test.go) | Sightings of registered and unknown devices; the dashboard snapshot |
| [`internal/report/bench_test.go`](../internal/report/bench_test.go) | Daily totals and timelines |
| [`internal/capture/bench_test.go`](../internal/capture/bench_test.go) | Packet decoding |

The office they run against is built by `testutil.Office`, the same one the store tests use. CI runs every benchmark once on each push, so they keep compiling and passing.
