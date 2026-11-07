//go:build features

package main

import "testing"

// TestFeatures runs the worker through the standard application lifecycle for
// the external Cucumber harness and fails if the application returns an error.
func TestFeatures(t *testing.T) {
	if err := app.Run(t.Context()); err != nil {
		t.Fatal(err.Error())
	}
}
