package models

import "time"

type CycleType string

const (
	CycleRolling  CycleType = "rolling"
	CycleWeekly   CycleType = "weekly"
	CycleMonthly  CycleType = "monthly"
)

type HealthStatus string

const (
	HealthGreen  HealthStatus = "GREEN"
	HealthOrange HealthStatus = "ORANGE"
	HealthRed    HealthStatus = "RED"
)

type UsageData struct {
	Cycle          CycleType
	Percentage     float64
	ResetIn        time.Duration
	ElapsedHours   float64
	Velocity       float64
	TimeToExhaust  time.Duration
	Health         HealthStatus
	ExhaustionDate *time.Time
}

type ModelStats struct {
	ModelID         string
	Calls          int
	PromptTokens   int64
	CompletionTokens int64
	CostWeight     float64
}

type ModelSummary struct {
	TotalCalls          int
	TotalPromptTokens   int64
	TotalCompletionTokens int64
	TopByVolume         string
	TopByCost           string
	Models              []ModelStats
}

type ContextStats struct {
	MaxContextRecorded   int64
	AveragePromptSize   int64
	AverageResponseSize  int64
	SessionGrowth        []int64
	WarningThreshold     float64
	WarningMessage       string
}

type Session struct {
	ID        string
	Cookie    string
	Workspace string
	AuthMode  AuthMode
}

type AuthMode string

const (
	AuthModeAuto   AuthMode = "auto"
	AuthModeManual AuthMode = "manual"
)

type WorkspaceUsage struct {
	Rolling  UsageData
	Weekly   UsageData
	Monthly  UsageData
	LastSync time.Time
}

type ExecutionPayload struct {
	Timestamp       time.Time
	ModelID        string
	PromptTokens   int64
	CompletionTokens int64
}
