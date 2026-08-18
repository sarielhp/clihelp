package clihelp

import "testing"

func TestArgsValidators(t *testing.T) {
	t.Run("NoArgs", func(t *testing.T) {
		if err := NoArgs([]string{}); err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		if err := NoArgs([]string{"foo"}); err == nil {
			t.Errorf("expected error with args, got nil")
		}
	})

	t.Run("ExactArgs", func(t *testing.T) {
		val := ExactArgs(2)
		if err := val([]string{"a", "b"}); err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		if err := val([]string{"a"}); err == nil {
			t.Errorf("expected error for 1 arg, got nil")
		}
		if err := val([]string{"a", "b", "c"}); err == nil {
			t.Errorf("expected error for 3 args, got nil")
		}
	})

	t.Run("MinimumNArgs", func(t *testing.T) {
		val := MinimumNArgs(2)
		if err := val([]string{"a", "b"}); err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		if err := val([]string{"a", "b", "c"}); err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		if err := val([]string{"a"}); err == nil {
			t.Errorf("expected error for 1 arg, got nil")
		}
	})

	t.Run("MaximumNArgs", func(t *testing.T) {
		val := MaximumNArgs(2)
		if err := val([]string{"a"}); err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		if err := val([]string{"a", "b"}); err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		if err := val([]string{"a", "b", "c"}); err == nil {
			t.Errorf("expected error for 3 args, got nil")
		}
	})

	t.Run("RangeArgs", func(t *testing.T) {
		val := RangeArgs(1, 3)
		if err := val([]string{}); err == nil {
			t.Errorf("expected error for 0 args, got nil")
		}
		if err := val([]string{"a"}); err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		if err := val([]string{"a", "b", "c"}); err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		if err := val([]string{"a", "b", "c", "d"}); err == nil {
			t.Errorf("expected error for 4 args, got nil")
		}
	})
}
