package template

import "testing"

func TestInit(t *testing.T) {
	err := Init("/tmp/vbox/env")
	if err != nil {
		t.Fatal(err)
	}
}
