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
	client   *http.Client
	workspace string
	cookie   string
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

	doc.Find("[class*='usage'], [class*='quota'], [class*='cycle']").Each(func(i int, sel *goquery.Selection) {
		text := sel.Text()
		if strings.Contains(text, "rolling") || strings.Contains(text, "Rolling") {
			usage.Rolling = parseCycle(sel)
		} else if strings.Contains(text, "week") || strings.Contains(text, "Week") {
			usage.Weekly = parseCycle(sel)
		} else if strings.Contains(text, "month") || strings.Contains(text, "Month") {
			usage.Monthly = parseCycle(sel)
		}
	})

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

func parseCycle(sel *goquery.Selection) models.UsageData {
	data := models.UsageData{}

	sel.Find("span, div, p").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		if strings.Contains(text, "%") {
			if pct := extractPercentage(text); pct > 0 {
				data.Percentage = pct
			}
		}
		if strings.Contains(text, "day") || strings.Contains(text, "hour") || strings.Contains(text, "min") {
			data.ResetIn = extractDuration(text)
			data.ElapsedHours = estimateElapsedHours(data.ResetIn)
		}
	})

	data.Velocity = calculateVelocity(data.Percentage, data.ElapsedHours)
	data.TimeToExhaust = calculateTimeToExhaust(data.Percentage, data.Velocity)
	data.Health = determineHealth(data)

	return data
}

func (s *Scraper) fallbackCycle(cycle models.CycleType) models.UsageData {
	var resetIn time.Duration
	var elapsedHours float64

	switch cycle {
	case models.CycleRolling:
		resetIn = 24 * time.Hour
		elapsedHours = 1
	case models.CycleWeekly:
		resetIn = 7 * 24 * time.Hour
		elapsedHours = 24
	case models.CycleMonthly:
		resetIn = 30 * 24 * time.Hour
		elapsedHours = 72
	}

	return models.UsageData{
		Cycle:        cycle,
		Percentage:   0,
		ResetIn:      resetIn,
		ElapsedHours: elapsedHours,
		Velocity:     0,
		Health:       models.HealthGreen,
	}
}

var percentRegex = regexp.MustCompile(`(\d+(?:\.\d+)?)`)

func extractPercentage(text string) float64 {
	matches := percentRegex.FindStringSubmatch(text)
	if len(matches) > 1 {
		pct, _ := strconv.ParseFloat(matches[1], 64)
		return pct
	}
	return 0
}

var durationRegex = regexp.MustCompile(`(\d+)\s*(day|hour|min|hr)`)

func extractDuration(text string) time.Duration {
	matches := durationRegex.FindAllStringSubmatch(text, -1)
	var total time.Duration

	for _, m := range matches {
		val, _ := strconv.Atoi(m[1])
		switch m[2] {
		case "day":
			total += time.Duration(val) * 24 * time.Hour
		case "hour", "hr":
			total += time.Duration(val) * time.Hour
		case "min":
			total += time.Duration(val) * time.Minute
		}
	}

	if total == 0 {
		total = 24 * time.Hour
	}

	return total
}

func estimateElapsedHours(resetIn time.Duration) float64 {
	switch {
	case resetIn <= 24*time.Hour:
		return 1
	case resetIn <= 7*24*time.Hour:
		return 24
	default:
		return 72
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

func ValidateWorkspaceURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	if u.Host != "opencode.ai" && !strings.Contains(u.Host, "opencode.ai") {
		return "", fmt.Errorf("invalid workspace URL")
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	for i, part := range parts {
		if part == "workspace" && i+1 < len(parts) {
			return parts[i+1], nil
		}
	}

	if len(parts) >= 2 && parts[0] == "" {
		return parts[1], nil
	}

	return "", fmt.Errorf("workspace ID not found in URL")
}
