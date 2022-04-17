package utils

import "encoding/json"

type Secret string

func (s *Secret) Reveal() string {
	if s == nil {
		return ""
	} else {
		return string(*s)
	}
}

func (s Secret) MarshalJSON() ([]byte, error) {
	value := string(s)
	return json.Marshal(&value)
}

func (Secret) String() string {
	return "<secret>"
}

func (Secret) GoString() string {
	return "<secret>"
}
