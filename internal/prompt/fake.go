package prompt

import "fmt"

// FakeSelector is a test double for Selector. It is exported from production
// code because Go test files (_test.go) cannot be imported across packages.
// Tests in internal/cli and internal/plaidlogin depend on prompt.NewFake().
type FakeSelector struct {
	choices []string
}

func NewFake(choices ...string) *FakeSelector {
	return &FakeSelector{choices: choices}
}

func (s *FakeSelector) Select(title string, choices []Choice) (string, error) {
	if len(s.choices) == 0 {
		return "", fmt.Errorf("no fake prompt choice queued for %q", title)
	}
	value := s.choices[0]
	s.choices = s.choices[1:]
	for _, choice := range choices {
		if choice.Value == value {
			return value, nil
		}
	}
	return "", fmt.Errorf("fake prompt choice %q is not available for %q", value, title)
}
