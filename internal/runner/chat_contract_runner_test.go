package runner

import (
	"strings"
	"testing"
)

func TestBuildChatContractPromptIncludesExpectedSections(t *testing.T) {
	req := ChatContractRequest{
		ThreadID:       "thread-1",
		SystemFrame:    "You are a chat assistant.",
		History:        []string{"User: hello", "Assistant: hi"},
		ContinuityView: "Current matter: housing search in Hangzhou",
		CurrentTurn:    "What about budget limits?",
	}

	got := BuildChatContractPrompt(req)
	want := strings.Join([]string{
		"History:\nUser: hello\nAssistant: hi",
		"Continuity:\nCurrent matter: housing search in Hangzhou",
		"Current turn:\nWhat about budget limits?",
	}, "\n\n")
	if got != want {
		t.Fatalf("unexpected prompt\nwant:\n%s\n\ngot:\n%s", want, got)
	}
	if strings.Contains(got, "System:\n") {
		t.Fatalf("prompt should not inline system frame, got %q", got)
	}
}

func TestBuildChatContractPromptOmitsEmptyOptionalSections(t *testing.T) {
	got := BuildChatContractPrompt(ChatContractRequest{
		ThreadID:    "thread-2",
		CurrentTurn: "What next?",
	})

	want := "Current turn:\nWhat next?"
	if got != want {
		t.Fatalf("unexpected prompt\nwant:\n%s\n\ngot:\n%s", want, got)
	}
	if strings.Contains(got, "History:\n") {
		t.Fatalf("prompt should omit history section when empty, got %q", got)
	}
	if strings.Contains(got, "Continuity:\n") {
		t.Fatalf("prompt should omit continuity section when empty, got %q", got)
	}
}
