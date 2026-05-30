# SPEC.md: OpenCode Telemetry and Analytics TUI (OCTA)

This specification outlines the architecture, data flows, and interface for **OCTA**, a terminal user interface (TUI) designed to extract, analyze, and forecast consumption metrics for users with an OpenCode Go subscription.

---

## 1. Executive Summary

OpenCode currently lacks a public API endpoint for programmatic access to account usage, token history, or quota management. **OCTA** solves this by combining web scraping (via local browser cookie extraction or manual session tokens) with local telemetry mining.

By analyzing both the server-side dashboard metrics and the local application execution history, OCTA provides developers with deep operational awareness, custom-tailored token forecasting, and cost modeling directly within their terminal workflow.

---

## 2. Core Functional Requirements

### 2.1. Authentication & Session Extraction (Ingestion Layer)

The tool must support dual-mode authentication to obtain the required session cookies seamlessly:

- **Automatic Mode (Smart Local Extraction):**
- Utilize local database readers (`kooky`) to inspect local browser storage (Chrome, Firefox, Edge, Brave, Safari).
- Securely request local system permissions if required to read cookie jars.
- Search for active `opencode.ai` domain session tokens (`__session`).

- **Manual Mode (Fallback):**
- Provide an interactive prompt requesting the user to paste their `Cookie` header string or explicit session ID.
- Allow loading the session via a configuration file (`~/.config/octa/config.json`) or an environment variable (`OPENCODE_COOKIE`).

### 2.2. Metric Extraction & Aggregation

OCTA operates on a two-pronged data ingestion strategy:

#### A. Cloud Scraping (Account Level Status)

- Target Endpoint: `[https://opencode.ai/workspace/](https://opencode.ai/workspace/){workspace_id}/usage`
- **Scraped Attributes:**
- Rolling Usage percentage & Time until Reset.
- Weekly Usage percentage & Time until Reset.
- Monthly Usage percentage & Time until Reset.

#### B. Local Database Mining (Granular Forensics)

- Target Paths: `~/.local/share/opencode/opencode.db` and associated JSON message logs.
- **Harvested Attributes:**
- Timestamps of individual execution payloads.
- Model identifiers invoked (e.g., `DeepSeek V4 Pro`, `Go-Default`).
- Token counts per payload: Prompt (`prompt_tokens`) and Completion (`completion_tokens`).

### 2.3. Predictive Analytics Engine

The tool must process historical consumption velocities to generate forecasting projections:

- **Depletion Velocity ($V_c$):** Calculated per cycle (Rolling, Weekly, Monthly) based on consumed percentage relative to time elapsed.

$$V_c = \frac{\% \text{ Consumed}}{T_{\text{elapsed}} \text{ (hours)}}$$

- **Burn Rate Projection:** Estimation of remaining lifespan before absolute quota exhaustion ($T_{\text{exhaust}}$).

$$T_{\text{exhaust}} = \frac{100\% - \% \text{ Consumed}}{V_c}$$

- **Context Window Profiling:** Analyze average token payload distribution to track systemic context creep over iterative prompts.
- **Operational Health Status:**
- `GREEN`: Current burn rate safely reaches the cycle reset.
- `ORANGE`: Close margin; high-context prompts may accelerate exhaustion before the scheduled reset.
- `RED`: Current velocity guarantees quota exhaustion prior to reset.

---

## 3. Architecture & Technical Stack

```
+-------------------------------------------------------------+
|                         OCTA TUI                            |
|             (Charmbracelet Bubble Tea + Lip Gloss)           |
+-------------------------------------------------------------+
                               |
            +------------------+------------------+
            |                                     |
            v                                     v
+-----------------------+               +-----------------------+
|  Cloud Ingestion      |               |  Local Forensic Engine|
|  - goquery (Scraper)  |               |  - SQLite3 Driver     |
|  - kooky (Cookie Jar) |               |  - Token Matcher      |
+-----------------------+               +-----------------------+
            |                                     |
            v                                     v
+-------------------------------------------------------------+
|                  Predictive Analytics Core                  |
|          - Burn Rate / Time-to-Exhaustion Math              |
|          - Context Creep & Model Weight Matrix              |
+-------------------------------------------------------------+

```

- **Language:** Go (Golang) 1.22+
- **TUI Framework:**
- `[github.com/charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea)` (Elm architecture loop)
- `[github.com/charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss)` (UI Styling & Layouts)
- `[github.com/charmbracelet/bubbles/progress](https://github.com/charmbracelet/bubbles/progress)` (Progress bars)

