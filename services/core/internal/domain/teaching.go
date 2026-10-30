package domain

type TeachingPreferences struct {
	Revision int64    `json:"revision"`
	Subjects []string `json:"subjects"`
	Grades   []int    `json:"grades"`
	Duration int      `json:"default_duration_minutes"`
}

func DefaultTeachingPreferences() TeachingPreferences {
	return TeachingPreferences{Subjects: []string{"math"}, Grades: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}, Duration: 60}
}
