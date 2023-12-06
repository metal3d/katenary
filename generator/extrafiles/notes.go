package extrafiles

import _ "embed"

//go:embed notes.tpl
var notesTemplate string

// NoteTXTFile returns the content of the note.txt file.
func NotesFile() string {
	return notesTemplate
}
