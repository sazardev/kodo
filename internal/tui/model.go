package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"kodo/internal/models"
)

type Model struct {
	tab       int
	loading   bool
	authMode  string
	workspace string
	err       error

	usage   *TUIUsage
	models  *TUIModels
	context *TUIContext

	refreshFn func() *ServiceData
}

type ServiceData struct {
	Usage   *models.WorkspaceUsage
	Models  *models.ModelSummary
	Context *models.ContextStats
	Summary *ServiceSummary
}

type ServiceSummary struct {
	HealthStatus string
	LastSync     time.Time
	AuthMode     string
	Workspace    string
	Error        error
}

type TUIUsage struct {
	Rolling  TUICycle
	Weekly   TUICycle
	Monthly  TUICycle
	LastSync string
}

type TUICycle struct {
	Percentage  float64
	ResetIn     string
	Trend       string
	Health      string
	Critical    bool
	CriticalMsg string
	ExhaustDate string
}

type TUIModels struct {
	Filter      string
	Summary     TUIModelSummary
	TotalTokens string
}

type TUIModelSummary struct {
	TopByVolume string
	TopByCost   string
	Rows        []TUIModelRow
}

type TUIModelRow struct {
	Name              string
	Calls             int
	PromptTokens      string
	CompletionTokens  string
	CostWeight        string
}

type TUIContext struct {
	MaxContext   string
	AvgPrompt    string
	AvgResponse  string
	GrowthChart  []int64
	Warning      string
}

func New(refreshFn func() *ServiceData) *Model {
	return &Model{
		tab:        0,
		authMode:   "auto",
		workspace:  "",
		refreshFn:  refreshFn,
	}
}

func (m *Model) Init() tea.Cmd {
	return m.loadData
}

func (m *Model) loadData() tea.Msg {
	if m.refreshFn == nil {
		return nil
	}
	data := m.refreshFn()
	return DataLoadedMsg{Data: data}
}

type DataLoadedMsg struct {
	Data *ServiceData
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
			m.loading = true
			return m, m.loadData
		}

	case DataLoadedMsg:
		m.loading = false
		m.updateWithData(msg.Data)
	}

	return m, nil
}

func (m *Model) updateWithData(data *ServiceData) {
	if data == nil {
		return
	}

	m.authMode = data.Summary.AuthMode
	m.workspace = data.Summary.Workspace
	m.err = data.Summary.Error

	if data.Usage != nil {
		m.usage = convertUsage(data.Usage)
	}

	if data.Models != nil {
		m.models = convertModels(data.Models)
	}

	if data.Context != nil {
		m.context = convertContext(data.Context)
	}
}

func convertUsage(u *models.WorkspaceUsage) *TUIUsage {
	if u == nil {
		return nil
	}
	return &TUIUsage{
		Rolling:  convertCycle(u.Rolling),
		Weekly:   convertCycle(u.Weekly),
		Monthly:  convertCycle(u.Monthly),
		LastSync: u.LastSync.Format("15:04:05"),
	}
}

func convertCycle(c models.UsageData) TUICycle {
	cycle := TUICycle{
		Percentage: c.Percentage,
		ResetIn:    formatDuration(c.ResetIn),
	}

	switch c.Health {
	case models.HealthGreen:
		cycle.Trend = "Safe (Stable)"
		cycle.Health = "GREEN"
	case models.HealthOrange:
		cycle.Trend = "Caution"
		cycle.Health = "ORANGE"
		cycle.Critical = true
		cycle.CriticalMsg = "Elevated burn rate detected"
		if c.ExhaustionDate != nil {
			cycle.ExhaustDate = c.ExhaustionDate.Format("Jan 02, 2006")
		}
	case models.HealthRed:
		cycle.Trend = "Critical"
		cycle.Health = "RED"
		cycle.Critical = true
		cycle.CriticalMsg = fmt.Sprintf("Exhaustion in %.1f hours", c.TimeToExhaust.Hours())
		if c.ExhaustionDate != nil {
			cycle.ExhaustDate = c.ExhaustionDate.Format("Jan 02, 2006")
		}
	}

	return cycle
}

