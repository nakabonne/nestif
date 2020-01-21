package testdata

func _() {
	var b1, b2, b3, b4 bool

	if b1 { // complexity: 4
		if b2 { // +1
		} else { // +1
			if b3 { // +2
			}
		}
	}

	if b1 { // complexity: 4
		if b2 { // +1
		} else if b3 { // +1
			if b4 { // +2
			}
		}
	}

}
