package generator

import (
	"fmt"
	"math/rand"
	"time"
)

// color generation
func GenerateDarkColor() string {
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)

	red := r.Intn(128)   // Red component
	green := r.Intn(128) // Green component
	blue := r.Intn(128)  // Blue component

	return fmt.Sprintf("#%02x%02x%02x", red, green, blue)
}
