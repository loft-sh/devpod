package huh

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh/accessibility"
	"github.com/charmbracelet/lipgloss"
)

// Input is a input field.
//
// The input field is a field that allows the user to enter text. Use it to user
// input. It can be used for collecting text, passwords, or other short input.
//
// The input field supports Suggestions, Placeholder, and Validation.
type Input struct {
	accessor Accessor[string]
	key      string
	id       int

	title       Eval[string]
	description Eval[string]
	placeholder Eval[string]
	suggestions Eval[[]string]

	textinput textinput.Model

	inline   bool
	validate func(string) error
	err      error
	focused  bool

	accessible bool
	width      int
	height     int // not really used anywhere

	theme  *Theme
	keymap InputKeyMap
}

// NewInput creates a new input field.
//
// The input field is a field that allows the user to enter text. Use it to user
// input. It can be used for collecting text, passwords, or other short input.
//
// The input field supports Suggestions, Placeholder, and Validation.
func NewInput() *Input {
	input := textinput.New()

	i := &Input{
		accessor:    &EmbeddedAccessor[string]{},
		textinput:   input,
		validate:    func(string) error { return nil },
		id:          nextID(),
		title:       Eval[string]{cache: make(map[uint64]string)},
		description: Eval[string]{cache: make(map[uint64]string)},
		placeholder: Eval[string]{cache: make(map[uint64]string)},
		suggestions: Eval[[]string]{cache: make(map[uint64][]string)},
	}

	return i
}

// Value sets the value of the input field.
func (i *Input) Value(value *string) *Input {
	return i.Accessor(NewPointerAccessor(value))
}

// Accessor sets the accessor of the input field.
func (i *Input) Accessor(accessor Accessor[string]) *Input {
	i.accessor = accessor
	i.textinput.SetValue(i.accessor.Get())
	return i
}

// Key sets the key of the input field.
func (i *Input) Key(key string) *Input {
	i.key = key
	return i
}

// Title sets the title of the input field.
//
// The Title is static for dynamic Title use `TitleFunc`.
func (i *Input) Title(title string) *Input {
	i.title.val = title
	i.title.fn = nil
	return i
}

// Description sets the description of the input field.
//
// The Description is static for dynamic Description use `DescriptionFunc`.
func (i *Input) Description(description string) *Input {
	i.description.val = description
	i.description.fn = nil
	return i
}

// TitleFunc sets the title func of the input field.
//
// The TitleFunc will be re-evaluated when the binding of the TitleFunc changes.
// This is useful when you want to display dynamic content and update the title
// when another part of your form changes.
//
// See README#Dynamic for more usage information.
func (i *Input) TitleFunc(f func() string, bindings any) *Input {
	i.title.fn = f
	i.title.bindings = bindings
	return i
}

// DescriptionFunc sets the description func of the input field.
//
// The DescriptionFunc will be re-evaluated when the binding of the
// DescriptionFunc changes. This is useful when you want to display dynamic
// content and update the description when another part of your form changes.
//
// See README#Dynamic for more usage information.
func (i *Input) DescriptionFunc(f func() string, bindings any) *Input {
	i.description.fn = f
	i.description.bindings = bindings
	return i
}

// Prompt sets the prompt of the input field.
func (i *Input) Prompt(prompt string) *Input {
	i.textinput.Prompt = prompt
	return i
}

// CharLimit sets the character limit of the input field.
func (i *Input) CharLimit(charlimit int) *Input {
	i.textinput.CharLimit = charlimit
	return i
}

// Suggestions sets the suggestions to display for autocomplete in the input
// field.
//
// The suggestions are static for dynamic suggestions use `SuggestionsFunc`.
func (i *Input) Suggestions(suggestions []string) *Input {
	i.suggestions.fn = nil

	i.textinput.ShowSuggestions = len(suggestions) > 0
	i.textinput.KeyMap.AcceptSuggestion.SetEnabled(len(suggestions) > 0)
	i.textinput.SetSuggestions(suggestions)
	return i
}

// SuggestionsFunc sets the suggestions func to display for autocomplete in the
// input field.
//
// The SuggestionsFunc will be re-evaluated when the binding of the
// SuggestionsFunc changes. This is useful when you want to display dynamic
// suggestions when another part of your form changes.
//
// See README#Dynamic for more usage information.
func (i *Input) SuggestionsFunc(f func() []string, bindings any) *Input {
	i.suggestions.fn = f
	i.suggestions.bindings = bindings
	i.suggestions.loading = true

	i.textinput.KeyMap.AcceptSuggestion.SetEnabled(f != nil)
	i.textinput.ShowSuggestions = f != nil
	return i
}

// EchoMode sets the input behavior of the text Input field.
type EchoMode textinput.EchoMode

const (
	// EchoNormal displays text as is.
	// This is the default behavior.
	EchoModeNormal EchoMode = EchoMode(textinput.EchoNormal)

	// EchoPassword displays the EchoCharacter mask instead of actual characters.
	// This is commonly used for password fields.
	EchoModePassword EchoMode = EchoMode(textinput.EchoPassword)

	// EchoNone displays nothing as characters are entered.
	// This is commonly seen for password fields on the command line.
	EchoModeNone EchoMode = EchoMode(textinput.EchoNone)
)

