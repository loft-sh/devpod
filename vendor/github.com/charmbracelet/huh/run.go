package huh

// Run runs a single field by wrapping it within a group and a form.
func Run(field Field) error {
	group := NewGroup(field)
	form := NewForm(group).WithShowHelp(false)
	return form.Run()
}
