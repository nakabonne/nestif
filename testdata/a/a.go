package a

func _() {
	var b1, b2 bool

	if b1 { // complexity: 0
	}
	if b1 { // complexity: 1
		if b2 { // +1
		}
	}
}
