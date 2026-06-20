package jobs

import "testing"

func TestSafeRawJSON(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"valid object", `{"a":1}`, `{"a":1}`},
		{"valid array", `[{"white":"e4"}]`, `[{"white":"e4"}]`},
		{"empty string", "", "{}"},
		// A num_predict cap truncates the model output mid-JSON: invalid -> "{}"
		// so the result_raw_json::jsonb cast cannot fail with SQLSTATE 22P02.
		{"truncated object", `{"moves":[{"white":"e4","black":`, "{}"},
		{"whitespace only", "   ", "{}"},
		{"garbage", "not json", "{}"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := safeRawJSON(c.in); got != c.want {
				t.Errorf("safeRawJSON(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
