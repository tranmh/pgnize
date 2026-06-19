package recognition

// Registry maps a selectable backend key (e.g. "ollama", "gemini") to a concrete
// Recognizer. Recognition is pluggable and the backend is chosen per request, so the
// HTTP layer validates the requested key against the advertised set and the worker
// resolves the stored key back to a Recognizer at claim time.
type Registry struct {
	order []string
	byKey map[string]*backendEntry
	def   string
}

type backendEntry struct {
	key       string
	label     string
	advertise bool
	rec       Recognizer
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{byKey: map[string]*backendEntry{}}
}

// Register adds (or replaces) a backend. advertise controls whether it is offered to
// clients via Advertised/Available; non-advertised backends are still Resolve-able so a
// previously enqueued job can always be processed. The first Register sets the default.
func (r *Registry) Register(key, label string, advertise bool, rec Recognizer) {
	if _, ok := r.byKey[key]; !ok {
		r.order = append(r.order, key)
	}
	r.byKey[key] = &backendEntry{key: key, label: label, advertise: advertise, rec: rec}
	if r.def == "" {
		r.def = key
	}
}

// SetDefault selects the backend used when a request omits a backend key.
func (r *Registry) SetDefault(key string) { r.def = key }

// Default returns the default backend key.
func (r *Registry) Default() string { return r.def }

// Resolve returns the recognizer for key; an empty key resolves to the default.
func (r *Registry) Resolve(key string) (Recognizer, bool) {
	if key == "" {
		key = r.def
	}
	e, ok := r.byKey[key]
	if !ok {
		return nil, false
	}
	return e.rec, true
}

// Name returns the resolved recognizer Name() for key (empty → default), or "" if the
// key is unknown. Used to stamp recognition_jobs.recognizer_name at enqueue time.
func (r *Registry) Name(key string) string {
	rec, ok := r.Resolve(key)
	if !ok {
		return ""
	}
	return rec.Name()
}

// Available reports whether key is a selectable (advertised) backend. The empty key —
// meaning "use the server default" — is always allowed.
func (r *Registry) Available(key string) bool {
	if key == "" {
		return true
	}
	e, ok := r.byKey[key]
	return ok && e.advertise
}

// BackendInfo is the public description of a selectable backend.
type BackendInfo struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Default bool   `json:"default"`
}

// Advertised lists selectable backends in registration order.
func (r *Registry) Advertised() []BackendInfo {
	out := make([]BackendInfo, 0, len(r.order))
	for _, k := range r.order {
		e := r.byKey[k]
		if e.advertise {
			out = append(out, BackendInfo{Key: e.key, Label: e.label, Default: e.key == r.def})
		}
	}
	return out
}
