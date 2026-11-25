package main

import "testing"

func TestParsePorts(t *testing.T) {
	tests := []struct {
		spec    string
		want    []int
		wantErr bool
	}{
		{"25565", []int{25565}, false},
		{"1-3", []int{1, 2, 3}, false},
		{"1-2,4", []int{1, 2, 4}, false},
		{"3-1", nil, true},
		{"0", nil, true},
		{"65536", nil, true},
		{"abc", nil, true},
		{"1-", nil, true},
		{"", nil, true},
	}
	for _, test := range tests {
		portsFlag = test.spec
		got, err := parsePorts()
		if test.wantErr {
			if err == nil {
				t.Fatalf("%q: expected an error, got %v", test.spec, got)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%q: %v", test.spec, err)
		}
		if len(got) != len(test.want) {
			t.Fatalf("%q: got %v, want %v", test.spec, got, test.want)
		}
		for index := range got {
			if got[index] != test.want[index] {
				t.Fatalf("%q: got %v, want %v", test.spec, got, test.want)
			}
		}
	}
}