func formatDuration(d time.Duration) string {
	if d == 0 {
		return "N/A"
	}

	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%d days %d hours", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%d hours %d minutes", hours, mins)
	}
	return fmt.Sprintf("%d minutes", mins)
}

func convertModels(s *models.ModelSummary) *TUIModels {
	if s == nil {
		return nil
	}

	data := &TUIModels{
		Filter: "All Time",
		Summary: TUIModelSummary{
			TopByVolume: s.TopByVolume,
			TopByCost:   s.TopByCost,
			Rows:        make([]TUIModelRow, 0, len(s.Models)),
		},
		TotalTokens: formatTokens(s.TotalPromptTokens + s.TotalCompletionTokens),
	}

	for _, m := range s.Models {
		data.Summary.Rows = append(data.Summary.Rows, TUIModelRow{
			Name:             m.ModelID,
			Calls:            m.Calls,
			PromptTokens:     formatTokens(m.PromptTokens),
			CompletionTokens: formatTokens(m.CompletionTokens),
			CostWeight:       formatCostWeight(m.CostWeight),
		})
	}

	return data
}

func formatTokens(t int64) string {
	if t >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(t)/1_000_000)
	}
	if t >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(t)/1_000)
	}
	return fmt.Sprintf("%d", t)
}

func formatCostWeight(w float64) string {
	filled := int(w / 20)
	if filled > 5 {
		filled = 5
	}
	empty := 5 - filled
	return strings.Repeat("★", filled) + strings.Repeat("░", empty)
}

func convertContext(c *models.ContextStats) *TUIContext {
	if c == nil {
		return nil
	}

	data := &TUIContext{
		MaxContext:  fmt.Sprintf("%d tokens", c.MaxContextRecorded),
		AvgPrompt:   fmt.Sprintf("%d tokens", c.AveragePromptSize),
		AvgResponse: fmt.Sprintf("%d tokens", c.AverageResponseSize),
		GrowthChart: c.SessionGrowth,
	}

	if c.WarningMessage != "" {
		data.Warning = fmt.Sprintf("%s (%.0f%% of 16k limit)", c.WarningMessage, c.WarningThreshold)
	}

	return data
}

func (m *Model) View() string {
	header := renderHeader(m.workspace, m.authMode)
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

	footer := renderFooter(m.loading)

	if m.err != nil {
		content = ErrorBanner(m.err.Error()) + "\n\n" + content
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, "", tabs, content, footer)
}

