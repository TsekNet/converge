package logging

import "testing"

func TestInit_Verbose(t *testing.T) {
	Init(true)
}

func TestInit_Quiet(t *testing.T) {
	Init(false)
}
