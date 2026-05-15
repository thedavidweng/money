package prompt

import "testing"

func TestFakeSelectReturnsQueuedChoice(t *testing.T) {
	fake := NewFake("plaid")
	got, err := fake.Select("Choose provider", []Choice{{Label: "Plaid", Value: "plaid"}})
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if got != "plaid" {
		t.Fatalf("choice = %q", got)
	}
}

func TestFakeSelectRejectsUnqueuedChoice(t *testing.T) {
	fake := NewFake()
	_, err := fake.Select("Choose provider", []Choice{{Label: "Plaid", Value: "plaid"}})
	if err == nil {
		t.Fatal("Select succeeded without queued choice")
	}
}