func renderHeader(workspace, authMode string) string {
	title := TitleStyle.Render("OpenCode Telemetry & Analytics [OCTA]")
	sub := HeaderStyle.Render("[Go Subscription]")

	left := lipgloss.JoinHorizontal(lipgloss.Bottom, title, DimStyle.Render(" ── "), sub)

	if workspace != "" {
		ws := DimStyle.Render("Workspace: ") + ValueStyle.Render(workspace)
		right := lipgloss.JoinHorizontal(lipgloss.Bottom, ws, DimStyle.Render(" | "), DimStyle.Render("Auth: "), ValueStyle.Render(authMode))
		return lipgloss.JoinHorizontal(lipgloss.Top, left, DimStyle.Render(strings.Repeat(" ", 30))+right)
	}

	return left
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

func renderFooter(loading bool) string {
	if loading {
		return DimStyle.Render("[Loading... Press ESC to cancel]")
	}
	return DimStyle.Render("[q: Exit] [r: Refresh] [1-3: Switch Tab]")
}

func (m *Model) renderDashboard() string {
	if m.usage == nil {
		return PlaceholderView("No usage data available. Press [r] to refresh.")
	}

	return m.usage.Rolling.Render("ROLLING USAGE") +
		"\n\n" +
		m.usage.Weekly.Render("WEEKLY USAGE") +
		"\n\n" +
		m.usage.Monthly.Render("MONTHLY USAGE") +
		"\n\n" +
		DimStyle.Render("Last sync: ")+m.usage.LastSync
}

func (u *TUIUsage) Render(label string) string {
	return u.Rolling.Render(label)
}

func (c *TUICycle) Render(label string) string {
	bar := renderProgressBar(c.Percentage)
	line := label + " " + bar + " " + ValueStyle.Render(fmt.Sprintf("%.0f", c.Percentage)) + "%"

	if c.Critical {
		line += "\n" + CriticalStyle.Render("  ⚠ "+c.CriticalMsg)
		if c.ExhaustDate != "" {
			line += "\n  " + DimStyle.Render("Estimated exhaustion: "+c.ExhaustDate)
		}
	} else {
		line += "\n  " + DimStyle.Render("Resets in: "+c.ResetIn)
		line += "\n  " + getTrendStyle(c.Health).Render("  Trend: "+c.Trend)
	}

	return line
}

func getTrendStyle(health string) lipgloss.Style {
	switch health {
	case "GREEN":
		return SuccessStyle
	case "ORANGE":
		return WarningStyle
	case "RED":
		return CriticalStyle
	default:
		return DimStyle
	}
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
		return PlaceholderView("No model data available. Press [r] to refresh.")
	}

	header := HeaderStyle.Render("MODEL                    CALLS      PROMPT TOKENS   COMPL. TOKENS   EST. COST/WEIGHT")
	separator := DimStyle.Render(strings.Repeat("─", 70))

	var lines []string
	lines = append(lines, DimStyle.Render("Filter: [All Time] | [Month] | [Week] | [Day]"))
	lines = append(lines, "")
	lines = append(lines, header)
	lines = append(lines, separator)

	for _, row := range m.models.Summary.Rows {
		line := fmt.Sprintf("%-24s %-10d %-15s %-15s [%-5s]",
			row.Name, row.Calls, row.PromptTokens, row.CompletionTokens, row.CostWeight)
		lines = append(lines, ValueStyle.Render(line))
	}

	lines = append(lines, "")
	lines = append(lines, DimStyle.Render("Top Model by Volume: "+SuccessStyle.Render(m.models.Summary.TopByVolume)))
	lines = append(lines, DimStyle.Render("Top Model by Cost/Weight: "+WarningStyle.Render(m.models.Summary.TopByCost)))
	lines = append(lines, DimStyle.Render("Total Tokens: "+m.models.TotalTokens))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m *Model) renderContext() string {
	if m.context == nil {
		return PlaceholderView("No context data available. Press [r] to refresh.")
	}

	lines := []string{
		HeaderStyle.Render("CONTEXT PAYLOAD DISTRIBUTION"),
		"",
		Row("Max Context Recorded:", m.context.MaxContext),
		Row("Average Prompt Size:", m.context.AvgPrompt),
		Row("Average Response Size:", m.context.AvgResponse),
		"",
		HeaderStyle.Render("CONTEXT CREEP PROFILE"),
		"",
	}

	if len(m.context.GrowthChart) > 0 {
		chart := renderSparkline(m.context.GrowthChart)
		lines = append(lines, chart...)
	}

	if m.context.Warning != "" {
		lines = append(lines, "")
		lines = append(lines, WarningStyle.Render("⚠ "+m.context.Warning))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func Row(label, value string) string {
	return DimStyle.Render(label) + "  " + ValueStyle.Render(value)
}

func renderSparkline(data []int64) []string {
	if len(data) == 0 {
		return nil
	}

	lines := make([]string, 0, len(data)+2)
	lines = append(lines, DimStyle.Render("  Seq │ Tokens"))

	max := data[0]
	for _, v := range data {
		if v > max {
			max = v
		}
	}

	for i, v := range data {
		height := 5
		if max > 0 {
			height = int(float64(v) / float64(max) * 5)
		}
		if height < 1 {
			height = 1
		}
		bar := SuccessStyle.Render(strings.Repeat("█", height))
		lines = append(lines, fmt.Sprintf("  %2d  │ %s %d", i+1, bar, v))
	}

	return lines
}

func PlaceholderView(msg string) string {
	return "\n\n" + Center(msg) + "\n\n"
}

func Center(msg string) string {
	pad := (80 - len(msg)) / 2
	if pad < 0 {
		pad = 0
	}
	return DimStyle.Render(strings.Repeat(" ", pad) + msg)
}

func ErrorBanner(err string) string {
	return ErrorStyle.Render("✗ Error: ") + DimStyle.Render(err)
}
