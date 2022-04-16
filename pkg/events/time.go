package events

import (
	"fmt"
	"time"
)

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

func (t Time) Timestamp() int64 {
	return time.Time(t).Unix()
}

func (t Time) DiscordRelative() string {
	return fmt.Sprintf("<t:%d:R>", t.Timestamp())
}
