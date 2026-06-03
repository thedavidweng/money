package prompt

import (
	"fmt"
	"io"

	huh "charm.land/huh/v2"
)

type Choice struct {
	Label string
	Value string
}

type Selector interface {
	Select(title string, choices []Choice) (string, error)
}

type HuhSelector struct {
	Input  io.Reader
	Output io.Writer
}

func (s HuhSelector) Select(title string, choices []Choice) (string, error) {
	if len(choices) == 0 {
		return "", fmt.Errorf("no choices available")
	}
	value := choices[0].Value
	options := make([]huh.Option[string], 0, len(choices))
	for i, choice := range choices {
		option := huh.NewOption(choice.Label, choice.Value)
		if i == 0 {
			option = option.Selected(true)
		}
		options = append(options, option)
	}
	form := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title(title).
			Options(options...).
			Value(&value),
	))
	if s.Input != nil {
		form = form.WithInput(s.Input)
	}
	if s.Output != nil {
		form = form.WithOutput(s.Output)
	}
	if err := form.Run(); err != nil {
		return "", err
	}
	return value, nil
}
