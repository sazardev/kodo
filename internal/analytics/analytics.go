package analytics

import (
	"kodo/internal/models"
	"time"
)

type Engine struct {
	usage    *models.WorkspaceUsage
	context  *models.ContextStats
	models   *models.ModelSummary
}

func New() *Engine {
	return &Engine{}
}

func (e *Engine) WithUsage(usage *models.WorkspaceUsage) *Engine {
	e.usage = usage
	return e
}

func (e *Engine) WithContext(ctx *models.ContextStats) *Engine {
	e.context = ctx
	return e
}

func (e *Engine) WithModels(models *models.ModelSummary) *Engine {
	e.models = models
	return e
}

func (e *Engine) CalculateBurnRate() {
	if e.usage == nil {
		return
	}

	e.calculateCycleBurnRate(&e.usage.Rolling)
	e.calculateCycleBurnRate(&e.usage.Weekly)
	e.calculateCycleBurnRate(&e.usage.Monthly)
}

func (e *Engine) calculateCycleBurnRate(data *models.UsageData) {
	if data.ElapsedHours <= 0 {
		data.ElapsedHours = 1
	}

	data.Velocity = data.Percentage / data.ElapsedHours

	if data.Velocity > 0 {
		remaining := 100.0 - data.Percentage
		hoursToExhaust := remaining / data.Velocity
		data.TimeToExhaust = time.Duration(hoursToExhaust * float64(time.Hour))

		exhaustionDate := time.Now().Add(data.TimeToExhaust)
		data.ExhaustionDate = &exhaustionDate
	}

	data.Health = e.determineHealth(data)
}

func (e *Engine) determineHealth(data *models.UsageData) models.HealthStatus {
	if data.Velocity <= 0 {
		return models.HealthGreen
	}

	remainingPct := 100.0 - data.Percentage
	hoursRemaining := remainingPct / data.Velocity
	resetHoursRemaining := data.ResetIn.Hours()

	if hoursRemaining <= 0 || hoursRemaining < resetHoursRemaining*0.3 {
		return models.HealthRed
	}
	if hoursRemaining < resetHoursRemaining*0.7 {
		return models.HealthOrange
	}

	return models.HealthGreen
}

func (e *Engine) ProjectExhaustionDate() *time.Time {
	if e.usage == nil {
		return nil
	}

	if e.usage.Monthly.ExhaustionDate != nil {
		return e.usage.Monthly.ExhaustionDate
	}

	return nil
}

func (e *Engine) CalculateContextCreep() float64 {
	if e.context == nil || len(e.context.SessionGrowth) < 2 {
		return 0
	}

	growth := e.context.SessionGrowth
	initial := growth[0]
	if initial <= 0 {
		return 0
	}

	final := growth[len(growth)-1]
	creep := float64(final) / float64(initial) * 100

	return creep
}

func (e *Engine) IsHighContextUser() bool {
	if e.context == nil {
		return false
	}
	return e.context.MaxContextRecorded > 16000
}

func (e *Engine) GetHealthSummary() string {
	if e.usage == nil {
		return "No data available"
	}

	cycles := []models.UsageData{
		e.usage.Rolling,
		e.usage.Weekly,
		e.usage.Monthly,
	}

	worst := models.HealthGreen
	for _, c := range cycles {
		if c.Health == models.HealthRed {
			worst = models.HealthRed
			break
		}
		if c.Health == models.HealthOrange && worst != models.HealthRed {
			worst = models.HealthOrange
		}
	}

	switch worst {
	case models.HealthRed:
		return "CRITICAL: Quota exhaustion imminent"
	case models.HealthOrange:
		return "WARNING: Burn rate elevated"
	default:
		return "OK: Usage within safe limits"
	}
}
