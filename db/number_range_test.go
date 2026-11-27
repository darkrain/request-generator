package db

import "testing"

func TestNumberRangeBounds(t *testing.T) {
	cases := []struct {
		in     string
		lo, hi string
		ok     bool
	}{
		{"18-20", "18", "20", true},
		{"18-", "18", "", true},
		{"-10--5", "-10", "-5", true},
		{"-10-5", "-10", "5", true},
		{"-5-", "-5", "", true},
		{"1.5-2.5", "1.5", "2.5", true},
		{"..20", "", "20", true},
		{"-50..-10", "-50", "-10", true},
		{"5..", "5", "", true},
		{" 3 - 7 ", "3", "7", true},
		// a lone number is equality, not a range
		{"42", "", "", false},
		{"-42", "", "", false},
		{"-1.5", "", "", false},
		{"1e-5", "", "", false},
		// not numbers
		{"abc", "", "", false},
		{"a-b", "", "", false},
		{"-", "", "", false},
		{"..", "", "", false},
		{"2026-01-01..2026-02-01", "", "", false},
	}
	for _, c := range cases {
		lo, hi, ok := numberRangeBounds(c.in)
		if ok != c.ok || lo != c.lo || hi != c.hi {
			t.Errorf("numberRangeBounds(%q) = %q, %q, %v; want %q, %q, %v", c.in, lo, hi, ok, c.lo, c.hi, c.ok)
		}
	}
}
