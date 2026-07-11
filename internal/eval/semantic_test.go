package eval

import "testing"

func TestScoreContinuitySignalsTracksSignalNoiseAndGrounding(t *testing.T) {
	score := ScoreContinuitySignals(ContinuitySignalInput{
		ExpectedSignals:  []string{"workspace resolver", "candidate confirmation"},
		ForbiddenNoise:   []string{"unrelated chat", "mascot joke"},
		SourceEvidence:   []string{"workspace resolver", "candidate confirmation", "audit record"},
		Output:           "Use the workspace resolver, ask for candidate confirmation, and leave an audit record.",
		BaselineScore:    0.50,
		ContextMeshScore: 0.85,
	})

	if score.Continuation != 1 {
		t.Fatalf("expected full continuation, got %f", score.Continuation)
	}
	if score.Noise != 0 {
		t.Fatalf("expected zero noise, got %f", score.Noise)
	}
	if score.Groundedness != 1 {
		t.Fatalf("expected grounded output, got %f", score.Groundedness)
	}
	if score.RelativeImprovement != 0.35 {
		t.Fatalf("expected relative improvement 0.35, got %f", score.RelativeImprovement)
	}
}

func TestScoreContinuitySignalsPenalizesNoiseAndUnsupportedOutput(t *testing.T) {
	score := ScoreContinuitySignals(ContinuitySignalInput{
		ExpectedSignals: []string{"workspace resolver", "candidate confirmation"},
		ForbiddenNoise:  []string{"mascot joke", "slack bot"},
		SourceEvidence:  []string{"workspace resolver"},
		Output:          "Use the workspace resolver, add the mascot joke, and build a Slack bot.",
	})

	if score.Continuation != 0.5 {
		t.Fatalf("expected partial continuation, got %f", score.Continuation)
	}
	if score.Noise != 1 {
		t.Fatalf("expected full noise hit, got %f", score.Noise)
	}
	if score.Groundedness != 1.0/3.0 {
		t.Fatalf("expected groundedness penalty for unsupported output, got %f", score.Groundedness)
	}
}
