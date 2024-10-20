package sse_test

import (
	"reflect"
	"testing"
)

// assertEqual checks if two values are equal. If they are not, it logs using t.Fatalf()
func assertEqual[T any](t testing.TB, got, correct T) {
	t.Helper()
	if !reflect.DeepEqual(got, correct) {
		t.Fatalf("assertEqual: expected values to be equal, got:\n%+v\ncorrect:\n%+v", got, correct)
	}
}

// assertError checks if an error is not nil. If it's nil, it logs using t.Fatalf()
func assertError(t testing.TB, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("assertError: expected an error, got nil")
	}
}

// assertNoError checks if an error is nil. If it's not nil, it logs using t.Fatalf()
func assertNoError(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("assertNoError: expected no error, got %+v", err)
	}
}