- **HTML Parsing & Scraping:** `[github.com/PuerkitoBio/goquery](https://github.com/PuerkitoBio/goquery)`
- **Local Browser Auth:** `[github.com/browserutils/kooky](https://github.com/browserutils/kooky)`
- **Database Interface:** `database/sql` with a pure-Go SQLite driver (e.g., `modernc.org/sqlite`) to maintain zero-dependency cross-compilation.

---

## 4. User Interface & Layout Design (TUI Wireframes)

OCTA provides a multi-tab workspace layout utilizing a minimalist, high-contrast typography style.

### Tab 1: Dashboard (`usage`)

Displays real-time cloud subscription status coupled with predictive burn meters.

```
OpenCode Telemetry & Analytics [OCTA] ─────────────────────────── [Go Subscription]

 [1] Quota Overview   [2] Historical Models   [3] Context Analytics

 ROLLING USAGE [■░░░░░░░░░] 1%
 └─ Resets in: 2 hours 31 minutes
 └─ Trend: Safe (Stable)

 WEEKLY USAGE  [■■░░░░░░░░] 20%
 └─ Resets in: 1 day 6 hours
 └─ Trend: Safe (Stable)

 MONTHLY USAGE [■░░░░░░░░░] 10%
 └─ Resets in: 28 days 4 hours
 └─ CRITICAL TREND DETECTED:
    Current velocity will exhaust limits in 18.3 days.
    Estimated exhaustion date: June 17, 2026.
    [CRITICAL] Quota will deplete 10 days BEFORE system reset.

───────────────────────────────────────────────────────────────────────────────────
[q: Exit] [r: Refresh] [m: Switch Auth Mode]

```

### Tab 2: Historical Models (`models`)

Provides explicit token breakdowns parsed by model variant and localized interval filters.

```
OpenCode Telemetry & Analytics [OCTA] ─────────────────────────── [Go Subscription]

 [1] Quota Overview   [2] Historical Models   [3] Context Analytics

 Filter: [All Time] | [Month] | [Week] | [Day]

 MODEL                    CALLS      PROMPT TOKENS   COMPL. TOKENS   EST. COST/WEIGHT
 ──────────────────────────────────────────────────────────────────────────────────
 DeepSeek V4 Pro           245       245,300         82,500          [★★★★★] 72.3%
 Go-Default                120       45,000          34,200          [★★░░░] 17.5%
 Qwen 2.5 Coder 32B        45        31,000          12,000          [★░░░░] 10.2%

 Top Model by Volume: DeepSeek V4 Pro
 Top Model by Cost/Weight: DeepSeek V4 Pro
 Total Tokens Processed This Month: 450,000 tokens
───────────────────────────────────────────────────────────────────────────────────
[q: Exit] [r: Refresh] [ Arrows: Navigate Filter ]

```

### Tab 3: Context Analytics (`context`)

Focuses on tracking prompt sizes and payload distributions to identify heavy development context sessions.

```
OpenCode Telemetry & Analytics [OCTA] ─────────────────────────── [Go Subscription]

 [1] Quota Overview   [2] Historical Models   [3] Context Analytics

 CONTEXT PAYLOAD DISTRIBUTION
 Max Context Recorded:  32,451 tokens (Refactoring payload)
 Average Prompt Size:   2,310 tokens
 Average Response Size: 610 tokens

 CONTEXT CREEP PROFILE (Iterative Session Growth)
  32k ┤       *
  16k ┤      **
   8k ┤    ****
   2k ┼ *******
      └─┴─┴─┴─┴─┴─┴─┴─┴─
        Iterative Chat Sessions (Last 10 sequences)

 WARNING: 12% of your conversations scale past 16k context within 5 prompts.
 Recommendation: Clear chat history or isolate files to preserve your Go quota.
───────────────────────────────────────────────────────────────────────────────────
[q: Exit] [r: Refresh]

```

---

## 5. Security & Privacy Considerations

1. **Local Isolation:** Session credentials extracted from system storage or entered manually must stay strictly within process memory. They should never be transmitted to any third-party telemetry platform or remote server.
2. **Configuration File Security:** If session parameters or system tokens are cached locally (`~/.config/octa/config.json`), file system read/write masks must be hard-restricted explicitly to the executing system user (`0600`).
3. **Read-Only Operations:** Local database connections to local app profiles (`opencode.db`) must explicitly initialize with `mode=ro` connection parameters to guarantee that index integrity remains unaffected.
