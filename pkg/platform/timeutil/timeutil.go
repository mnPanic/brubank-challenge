package timeutil

import "time"

// 2021-01-17T18:57:34Z
const LayoutISO8601 = "2006-01-02T15:04:05Z"

// Period represents a period of time
type Period struct {
	Start time.Time
	End   time.Time
}

func (p Period) Contains(t time.Time) bool {
	return t.After(p.Start) && t.Before(p.End)
}
