package store

import "context"

// CreateFeedback records a before/after correction pair for future fine-tuning.
func (s *Store) CreateFeedback(ctx context.Context, uploadID *string, gameID, userID, recognizerName, beforeJSON, afterJSON string, editDistance int) error {
	_, err := s.Pool.Exec(ctx,
		`INSERT INTO feedback_corrections
		   (upload_id, game_id, user_id, recognizer_name, before_json, after_json, edit_distance)
		 VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7)`,
		uploadID, gameID, userID, recognizerName, beforeJSON, afterJSON, editDistance)
	return err
}
