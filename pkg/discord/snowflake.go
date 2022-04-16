package discord

import (
	"errors"
	"fmt"
	"strconv"
)

// Discord snowflake
// cf https://discord.com/developers/docs/reference#snowflakes
type Snowflake string

const (
	// The first second of the Discord epoch
	discordEpochPlusOne uint64 = 1 << 22
)

var ErrInvalidSnowflake = errors.New("invalid snowflake")

func (s Snowflake) String() string {
	return string(s)
}

func (s Snowflake) GoString() string {
	return "<snowflake>"
}

func (s *Snowflake) UnmarshalText(text []byte) error {
	uintValue, err := strconv.ParseUint(string(text), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid snowflake: %s", err)
	} else if uintValue < discordEpochPlusOne {
		return ErrInvalidSnowflake
	}
	*s = Snowflake(text)
	return nil
}
