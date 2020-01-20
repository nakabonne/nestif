package testdata

func _() { // complexity: 3
	var b1, b2, b3 bool

	if b1 {
		if b2 { // +1
		} else {
			if b3 { // +2
			}
		}
	}
	/*
		if b1 {
			if b2 { // +1
			} else if b3 {
				if b4 { // +2
				}
			}
		}
	*/
}
