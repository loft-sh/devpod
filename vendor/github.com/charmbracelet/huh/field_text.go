package huh

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh/accessibility"
	"github.com/charmbracelet/lipgloss"
)

// Text is a text field.
//
// A text box is responsible for getting multi-line input from the user. Use
// it to gather longer-form user input. The Text field can be filled with an
// EDITOR.
type Text struct {
	accessor Accessor[string]
	key      string
	id       int

	title       Eval[string]
	description Eval[string]
	placeholder Eval[string]

	editorCmd       string
	editorArgs      []string
	editorExtension string

	textarea textarea.Model

	focused  bool
	validate func(string) error
	err      error

	accessible bool
	width      int

	theme  *Theme
	keymap TextKeyMap
}

// NewText creates a new text field.
//
// A text box is responsible for getting multi-line input from the user. Use
// it to gather longer-form user input. The Text field can be filled with an
// EDITOR.
func NewText() *Text {
	text := textarea.New()
	text.ShowLineNumbers = false
	text.Prompt = ""
	text.FocusedStyle.CursorLine = lipgloss.NewStyle()

	editorCmd, editorArgs := getEditor()

	t := &Text{
		accessor:        &EmbeddedAccessor[string]{},
		id:              nextID(),
		textarea:        text,
		validate:        func(string) error { return nil },
		editorCmd:       editorCmd,
		editorArgs:      editorArgs,
		editorExtension: "md",
		title:           Eval[string]{cache: make(map[uint64]string)},
		description:     Eval[string]{cache: make(map[uint64]string)},
		placeholder:     Eval[string]{cache: make(map[uint64]string)},
	}

	return t
}

// Value sets the value of the text field.
func (t *Text) Value(value *string) *Text {
	return t.Accessor(NewPointerAccessor(value))
}

// Accessor sets the accessor of the text field.
func (t *Text) Accessor(accessor Accessor[string]) *Text {
	t.accessor = accessor
	t.textarea.SetValue(t.accessor.Get())
	return t
}

// Key sets the key of the text field.
func (t *Text) Key(key string) *Text {
	t.key = key
	return t
}

// Title sets the text field's title.
//
// This title will be static, for dynamic titles use `TitleFunc`.
func (t *Text) Title(title string) *Text {
	t.title.val = title
	t.title.fn = nil
	return t
}

// TitleFunc sets the text field's title func.
//
// The TitleFunc will be re-evaluated when the binding of the TitleFunc changes.
// This is useful when you want to display dynamic content and update the title
// when another part of your form changes.
//
// See README#Dynamic for more usage information.
func (t *Text) TitleFunc(f func() string, bindings any) *Text {
	t.title.fn = f
	t.title.bindings = bindings
	return t
}

// Description sets the description of the text field.
//
// This description will be static, for dynamic description use `DescriptionFunc`.
func (t *Text) Description(description string) *Text {
	t.description.val = description
	t.description.fn = nil
	return t
}

// DescriptionFunc sets the description func of the text field.
//
// The DescriptionFunc will be re-evaluated when the binding of the
// DescriptionFunc changes. This is useful when you want to display dynamic
// content and update the description when another part of your form changes.
//
// See README#Dynamic for more usage information.
func (t *Text) DescriptionFunc(f func() string, bindings any) *Text {
	t.description.fn = f
	t.description.bindings = bindings
	return t
}

// Lines sets the number of lines to show of the text field.
func (t *Text) Lines(lines int) *Text {
	t.textarea.SetHeight(lines)
	return t
}

// CharLimit sets the character limit of the text field.
func (t *Text) CharLimit(charlimit int) *Text {
	t.textarea.CharLimit = charlimit
	return t
}

// ShowLineNumbers sets whether or not to show line numbers.
func (t *Text) ShowLineNumbers(show bool) *Text {
	t.textarea.ShowLineNumbers = show
	return t
}

// Placeholder sets the placeholder of the text field.
//
// This placeholder will be static, for dynamic placeholders use `PlaceholderFunc`.
func (t *Text) Placeholder(str string) *Text {
	t.textarea.Placeholder = str
	return t
}

// PlaceholderFunc sets the placeholder func of the text field.
//
// The PlaceholderFunc will be re-evaluated when the binding of the
// PlaceholderFunc changes. This is useful when you want to display dynamic
// content and update the placeholder when another part of your form changes.
//
// See README#Dynamic for more usage information.
func (t *Text) PlaceholderFunc(f func() string, bindings any) *Text {
	t.placeholder.fn = f
	t.placeholder.bindings = bindings
	return t
}

// Validate sets the validation function of the text field.
func (t *Text) Validate(validate func(string) error) *Text {
	t.validate = validate
	return t
}

const defaultEditor = "nano"