// EchoMode sets the echo mode of the input.
func (i *Input) EchoMode(mode EchoMode) *Input {
	i.textinput.EchoMode = textinput.EchoMode(mode)
	return i
}

// Password sets whether or not to hide the input while the user is typing.
//
// Deprecated: use EchoMode(EchoPassword) instead.
func (i *Input) Password(password bool) *Input {
	if password {
		i.textinput.EchoMode = textinput.EchoPassword
	} else {
		i.textinput.EchoMode = textinput.EchoNormal
	}
	return i
}

// Placeholder sets the placeholder of the text input.
func (i *Input) Placeholder(str string) *Input {
	i.textinput.Placeholder = str
	return i
}

// PlaceholderFunc sets the placeholder func of the text input.
func (i *Input) PlaceholderFunc(f func() string, bindings any) *Input {
	i.placeholder.fn = f
	i.placeholder.bindings = bindings
	return i
}

// Inline sets whether the title and input should be on the same line.
func (i *Input) Inline(inline bool) *Input {
	i.inline = inline
	return i
}

// Validate sets the validation function of the input field.
func (i *Input) Validate(validate func(string) error) *Input {
	i.validate = validate
	return i
}

// Error returns the error of the input field.
func (i *Input) Error() error { return i.err }

// Skip returns whether the input should be skipped or should be blocking.
func (*Input) Skip() bool { return false }

// Zoom returns whether the input should be zoomed.
func (*Input) Zoom() bool { return false }

// Focus focuses the input field.
func (i *Input) Focus() tea.Cmd {
	i.focused = true
	return i.textinput.Focus()
}

// Blur blurs the input field.
func (i *Input) Blur() tea.Cmd {
	i.focused = false
	i.accessor.Set(i.textinput.Value())
	i.textinput.Blur()
	i.err = i.validate(i.accessor.Get())
	return nil
}

// KeyBinds returns the help message for the input field.
func (i *Input) KeyBinds() []key.Binding {
	if i.textinput.ShowSuggestions {
		return []key.Binding{i.keymap.AcceptSuggestion, i.keymap.Prev, i.keymap.Submit, i.keymap.Next}
	}
	return []key.Binding{i.keymap.Prev, i.keymap.Submit, i.keymap.Next}
}

// Init initializes the input field.
func (i *Input) Init() tea.Cmd {
	i.textinput.Blur()
	return nil
}

// Update updates the input field.
func (i *Input) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case updateFieldMsg:
		var cmds []tea.Cmd
		if ok, hash := i.title.shouldUpdate(); ok {
			i.title.bindingsHash = hash
			if !i.title.loadFromCache() {
				i.title.loading = true
				cmds = append(cmds, func() tea.Msg {
					return updateTitleMsg{id: i.id, title: i.title.fn(), hash: hash}
				})
			}
		}
		if ok, hash := i.description.shouldUpdate(); ok {
			i.description.bindingsHash = hash
			if !i.description.loadFromCache() {
				i.description.loading = true
				cmds = append(cmds, func() tea.Msg {
					return updateDescriptionMsg{id: i.id, description: i.description.fn(), hash: hash}
				})
			}
		}
		if ok, hash := i.placeholder.shouldUpdate(); ok {
			i.placeholder.bindingsHash = hash
			if i.placeholder.loadFromCache() {
				i.textinput.Placeholder = i.placeholder.val
			} else {
				i.placeholder.loading = true
				cmds = append(cmds, func() tea.Msg {
					return updatePlaceholderMsg{id: i.id, placeholder: i.placeholder.fn(), hash: hash}
				})
			}
		}
		if ok, hash := i.suggestions.shouldUpdate(); ok {
			i.suggestions.bindingsHash = hash
			if i.suggestions.loadFromCache() {
				i.textinput.ShowSuggestions = len(i.suggestions.val) > 0
				i.textinput.SetSuggestions(i.suggestions.val)
			} else {
				i.suggestions.loading = true
				cmds = append(cmds, func() tea.Msg {
					return updateSuggestionsMsg{id: i.id, suggestions: i.suggestions.fn(), hash: hash}
				})
			}
		}
		return i, tea.Batch(cmds...)
	case updateTitleMsg:
		if i.id == msg.id && i.title.bindingsHash == msg.hash {
			i.title.update(msg.title)
		}
	case updateDescriptionMsg:
		if i.id == msg.id && i.description.bindingsHash == msg.hash {
			i.description.update(msg.description)
		}
	case updatePlaceholderMsg:
		if i.id == msg.id && i.placeholder.bindingsHash == msg.hash {
			i.placeholder.update(msg.placeholder)
			i.textinput.Placeholder = msg.placeholder
		}
	case updateSuggestionsMsg:
		if i.id == msg.id && i.suggestions.bindingsHash == msg.hash {
			i.suggestions.update(msg.suggestions)
			i.textinput.ShowSuggestions = len(msg.suggestions) > 0
			i.textinput.SetSuggestions(msg.suggestions)
		}
	case tea.KeyMsg:
		i.err = nil

		switch {
		case key.Matches(msg, i.keymap.Prev):
			value := i.textinput.Value()
			i.err = i.validate(value)
			if i.err != nil {
				return i, nil
			}
			cmds = append(cmds, PrevField)
		case key.Matches(msg, i.keymap.Next, i.keymap.Submit):
			value := i.textinput.Value()
			i.err = i.validate(value)
			if i.err != nil {
				return i, nil
			}
			cmds = append(cmds, NextField)
		}
	}

	i.textinput, cmd = i.textinput.Update(msg)
	cmds = append(cmds, cmd)
	i.accessor.Set(i.textinput.Value())

	return i, tea.Batch(cmds...)
}

