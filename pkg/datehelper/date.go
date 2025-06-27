package datehelper

import "time"

func Yesterday() int64 {
	return time.Now().AddDate(0, 0, -1).Unix()
}

func Tomorrow() int64 {
	return time.Now().AddDate(0, 0, 1).Unix()
}

func Today() int64 {
	return time.Now().Unix()
}
