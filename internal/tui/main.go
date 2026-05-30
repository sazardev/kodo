package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	tab      int
	usage    *UsageData
	models   *ModelData
	context  *ContextData
	loading  bool
	err      error
	authMode string
}

type UsageData struct {
	Rolling  UsageCycle
	Weekly   UsageCycle
	Monthly  UsageCycle
	LastSync string
}

type UsageCycle struct {
	Percentage    float64
	ResetIn       string
	Trend         string
	Health        string
	Critical      bool
	CriticalMsg   string
	ExhaustDate   string
}

type ModelData struct {
	Filter      string
	Summary     ModelSummary
	TotalTokens string
}

type ModelSummary struct {
	TopByVolume string
	TopByCost   string
	Rows        []ModelRow
}

type ModelRow struct {
	Name              string
	Calls            int
	PromptTokens     string
	CompletionTokens string
	CostWeight       string
}

type ContextData struct {
	MaxContext   string
	AvgPrompt    string
	AvgResponse  string
	GrowthChart  []int64
	Warning      string
}

func New() *Model {
	return &Model{
		tab:      0,
		authMode: "auto",
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "1":
			m.tab = 0
		case "2":
			m.tab = 1
		case "3":
			m.tab = 2
		case "r":
			return m, m.refresh()
		case "m":
			m.toggleAuthMode()
		}
	}
	return m, nil
}

func (m *Model) toggleAuthMode() {
	if m.authMode == "auto" {
		m.authMode = "manual"
	} else {
		m.authMode = "auto"
	}
}

func (m *Model) refresh() tea.Cmd {
	return func() tea.Msg {
		return nil
	}
}

func (m *Model) View() string {
	header := renderHeader()
	tabs := renderTabs(m.tab)

	var content string
	switch m.tab {
	case 0:
		content = m.renderDashboard()
	case 1:
		content = m.renderModels()
	case 2:
		content = m.renderContext()
	}

	footer := renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left, header, "", tabs, content, footer)
}

func renderHeader() string {
	title := "OpenCode Telemetry & Analytics [OCTA]"
	subscription := "[Go Subscription]"
	width := 80

	header := TitleStyle.Render(title) + " " + DimStyle.Render(strings.Repeat("─", max(0, width-len(title)-len(subscription)-1))) + " " + HeaderStyle.Render(subscription)
	return header
}

func renderTabs(active int) string {
	tabs := []string{"[1] Quota Overview", "[2] Historical Models", "[3] Context Analytics"}

	var rendered []string
	for i, tab := range tabs {
		if i == active {
			rendered = append(rendered, TabActiveStyle.Render(tab))
		} else {
			rendered = append(rendered, TabInactiveStyle.Render(tab))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

func renderFooter() string {
	return DimStyle.Render("[q: Exit] [r: Refresh] [m: Switch Auth Mode]")
}

func (m *Model) renderDashboard() string {
	if m.usage == nil {
		return PlaceholderView("Loading usage data...")
	}

	return m.usage.Rolling.Render("ROLLING USAGE") +
		"\n\n" +
		m.usage.Weekly.Render("WEEKLY USAGE") +
		"\n\n" +
		m.usage.Monthly.Render("MONTHLY USAGE")
}

func (u *UsageCycle) Render(label string) string {
	bar := renderProgressBar(u.Percentage)
	line := label + " " + bar + " " + ValueStyle.Render(fmt.Sprintf("%.0f", u.Percentage)) + "%"

	if u.Critical {
		line += "\n" + CriticalStyle.Render("  CRITICAL: "+u.CriticalMsg)
		if u.ExhaustDate != "" {
			line += "\n  " + DimStyle.Render("Estimated exhaustion: "+u.ExhaustDate)
		}
	} else {
		line += "\n  " + DimStyle.Render("Resets in: "+u.ResetIn)
		line += "\n  " + DimStyle.Render("Trend: "+u.Trend)
	}

	return line
}

func renderProgressBar(percentage float64) string {
	width := 14
	filled := int(percentage / 100 * float64(width))
	if filled > width {
		filled = width
	}
	empty := width - filled

	bar := ProgressFillStyle.Render(strings.Repeat("■", filled)) +
		ProgressBarStyle.Render(strings.Repeat("░", empty))

	return "[" + bar + "]"
}

func (m *Model) renderModels() string {
	if m.models == nil {
		return PlaceholderView("Loading model data...")
	}

	header := "MODEL                    CALLS      PROMPT TOKENS   COMPL. TOKENS   EST. COST/WEIGHT"
	separator := strings.Repeat("─", len(header))

	var lines []string
	lines = append(lines, DimStyle.Render("Filter: [All Time] | [Month] | [Week] | [Day]"))
	lines = append(lines, "")
	lines = append(lines, HeaderStyle.Render(header))
	lines = append(lines, DimStyle.Render(separator))

	for _, row := range m.models.Summary.Rows {
		line := fmt.Sprintf("%-24s %-10d %-15s %-15s [%s]",
			row.Name, row.Calls, row.PromptTokens, row.CompletionTokens, row.CostWeight)
		lines = append(lines, ValueStyle.Render(line))
	}

	lines = append(lines, "")
	lines = append(lines, DimStyle.Render("Top Model by Volume: "+SuccessStyle.Render(m.models.Summary.TopByVolume)))
	lines = append(lines, DimStyle.Render("Top Model by Cost/Weight: "+WarningStyle.Render(m.models.Summary.TopByCost)))
	lines = append(lines, DimStyle.Render("Total Tokens Processed: "+m.models.TotalTokens))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m *Model) renderContext() string {
	if m.context == nil {
		return PlaceholderView("Loading context data...")
	}

	lines := []string{
		HeaderStyle.Render("CONTEXT PAYLOAD DISTRIBUTION"),
		"",
		DimStyle.Render("Max Context Recorded:  ")+ValueStyle.Render(m.context.MaxContext),
		DimStyle.Render("Average Prompt Size:   ")+ValueStyle.Render(m.context.AvgPrompt),
		DimStyle.Render("Average Response Size: ")+ValueStyle.Render(m.context.AvgResponse),
		"",
		HeaderStyle.Render("CONTEXT CREEP PROFILE (Iterative Session Growth)"),
		"",
	}

	if len(m.context.GrowthChart) > 0 {
		chart := renderSimpleChart(m.context.GrowthChart)
		lines = append(lines, chart...)
	}

	if m.context.Warning != "" {
		lines = append(lines, "")
		lines = append(lines, WarningStyle.Render("WARNING: "+m.context.Warning))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func renderSimpleChart(data []int64) []string {
	if len(data) == 0 {
		return nil
	}

	max := data[0]
	for _, v := range data {
		if v > max {
			max = v
		}
	}

	lines := make([]string, 0, len(data)+1)
	for i, v := range data {
		height := 5
		if max > 0 {
			height = int(float64(v) / float64(max) * 5)
		}
		bar := SuccessStyle.Render(strings.Repeat("■", height))
		label := fmt.Sprintf("%2d", i+1)
		lines = append(lines, DimStyle.Render(label+"k │")+"  "+bar)
	}

	return lines
}

func PlaceholderView(msg string) string {
	return "\n\n" + DimStyle.Render(msg) + "\n\n"
}
