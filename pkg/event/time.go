package event

import "time"

type Time time.Time

func Now() Time {
	return Time(time.Now())
}

func (t Time) When() Time {
	return t
}

func (t Time) String() string {
	return time.Time(t).Format("2006-01-02 15:04:05")
}
