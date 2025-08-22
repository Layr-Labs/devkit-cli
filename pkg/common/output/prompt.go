package output

import "github.com/AlecAivazis/survey/v2"

// Confirm prompts the user to confirm an action with a yes/no question.
func Confirm(prompt string) (bool, error) {
	result := false
	c := &survey.Confirm{
		Message: prompt,
	}
	err := survey.AskOne(c, &result)
	return result, err
}

// InputHiddenString prompts the user to input a string. The input is hidden from the user.
// The validator is used to validate the input. The help text is displayed to the user when they ask for help.
// There is no default value.
func InputHiddenString(prompt, help string, validator func(string) error) (string, error) {
	var result string
	i := &survey.Password{
		Message: prompt,
		Help:    help,
	}

	err := survey.AskOne(i, &result, survey.WithValidator(func(ans interface{}) error {
		if err := validator(ans.(string)); err != nil {
			return err
		}
		return nil
	}))
	return result, err
}

// InputString prompts the user to input a string. The input is visible to the user.
// The validator is used to validate the input. The help text is displayed to the user when they ask for help.
// If defaultValue is not empty, it will be used as the default value.
func InputString(prompt, help, defaultValue string, validator func(string) error) (string, error) {
	var result string
	i := &survey.Input{
		Message: prompt,
		Help:    help,
		Default: defaultValue,
	}

	var opts []survey.AskOpt
	if validator != nil {
		opts = append(opts, survey.WithValidator(func(ans interface{}) error {
			if err := validator(ans.(string)); err != nil {
				return err
			}
			return nil
		}))
	}

	err := survey.AskOne(i, &result, opts...)
	return result, err
}