// getEditor returns the editor command and arguments.
func getEditor() (string, []string) {
	editor := strings.Fields(os.Getenv("EDITOR"))
	if len(editor) > 0 {
		return editor[0], editor[1:]
	}
	return defaultEditor, nil
}

// Editor specifies which editor to use.
//
// The first argument provided is used as the editor command (vim, nvim, nano, etc...)
// The following (optional) arguments provided are passed as arguments to the editor command.
func (t *Text) Editor(editor ...string) *Text {
	if len(editor) > 0 {
		t.editorCmd = editor[0]
	}
	if len(editor) > 1 {
		t.editorArgs = editor[1:]
	}
	return t
}

// EditorExtension specifies arguments to pass into the editor.
func (t *Text) EditorExtension(extension string) *Text {
	t.editorExtension = extension
	return t
}

// Error returns the error of the text field.
func (t *Text) Error() error { return t.err }

// Skip returns whether the textarea should be skipped or should be blocking.
func (*Text) Skip() bool { return false }

// Zoom returns whether the note should be zoomed.
func (*Text) Zoom() bool { return false }

// Focus focuses the text field.
func (t *Text) Focus() tea.Cmd {
	t.focused = true
	return t.textarea.Focus()
}

// Blur blurs the text field.
func (t *Text) Blur() tea.Cmd {
	t.focused = false
	t.accessor.Set(t.textarea.Value())
	t.textarea.Blur()
	t.err = t.validate(t.accessor.Get())
	return nil
}

// KeyBinds returns the help message for the text field.
func (t *Text) KeyBinds() []key.Binding {
	return []key.Binding{t.keymap.NewLine, t.keymap.Editor, t.keymap.Prev, t.keymap.Submit, t.keymap.Next}
}

type updateValueMsg []byte

// Init initializes the text field.
func (t *Text) Init() tea.Cmd {
	t.textarea.Blur()
	return nil
}

// Update updates the text field.
func (t *Text) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case updateValueMsg:
		t.textarea.SetValue(string(msg))
		t.textarea, cmd = t.textarea.Update(msg)
		cmds = append(cmds, cmd)
		t.accessor.Set(t.textarea.Value())
	case updateFieldMsg:
		var cmds []tea.Cmd
		if ok, hash := t.placeholder.shouldUpdate(); ok {
			t.placeholder.bindingsHash = hash
			if t.placeholder.loadFromCache() {
				t.textarea.Placeholder = t.placeholder.val
			} else {
				t.placeholder.loading = true
				cmds = append(cmds, func() tea.Msg {
					return updatePlaceholderMsg{id: t.id, placeholder: t.placeholder.fn(), hash: hash}
				})
			}
		}
		if ok, hash := t.title.shouldUpdate(); ok {
			t.title.bindingsHash = hash
			if !t.title.loadFromCache() {
				cmds = append(cmds, func() tea.Msg {
					return updateTitleMsg{id: t.id, title: t.title.fn(), hash: hash}
				})
			}
		}
		if ok, hash := t.description.shouldUpdate(); ok {
			t.description.bindingsHash = hash
			if !t.description.loadFromCache() {
				t.description.loading = true
				cmds = append(cmds, func() tea.Msg {
					return updateDescriptionMsg{id: t.id, description: t.description.fn(), hash: hash}
				})
			}
		}
		return t, tea.Batch(cmds...)
	case updatePlaceholderMsg:
		if t.id == msg.id && t.placeholder.bindingsHash == msg.hash {
			t.placeholder.update(msg.placeholder)
			t.textarea.Placeholder = msg.placeholder
		}
	case updateTitleMsg:
		if t.id == msg.id && t.title.bindingsHash == msg.hash {
			t.title.update(msg.title)
		}
	case updateDescriptionMsg:
		if t.id == msg.id && t.description.bindingsHash == msg.hash {
			t.description.update(msg.description)
		}
	case tea.KeyMsg:
		t.err = nil

		switch {
		case key.Matches(msg, t.keymap.Editor):
			ext := strings.TrimPrefix(t.editorExtension, ".")
			tmpFile, _ := os.CreateTemp(os.TempDir(), "*."+ext)
			cmd := exec.Command(t.editorCmd, append(t.editorArgs, tmpFile.Name())...) //nolint:gosec
			_ = os.WriteFile(tmpFile.Name(), []byte(t.textarea.Value()), 0o644)       //nolint:mnd,gosec
			cmds = append(cmds, tea.ExecProcess(cmd, func(error) tea.Msg {
				content, _ := os.ReadFile(tmpFile.Name())
				_ = os.Remove(tmpFile.Name())
				return updateValueMsg(content)
			}))
		case key.Matches(msg, t.keymap.Next, t.keymap.Submit):
			value := t.textarea.Value()
			t.err = t.validate(value)
			if t.err != nil {
				return t, nil
			}
			cmds = append(cmds, NextField)
		case key.Matches(msg, t.keymap.Prev):
			value := t.textarea.Value()
			t.err = t.validate(value)
			if t.err != nil {
				return t, nil
			}
			cmds = append(cmds, PrevField)
		}
	}

	t.textarea, cmd = t.textarea.Update(msg)
	cmds = append(cmds, cmd)
	t.accessor.Set(t.textarea.Value())

	return t, tea.Batch(cmds...)
}

