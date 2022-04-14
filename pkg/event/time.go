package event

import "time"

type Time time.Time

func Now() Time {
	return Time(time.Now())
}

func (t Time) When() time.Time {
	return time.Time(t)
}

func (t Time) String() string {
	return t.When().Format("2006-01-02 15:04:05")
}
