package httpapi

import "testing"

func TestLichessPGNEndpointAccepts(t *testing.T) {
	cases := []struct{ in, want string }{
		{"https://lichess.org/study/abc123", "https://lichess.org/api/study/abc123.pgn"},
		{"https://lichess.org/study/abc123/chap45", "https://lichess.org/api/study/abc123/chap45.pgn"},
		{"https://lichess.org/aBcD1234", "https://lichess.org/game/export/aBcD1234"},
		{"https://lichess.org/aBcD1234/white", "https://lichess.org/game/export/aBcD1234"},
	}
	for _, c := range cases {
		got, err := lichessPGNEndpoint(c.in)
		if err != nil {
			t.Errorf("%s: unexpected error %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s: got %s want %s", c.in, got, c.want)
		}
	}
}

func TestLichessPGNEndpointRejects(t *testing.T) {
	bad := []string{
		"http://lichess.org/study/abc",      // not https
		"https://evil.com/study/abc",        // wrong host
		"https://lichess.org.evil.com/abc",  // host-suffix trick
		"https://evil-lichess.org/abc",      // host-prefix trick
		"https://lichess.org/study/",        // missing study id
		"https://lichess.org/study/bad-id!", // non-alphanumeric id
		"http://127.0.0.1/abc",              // localhost / raw IP
		"https://localhost/abc",
		"ftp://lichess.org/abc",
		"https://lichess.org/", // no path
		"",
	}
	for _, in := range bad {
		if _, err := lichessPGNEndpoint(in); err == nil {
			t.Errorf("%q: expected rejection, got none", in)
		}
	}
}
