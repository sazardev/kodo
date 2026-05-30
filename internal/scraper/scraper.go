package scraper

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"kodo/internal/models"
)

type Scraper struct {
	client    *http.Client
	workspace string
	cookie    string
}

func NewScraper(workspace, cookie string) *Scraper {
	return &Scraper{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		workspace: workspace,
		cookie:   cookie,
	}
}

func (s *Scraper) FetchUsage() (*models.WorkspaceUsage, error) {
	targetURL := fmt.Sprintf("https://opencode.ai/workspace/%s/usage", s.workspace)
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Cookie", s.cookie)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	return s.parseUsage(doc), nil
}

func (s *Scraper) parseUsage(doc *goquery.Document) *models.WorkspaceUsage {
	usage := &models.WorkspaceUsage{
		LastSync: time.Now(),
	}

	html := doc.Text()

	usage.Rolling = s.parseCycleFromText(html, models.CycleRolling)
	usage.Weekly = s.parseCycleFromText(html, models.CycleWeekly)
	usage.Monthly = s.parseCycleFromText(html, models.CycleMonthly)

	if usage.Rolling.Percentage == 0 {
		usage.Rolling = s.fallbackCycle(models.CycleRolling)
	}
	if usage.Weekly.Percentage == 0 {
		usage.Weekly = s.fallbackCycle(models.CycleWeekly)
	}
	if usage.Monthly.Percentage == 0 {
		usage.Monthly = s.fallbackCycle(models.CycleMonthly)
	}

	return usage
}

func (s *Scraper) parseCycleFromText(html string, cycle models.CycleType) models.UsageData {
	data := models.UsageData{
		Cycle: cycle,
	}

	cycleLabels := map[models.CycleType][]string{
		models.CycleRolling: {"rolling", "Rolling", "24h", "last 24 hours"},
		models.CycleWeekly:  {"week", "Week", "weekly", "7 day"},
		models.CycleMonthly: {"month", "Month", "monthly", "30 day"},
	}

	labels := cycleLabels[cycle]

	for _, label := range labels {
		percentage := s.extractPercentageNearLabel(html, label)
		if percentage > 0 {
			data.Percentage = percentage
			break
		}
	}

	data.ResetIn = s.extractResetTime(html, labels)
	data.ElapsedHours = estimateElapsedHours(data.ResetIn, cycle)

	data.Velocity = calculateVelocity(data.Percentage, data.ElapsedHours)
	data.TimeToExhaust = calculateTimeToExhaust(data.Percentage, data.Velocity)
	data.Health = determineHealth(data)

	return data
}

func (s *Scraper) extractPercentageNearLabel(html, label string) float64 {
	idx := strings.Index(strings.ToLower(html), strings.ToLower(label))
	if idx == -1 {
		return 0
	}

	contextLength := 200
	start := idx - contextLength
	if start < 0 {
		start = 0
	}
	end := idx + len(label) + contextLength
	if end > len(html) {
		end = len(html)
	}

	context := html[start:end]

	percentRegex := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*%`)
	matches := percentRegex.FindAllStringSubmatch(context, -1)

	for i := len(matches) - 1; i >= 0; i-- {
		if pct, err := strconv.ParseFloat(matches[i][1], 64); err == nil {
			if pct <= 100 {
				return pct
			}
		}
	}

	return 0
}

func (s *Scraper) extractResetTime(html string, labels []string) time.Duration {
	for _, label := range labels {
		idx := strings.Index(strings.ToLower(html), strings.ToLower(label))
		if idx == -1 {
			continue
		}

		contextLength := 300
		start := idx - contextLength
		if start < 0 {
			start = 0
		}
		end := idx + len(label) + contextLength
		if end > len(html) {
			end = len(html)
		}

		context := html[start:end]

		if reset := extractDuration(context); reset > 0 {
			return reset
		}
	}

	return 24 * time.Hour
}

var percentRegex = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*%`)

func extractPercentage(text string) float64 {
	matches := percentRegex.FindStringSubmatch(text)
	if len(matches) > 1 {
		pct, _ := strconv.ParseFloat(matches[1], 64)
		return pct
	}
	return 0
}

var durationRegex = regexp.MustCompile(`(\d+)\s*(day|d|hour|hr|h|minute|min|m)`)

func extractDuration(text string) time.Duration {
	matches := durationRegex.FindAllStringSubmatch(text, -1)
	var total time.Duration

	for _, m := range matches {
		val, _ := strconv.Atoi(m[1])
		unit := strings.ToLower(m[2])
		switch {
		case strings.HasPrefix(unit, "day") || unit == "d":
			total += time.Duration(val) * 24 * time.Hour
		case strings.HasPrefix(unit, "hour") || strings.HasPrefix(unit, "hr") || unit == "h":
			total += time.Duration(val) * time.Hour
		case strings.HasPrefix(unit, "min") || unit == "m":
			total += time.Duration(val) * time.Minute
		}
	}

	return total
}

func estimateElapsedHours(resetIn time.Duration, cycle models.CycleType) float64 {
	resetHours := resetIn.Hours()

	switch cycle {
	case models.CycleRolling:
		return resetHours - 2
	case models.CycleWeekly:
		return resetHours - 6
	case models.CycleMonthly:
		return resetHours - 24
	default:
		return resetHours / 2
	}
}

func calculateVelocity(percentage, hours float64) float64 {
	if hours <= 0 {
		return 0
	}
	return percentage / hours
}

func calculateTimeToExhaust(percentage, velocity float64) time.Duration {
	if velocity <= 0 {
		return 0
	}
	remaining := 100.0 - percentage
	hours := remaining / velocity
	return time.Duration(hours * float64(time.Hour))
}

func determineHealth(data models.UsageData) models.HealthStatus {
	if data.Percentage >= 90 {
		return models.HealthRed
	}
	if data.Percentage >= 70 {
		return models.HealthOrange
	}
	return models.HealthGreen
}

func (s *Scraper) fallbackCycle(cycle models.CycleType) models.UsageData {
	var resetIn time.Duration

	switch cycle {
	case models.CycleRolling:
		resetIn = 24 * time.Hour
	case models.CycleWeekly:
		resetIn = 7 * 24 * time.Hour
	case models.CycleMonthly:
		resetIn = 30 * 24 * time.Hour
	}

	return models.UsageData{
		Cycle:      cycle,
		Percentage: 0,
		ResetIn:    resetIn,
		Health:     models.HealthGreen,
	}
}

func ValidateWorkspaceURL(rawURL string) (string, error) {
	if rawURL == "" {
		return "", fmt.Errorf("URL cannot be empty")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	host := strings.ToLower(u.Host)
	if !strings.Contains(host, "opencode.ai") {
		return "", fmt.Errorf("invalid workspace URL: not opencode.ai")
	}

	path := strings.Trim(u.Path, "/")
	parts := strings.Split(path, "/")

	for i, part := range parts {
		if part == "workspace" && i+1 < len(parts) {
			return parts[i+1], nil
		}
	}

	if len(parts) >= 1 && parts[0] != "" {
		return parts[0], nil
	}

	return "", fmt.Errorf("workspace ID not found in URL")
}
