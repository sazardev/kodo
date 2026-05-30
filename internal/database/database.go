package database

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
	"kodo/internal/models"
)

const LocalDBPath = "~/.local/share/opencode/opencode.db"

type DB struct {
	db *sql.DB
}

func New(path string) (*DB, error) {
	if strings.HasPrefix(path, "~") {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[2:])
	}

	db, err := sql.Open("sqlite", path+"?mode=ro")
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &DB{db: db}, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) FetchPayloads(limit int) ([]models.ExecutionPayload, error) {
	query := `
		SELECT timestamp, model, prompt_tokens, completion_tokens
		FROM executions
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := d.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payloads []models.ExecutionPayload
	for rows.Next() {
		var p models.ExecutionPayload
		var timestamp string
		var model string

		if err := rows.Scan(&timestamp, &model, &p.PromptTokens, &p.CompletionTokens); err != nil {
			continue
		}

		p.Timestamp, _ = time.Parse(time.RFC3339, timestamp)
		p.ModelID = model
		payloads = append(payloads, p)
	}

	return payloads, nil
}

func (d *DB) FetchModelSummary() (*models.ModelSummary, error) {
	query := `
		SELECT model, COUNT(*) as calls,
		       SUM(prompt_tokens) as prompt_total,
		       SUM(completion_tokens) as completion_total
		FROM executions
		GROUP BY model
		ORDER BY prompt_total DESC
	`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summary := &models.ModelSummary{
		Models: make([]models.ModelStats, 0),
	}

	totalPrompt := int64(0)
	totalCompletion := int64(0)

	for rows.Next() {
		var stats models.ModelStats
		if err := rows.Scan(&stats.ModelID, &stats.Calls, &stats.PromptTokens, &stats.CompletionTokens); err != nil {
			continue
		}

		stats.CostWeight = calculateCostWeight(stats.PromptTokens, stats.CompletionTokens)
		summary.Models = append(summary.Models, stats)

		summary.TotalCalls += stats.Calls
		totalPrompt += stats.PromptTokens
		totalCompletion += stats.CompletionTokens
	}

	summary.TotalPromptTokens = totalPrompt
	summary.TotalCompletionTokens = totalCompletion
	summary.TopByVolume = findTopByVolume(summary.Models)
	summary.TopByCost = findTopByCost(summary.Models)

	return summary, nil
}

func calculateCostWeight(prompt, completion int64) float64 {
	total := prompt + completion
	if total == 0 {
		return 0
	}
	weight := float64(completion) / float64(total) * 100
	return weight
}

func findTopByVolume(models []models.ModelStats) string {
	maxCalls := 0
	top := ""
	for _, m := range models {
		if m.Calls > maxCalls {
			maxCalls = m.Calls
			top = m.ModelID
		}
	}
	return top
}

func findTopByCost(models []models.ModelStats) string {
	maxWeight := 0.0
	top := ""
	for _, m := range models {
		if m.CostWeight > maxWeight {
			maxWeight = m.CostWeight
			top = m.ModelID
		}
	}
	return top
}

func (d *DB) FetchContextStats() (*models.ContextStats, error) {
	payloads, err := d.FetchPayloads(500)
	if err != nil {
		return nil, err
	}

	stats := &models.ContextStats{
		SessionGrowth: make([]int64, 0),
	}

	var totalPrompt, totalCompletion, count int64
	var maxContext int64

	for _, p := range payloads {
		if p.PromptTokens > maxContext {
			maxContext = p.PromptTokens
		}
		totalPrompt += p.PromptTokens
		totalCompletion += p.CompletionTokens
		count++
	}

	if count > 0 {
		stats.MaxContextRecorded = maxContext
		stats.AveragePromptSize = totalPrompt / count
		stats.AverageResponseSize = totalCompletion / count
	}

	growth := analyzeGrowth(payloads)
	stats.SessionGrowth = growth

	if maxContext > 16000 {
		stats.WarningThreshold = float64(maxContext) / 16000 * 100
		stats.WarningMessage = "High context usage detected"
	}

	return stats, nil
}

func analyzeGrowth(payloads []models.ExecutionPayload) []int64 {
	growth := make([]int64, 0, 10)
	var runningTotal int64

	for i, p := range payloads {
		if i >= 10 {
			break
		}
		runningTotal += p.PromptTokens
		growth = append(growth, runningTotal)
	}

	return growth
}

type MessageLog struct {
	Timestamp time.Time `json:"timestamp"`
	Model     string    `json:"model"`
	Tokens    TokenData `json:"tokens"`
}

type TokenData struct {
	Prompt      int64 `json:"prompt"`
	Completion  int64 `json:"completion"`
}

func (d *DB) ParseMessageLogs(logPath string) ([]models.ExecutionPayload, error) {
	data, err := os.ReadFile(logPath)
	if err != nil {
		return nil, err
	}

	var logs []MessageLog
	if err := json.Unmarshal(data, &logs); err != nil {
		var single MessageLog
		if err := json.Unmarshal(data, &single); err != nil {
			return nil, err
		}
		logs = []MessageLog{single}
	}

	payloads := make([]models.ExecutionPayload, 0, len(logs))
	for _, log := range logs {
		payloads = append(payloads, models.ExecutionPayload{
			Timestamp:       log.Timestamp,
			ModelID:         log.Model,
			PromptTokens:   log.Tokens.Prompt,
			CompletionTokens: log.Tokens.Completion,
		})
	}

	return payloads, nil
}
