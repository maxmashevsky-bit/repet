package domain

import "time"

type RewardPreference struct {
	Enabled     bool       `json:"enabled"`
	EnabledFrom *time.Time `json:"enabled_from"`
	UpdatedAt   *time.Time `json:"updated_at"`
}
type RewardPreferenceInput struct {
	Enabled *bool `json:"enabled"`
}
type RewardAward struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Points   int       `json:"points"`
	EarnedAt time.Time `json:"earned_at"`
}
type RewardSummary struct {
	Enabled            bool                     `json:"enabled"`
	XP                 int                      `json:"xp"`
	Level              int                      `json:"level"`
	LevelProgress      int                      `json:"level_progress"`
	LevelTarget        int                      `json:"level_target"`
	WeeklyXP           int                      `json:"weekly_xp"`
	WeeklyAchievements int                      `json:"weekly_achievements"`
	Awards             CatalogPage[RewardAward] `json:"awards"`
}
