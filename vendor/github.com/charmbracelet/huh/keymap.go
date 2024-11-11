package huh

import "github.com/charmbracelet/bubbles/key"

// KeyMap is the keybindings to navigate the form.
type KeyMap struct {
	Quit key.Binding

	Confirm     ConfirmKeyMap
	FilePicker  FilePickerKeyMap
	Input       InputKeyMap
	MultiSelect MultiSelectKeyMap
	Note        NoteKeyMap
	Select      SelectKeyMap
	Text        TextKeyMap
}

// InputKeyMap is the keybindings for input fields.
type InputKeyMap struct {
	AcceptSuggestion key.Binding
	Next             key.Binding
	Prev             key.Binding
	Submit           key.Binding
}

// TextKeyMap is the keybindings for text fields.
type TextKeyMap struct {
	Next    key.Binding
	Prev    key.Binding
	NewLine key.Binding
	Editor  key.Binding
	Submit  key.Binding
}

// SelectKeyMap is the keybindings for select fields.
type SelectKeyMap struct {
	Next         key.Binding
	Prev         key.Binding
	Up           key.Binding
	Down         key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
	GotoTop      key.Binding
	GotoBottom   key.Binding
	Left         key.Binding
	Right        key.Binding
	Filter       key.Binding
	SetFilter    key.Binding
	ClearFilter  key.Binding
	Submit       key.Binding
}

// MultiSelectKeyMap is the keybindings for multi-select fields.
type MultiSelectKeyMap struct {
	Next         key.Binding
	Prev         key.Binding
	Up           key.Binding
	Down         key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
	GotoTop      key.Binding
	GotoBottom   key.Binding
	Toggle       key.Binding
	Filter       key.Binding
	SetFilter    key.Binding
	ClearFilter  key.Binding
	Submit       key.Binding
	SelectAll    key.Binding
	SelectNone   key.Binding
}

// FilePickerKey is the keybindings for filepicker fields.
type FilePickerKeyMap struct {
	Open     key.Binding
	Close    key.Binding
	GoToTop  key.Binding
	GoToLast key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Back     key.Binding
	Select   key.Binding
	Up       key.Binding
	Down     key.Binding
	Prev     key.Binding
	Next     key.Binding
	Submit   key.Binding
}

// NoteKeyMap is the keybindings for note fields.
type NoteKeyMap struct {
	Next   key.Binding
	Prev   key.Binding
	Submit key.Binding
}

// ConfirmKeyMap is the keybindings for confirm fields.
type ConfirmKeyMap struct {
	Next   key.Binding
	Prev   key.Binding
	Toggle key.Binding
	Submit key.Binding
	Accept key.Binding
	Reject key.Binding
}

