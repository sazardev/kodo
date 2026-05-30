# kodo

**OpenCode Telemetry and Analytics TUI** — Your Go subscription, visualized in the terminal.

---

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![TUI](https://img.shields.io/badge/TUI-Bubble%20Tea-FF6B9D?style=flat-square)](https://github.com/charmbracelet/bubbletea)
[![License](https://img.shields.io/badge/License-MIT-blue?style=flat-square)](LICENSE)

---

## What is this?

kodo is a terminal dashboard that extracts, analyzes, and forecasts your **OpenCode Go subscription** consumption — directly from your terminal, no public API required.

It combines **web scraping** (via local browser cookies) with **local telemetry mining** (SQLite databases) to give you operational awareness, token forecasting, and cost modeling.

## Features

- **Quota Dashboard** — Rolling, weekly, and monthly usage with burn-rate projections
- **Historical Models** — Token breakdowns by model variant with cost/weight analysis
- **Context Analytics** — Track prompt sizes and detect context creep across sessions
- **Predictive Alerts** — Know *before* you run out of quota

## Tech Stack

| Component | Library |
|---|---|
| Language | Go 1.22+ |
| TUI | [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lip Gloss](https://github.com/charmbracelet/lipgloss) |
| Scraping | [goquery](https://github.com/PuerkitoBio/goquery) |
| Auth | [kooky](https://github.com/browserutils/kooky) (browser cookie extraction) |
| Database | SQLite3 via [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) |

## Quick Start

```bash
# Clone
git clone https://github.com/youruser/kodo.git && cd kodo

# Build
go build -o kodo .

# Run
./kodo
```

## Authentication

kodo supports two modes:

- **Automatic** — Reads session cookies from your local browsers (Chrome, Firefox, Edge, Brave, Safari)
- **Manual** — Paste your session cookie or load from `~/.config/octa/config.json`

## License

MIT
