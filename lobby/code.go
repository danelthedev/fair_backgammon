package lobby

import "math/rand"

const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no I/O/0/1

func genCode() string {
	b := make([]byte, 4)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}
