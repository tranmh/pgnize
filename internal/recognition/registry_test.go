package recognition

import "testing"

func TestRegistryResolveAndDefault(t *testing.T) {
	reg := NewRegistry()
	reg.Register("fake", "Fake", true, NewFake())
	reg.Register("ollama", "Ollama", true, NewOllama("http://x", "m"))
	reg.SetDefault("ollama")

	if reg.Default() != "ollama" {
		t.Fatalf("default = %q, want ollama", reg.Default())
	}

	// Empty key resolves to the default backend.
	rec, ok := reg.Resolve("")
	if !ok || rec.Name() != "ollama:m" {
		t.Fatalf("Resolve(\"\") = %v,%v want ollama:m", rec, ok)
	}

	rec, ok = reg.Resolve("fake")
	if !ok || rec.Name() != "fake" {
		t.Fatalf("Resolve(fake) = %v,%v want fake", rec, ok)
	}

	if _, ok := reg.Resolve("nope"); ok {
		t.Fatalf("Resolve(nope) should fail")
	}
}

func TestRegistryAvailableAndAdvertised(t *testing.T) {
	reg := NewRegistry()
	reg.Register("fake", "Fake", false, NewFake()) // resolvable but hidden
	reg.Register("gemini", "Gemini Flash", true, NewFake())
	reg.SetDefault("gemini")

	// Empty (use default) is always allowed.
	if !reg.Available("") {
		t.Fatal("empty key should be available")
	}
	if !reg.Available("gemini") {
		t.Fatal("gemini should be available")
	}
	if reg.Available("fake") {
		t.Fatal("hidden backend must not be selectable")
	}
	if reg.Available("missing") {
		t.Fatal("unknown backend must not be available")
	}

	adv := reg.Advertised()
	if len(adv) != 1 || adv[0].Key != "gemini" || !adv[0].Default {
		t.Fatalf("Advertised() = %+v, want only gemini flagged default", adv)
	}
}

func TestRegistryName(t *testing.T) {
	reg := NewRegistry()
	reg.Register("gemini", "Gemini", true, &Gemini{Model: "gemini-2.5-flash"})
	reg.SetDefault("gemini")
	if got := reg.Name(""); got != "gemini:gemini-2.5-flash" {
		t.Fatalf("Name(\"\") = %q", got)
	}
	if got := reg.Name("unknown"); got != "" {
		t.Fatalf("Name(unknown) = %q, want empty", got)
	}
}
