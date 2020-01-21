package testdata

func _() {
	var b1, b2, b3 bool

	if b1 { // complexity: 1
		if b2 { // +1
		}
	}

	if b1 { // complexity: 1
		if b2 { // +1
		}
	}

	if b1 { // complexity: 3
		if b2 { // +1
			if b3 { // +2
			}
		}
	}
}
