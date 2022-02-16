package writers

// CountSpaces returns the number of spaces from the begining of the line
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
