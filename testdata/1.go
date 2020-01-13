package testdata

var (
	b1 = true
	b2 = true
	b3 = true
)

func _() {
	if b1 { // complexity: 0
	}
}

func _() {
	if b1 { // complexity: 1
		if b2 { // +1
		}
	}
}

func _() {
	if b1 { // complexity: 6
		if b2 { // +1
			if b3 { // +2
			}
		}

		if b2 { // +1
			if b3 { // +2
			}
		}
	}
}

func _() { // complexity: 3
	if b1 {
		if b2 { // +1
		} else {
			if b3 { // +2
			}
		}
	}
}
