package testdata

var (
	b1 = true
	b2 = true
	b3 = true
)

func _() {
	if b1 {
	}
}

func _() {
	if b1 {
		if b2 {
		}
	}
}

func _() {
	if b1 {
		if b2 {
			if b3 {
			}
		}

		if b2 {
			if b3 {
			}
		}
	}
}
