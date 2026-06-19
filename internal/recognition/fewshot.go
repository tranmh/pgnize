package recognition

// SelectFewShot returns up to max of the most recent examples (caller passes newest-first).
// Few-shot context is pgnize's v1 "learning": a user's past corrected sheets steer the model.
func SelectFewShot(examples []Example, max int) []Example {
	if max <= 0 || len(examples) == 0 {
		return nil
	}
	if len(examples) > max {
		return examples[:max]
	}
	return examples
}
