package runner

type baselineSpec struct {
	ID      BaselineID
	Context string
}

func baselines(opts EvaluationOptions) []baselineSpec {
	return []baselineSpec{
		{ID: BaselineNoContext},
		{ID: BaselineStaleContext, Context: opts.StaleContext},
		{ID: BaselinePlainSummary, Context: opts.PlainSummary},
		{ID: BaselineContextMeshPacket, Context: opts.ContextPacket},
	}
}