func (t *Text) activeStyles() *FieldStyles {
	theme := t.theme
	if theme == nil {
		theme = ThemeCharm()
	}
	if t.focused {
		return &theme.Focused
	}
	return &theme.Blurred
}

func (t *Text) activeTextAreaStyles() *textarea.Style {
	if t.theme == nil {
		return &t.textarea.BlurredStyle
	}
	if t.focused {
		return &t.textarea.FocusedStyle
	}
	return &t.textarea.BlurredStyle
}

// View renders the text field.
func (t *Text) View() string {
	styles := t.activeStyles()
	textareaStyles := t.activeTextAreaStyles()

	// NB: since the method is on a pointer receiver these are being mutated.
	// Because this runs on every render this shouldn't matter in practice,
	// however.
	textareaStyles.Placeholder = styles.TextInput.Placeholder
	textareaStyles.Text = styles.TextInput.Text
	textareaStyles.Prompt = styles.TextInput.Prompt
	textareaStyles.CursorLine = styles.TextInput.Text
	t.textarea.Cursor.Style = styles.TextInput.Cursor
	t.textarea.Cursor.TextStyle = styles.TextInput.CursorText

	var sb strings.Builder
	if t.title.val != "" || t.title.fn != nil {
		sb.WriteString(styles.Title.Render(t.title.val))
		if t.err != nil {
			sb.WriteString(styles.ErrorIndicator.String())
		}
		sb.WriteString("\n")
	}
	if t.description.val != "" || t.description.fn != nil {
		sb.WriteString(styles.Description.Render(t.description.val))
		sb.WriteString("\n")
	}
	sb.WriteString(t.textarea.View())

	return styles.Base.Render(sb.String())
}

// Run runs the text field.
func (t *Text) Run() error {
	if t.accessible {
		return t.runAccessible()
	}
	return Run(t)
}

// runAccessible runs an accessible text field.
func (t *Text) runAccessible() error {
	styles := t.activeStyles()
	fmt.Println(styles.Title.Render(t.title.val))
	fmt.Println()
	t.accessor.Set(accessibility.PromptString("Input: ", func(input string) error {
		if err := t.validate(input); err != nil {
			// Handle the error from t.validate, return it
			return err
		}

		if len(input) > t.textarea.CharLimit {
			return fmt.Errorf("Input cannot exceed %d characters", t.textarea.CharLimit)
		}
		return nil
	}))
	fmt.Println()
	return nil
}

// WithTheme sets the theme on a text field.
func (t *Text) WithTheme(theme *Theme) Field {
	if t.theme != nil {
		return t
	}
	t.theme = theme
	return t
}

// WithKeyMap sets the keymap on a text field.
func (t *Text) WithKeyMap(k *KeyMap) Field {
	t.keymap = k.Text
	t.textarea.KeyMap.InsertNewline.SetKeys(t.keymap.NewLine.Keys()...)
	return t
}

// WithAccessible sets the accessible mode of the text field.
func (t *Text) WithAccessible(accessible bool) Field {
	t.accessible = accessible
	return t
}

// WithWidth sets the width of the text field.
func (t *Text) WithWidth(width int) Field {
	t.width = width
	t.textarea.SetWidth(width - t.activeStyles().Base.GetHorizontalFrameSize())
	return t
}

// WithHeight sets the height of the text field.
func (t *Text) WithHeight(height int) Field {
	adjust := 0
	if t.title.val != "" {
		adjust++
	}
	if t.description.val != "" {
		adjust++
	}
	t.textarea.SetHeight(height - t.activeStyles().Base.GetVerticalFrameSize() - adjust)
	return t
}

// WithPosition sets the position information of the text field.
func (t *Text) WithPosition(p FieldPosition) Field {
	t.keymap.Prev.SetEnabled(!p.IsFirst())
	t.keymap.Next.SetEnabled(!p.IsLast())
	t.keymap.Submit.SetEnabled(p.IsLast())
	return t
}

// GetKey returns the key of the field.
func (t *Text) GetKey() string { return t.key }

// GetValue returns the value of the field.
func (t *Text) GetValue() any {
	return t.accessor.Get()
}
