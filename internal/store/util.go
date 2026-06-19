package store

import (
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5/pgconn"
)

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func itoa(n int) string { return strconv.Itoa(n) }
