package chesskit

import "testing"

func TestNormalizeResult(t *testing.T) {
	cases := []struct {
		in   string
		want Result
	}{
		{"1-0", ResultWhiteWin},
		{"1:0", ResultWhiteWin},
		{"1 - 0", ResultWhiteWin},
		{"White", ResultWhiteWin},
		{"0-1", ResultBlackWin},
		{"0:1", ResultBlackWin},
		{"black wins", ResultBlackWin},
		{"1/2-1/2", ResultDraw},
		{"1/2", ResultDraw},
		{"½-½", ResultDraw},
		{"½", ResultDraw},
		{"remis", ResultDraw},
		{"Remis", ResultDraw},
		{"draw", ResultDraw},
		{"DRAW", ResultDraw},
		{"", ResultOngoing},
		{"*", ResultOngoing},
		{"something weird", ResultOngoing},
		{"unknown", ResultOngoing},
		{"  1-0  ", ResultWhiteWin},
	}
	for _, c := range cases {
		if got := NormalizeResult(c.in); got != c.want {
			t.Errorf("NormalizeResult(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