// NewDefaultKeyMap returns a new default keymap.
func NewDefaultKeyMap() *KeyMap {
	return &KeyMap{
		Quit: key.NewBinding(key.WithKeys("ctrl+c")),
		Input: InputKeyMap{
			AcceptSuggestion: key.NewBinding(key.WithKeys("ctrl+e"), key.WithHelp("ctrl+e", "complete")),
			Prev:             key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "back")),
			Next:             key.NewBinding(key.WithKeys("enter", "tab"), key.WithHelp("enter", "next")),
			Submit:           key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),
		},
		FilePicker: FilePickerKeyMap{
			GoToTop:  key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "first"), key.WithDisabled()),
			GoToLast: key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "last"), key.WithDisabled()),
			PageUp:   key.NewBinding(key.WithKeys("K", "pgup"), key.WithHelp("pgup", "page up"), key.WithDisabled()),
			PageDown: key.NewBinding(key.WithKeys("J", "pgdown"), key.WithHelp("pgdown", "page down"), key.WithDisabled()),
			Back:     key.NewBinding(key.WithKeys("h", "backspace", "left", "esc"), key.WithHelp("h", "back"), key.WithDisabled()),
			Select:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select"), key.WithDisabled()),
			Up:       key.NewBinding(key.WithKeys("up", "k", "ctrl+k", "ctrl+p"), key.WithHelp("↑", "up"), key.WithDisabled()),
			Down:     key.NewBinding(key.WithKeys("down", "j", "ctrl+j", "ctrl+n"), key.WithHelp("↓", "down"), key.WithDisabled()),

			Open:   key.NewBinding(key.WithKeys("l", "right", "enter"), key.WithHelp("enter", "open")),
			Close:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close"), key.WithDisabled()),
			Prev:   key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "back")),
			Next:   key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next")),
			Submit: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),
		},
		Text: TextKeyMap{
			Prev:    key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "back")),
			Next:    key.NewBinding(key.WithKeys("tab", "enter"), key.WithHelp("enter", "next")),
			Submit:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),
			NewLine: key.NewBinding(key.WithKeys("alt+enter", "ctrl+j"), key.WithHelp("alt+enter / ctrl+j", "new line")),
			Editor:  key.NewBinding(key.WithKeys("ctrl+e"), key.WithHelp("ctrl+e", "open editor")),
		},
		Select: SelectKeyMap{
			Prev:         key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "back")),
			Next:         key.NewBinding(key.WithKeys("enter", "tab"), key.WithHelp("enter", "select")),
			Submit:       key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),
			Up:           key.NewBinding(key.WithKeys("up", "k", "ctrl+k", "ctrl+p"), key.WithHelp("↑", "up")),
			Down:         key.NewBinding(key.WithKeys("down", "j", "ctrl+j", "ctrl+n"), key.WithHelp("↓", "down")),
			Left:         key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("←", "left"), key.WithDisabled()),
			Right:        key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("→", "right"), key.WithDisabled()),
			Filter:       key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
			SetFilter:    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "set filter"), key.WithDisabled()),
			ClearFilter:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "clear filter"), key.WithDisabled()),
			HalfPageUp:   key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("ctrl+u", "½ page up")),
			HalfPageDown: key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "½ page down")),
			GotoTop:      key.NewBinding(key.WithKeys("home", "g"), key.WithHelp("g/home", "go to start")),
			GotoBottom:   key.NewBinding(key.WithKeys("end", "G"), key.WithHelp("G/end", "go to end")),
		},
		MultiSelect: MultiSelectKeyMap{
			Prev:         key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "back")),
			Next:         key.NewBinding(key.WithKeys("enter", "tab"), key.WithHelp("enter", "confirm")),
			Submit:       key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),
			Toggle:       key.NewBinding(key.WithKeys(" ", "x"), key.WithHelp("x", "toggle")),
			Up:           key.NewBinding(key.WithKeys("up", "k", "ctrl+p"), key.WithHelp("↑", "up")),
			Down:         key.NewBinding(key.WithKeys("down", "j", "ctrl+n"), key.WithHelp("↓", "down")),
			Filter:       key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
			SetFilter:    key.NewBinding(key.WithKeys("enter", "esc"), key.WithHelp("esc", "set filter"), key.WithDisabled()),
			ClearFilter:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "clear filter"), key.WithDisabled()),
			HalfPageUp:   key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("ctrl+u", "½ page up")),
			HalfPageDown: key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "½ page down")),
			GotoTop:      key.NewBinding(key.WithKeys("home", "g"), key.WithHelp("g/home", "go to start")),
			GotoBottom:   key.NewBinding(key.WithKeys("end", "G"), key.WithHelp("G/end", "go to end")),
			SelectAll:    key.NewBinding(key.WithKeys("ctrl+a"), key.WithHelp("ctrl+a", "select all")),
			SelectNone:   key.NewBinding(key.WithKeys("ctrl+a"), key.WithHelp("ctrl+a", "select none"), key.WithDisabled()),
		},
		Note: NoteKeyMap{
			Prev:   key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "back")),
			Next:   key.NewBinding(key.WithKeys("enter", "tab"), key.WithHelp("enter", "next")),
			Submit: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),
		},
		Confirm: ConfirmKeyMap{
			Prev:   key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "back")),
			Next:   key.NewBinding(key.WithKeys("enter", "tab"), key.WithHelp("enter", "next")),
			Submit: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),
			Toggle: key.NewBinding(key.WithKeys("h", "l", "right", "left"), key.WithHelp("←/→", "toggle")),
			Accept: key.NewBinding(key.WithKeys("y", "Y"), key.WithHelp("y", "Yes")),
			Reject: key.NewBinding(key.WithKeys("n", "N"), key.WithHelp("n", "No")),
		},
	}
}
