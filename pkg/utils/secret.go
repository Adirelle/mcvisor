package utils

type Secret string

func (s *Secret) Reveal() string {
	if s == nil {
		return ""
	} else {
		return string(*s)
	}
}

func (s Secret) MarshalJSON() ([]byte, error) {
	return []byte(s), nil
}

func (Secret) String() string {
	return "<secret>"
}

func (Secret) GoString() string {
	return "<secret>"
}