func (i *Input) activeStyles() *FieldStyles {
	theme := i.theme
	if theme == nil {
		theme = ThemeCharm()
	}
	if i.focused {
		return &theme.Focused
	}
	return &theme.Blurred
}

// View renders the input field.
func (i *Input) View() string {
	styles := i.activeStyles()

	// NB: since the method is on a pointer receiver these are being mutated.
	// Because this runs on every render this shouldn't matter in practice,
	// however.
	i.textinput.PlaceholderStyle = styles.TextInput.Placeholder
	i.textinput.PromptStyle = styles.TextInput.Prompt
	i.textinput.Cursor.Style = styles.TextInput.Cursor
	i.textinput.Cursor.TextStyle = styles.TextInput.CursorText
	i.textinput.TextStyle = styles.TextInput.Text

	// Adjust text input size to its char limit if it fit in its width
	if i.textinput.CharLimit > 0 {
		i.textinput.Width = min(i.textinput.CharLimit, i.textinput.Width)
	}

	var sb strings.Builder
	if i.title.val != "" || i.title.fn != nil {
		sb.WriteString(styles.Title.Render(i.title.val))
		if !i.inline {
			sb.WriteString("\n")
		}
	}
	if i.description.val != "" || i.description.fn != nil {
		sb.WriteString(styles.Description.Render(i.description.val))
		if !i.inline {
			sb.WriteString("\n")
		}
	}
	sb.WriteString(i.textinput.View())

	return styles.Base.Render(sb.String())
}

// Run runs the input field in accessible mode.
func (i *Input) Run() error {
	if i.accessible {
		return i.runAccessible()
	}
	return i.run()
}

// run runs the input field.
func (i *Input) run() error {
	return Run(i)
}

// runAccessible runs the input field in accessible mode.
func (i *Input) runAccessible() error {
	styles := i.activeStyles()
	fmt.Println(styles.Title.Render(i.title.val))
	fmt.Println()
	i.accessor.Set(accessibility.PromptString("Input: ", i.validate))
	fmt.Println(styles.SelectedOption.Render("Input: " + i.accessor.Get() + "\n"))
	return nil
}

// WithKeyMap sets the keymap on an input field.
func (i *Input) WithKeyMap(k *KeyMap) Field {
	i.keymap = k.Input
	i.textinput.KeyMap.AcceptSuggestion = i.keymap.AcceptSuggestion
	return i
}

// WithAccessible sets the accessible mode of the input field.
func (i *Input) WithAccessible(accessible bool) Field {
	i.accessible = accessible
	return i
}

// WithTheme sets the theme of the input field.
func (i *Input) WithTheme(theme *Theme) Field {
	if i.theme != nil {
		return i
	}
	i.theme = theme
	return i
}

// WithWidth sets the width of the input field.
func (i *Input) WithWidth(width int) Field {
	styles := i.activeStyles()
	i.width = width
	frameSize := styles.Base.GetHorizontalFrameSize()
	promptWidth := lipgloss.Width(i.textinput.PromptStyle.Render(i.textinput.Prompt))
	titleWidth := lipgloss.Width(styles.Title.Render(i.title.val))
	descriptionWidth := lipgloss.Width(styles.Description.Render(i.description.val))
	i.textinput.Width = width - frameSize - promptWidth - 1
	if i.inline {
		i.textinput.Width -= titleWidth
		i.textinput.Width -= descriptionWidth
	}
	return i
}

// WithHeight sets the height of the input field.
func (i *Input) WithHeight(height int) Field {
	i.height = height
	return i
}

// WithPosition sets the position of the input field.
func (i *Input) WithPosition(p FieldPosition) Field {
	i.keymap.Prev.SetEnabled(!p.IsFirst())
	i.keymap.Next.SetEnabled(!p.IsLast())
	i.keymap.Submit.SetEnabled(p.IsLast())
	return i
}

// GetKey returns the key of the field.
func (i *Input) GetKey() string { return i.key }

// GetValue returns the value of the field.
func (i *Input) GetValue() any {
	return i.accessor.Get()
}
