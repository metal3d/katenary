package writers

// IndentSize set the indentation size for yaml output.
var IndentSize = 2

// CountSpaces returns the number of spaces from the begining of the line.
func CountSpaces(line string) int {
	var spaces int
	for _, char := range line {
		if char == ' ' {
			spaces++
		} else {
			break
		}
	}
	return spaces
}
