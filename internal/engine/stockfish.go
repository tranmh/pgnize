package engine

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"
)

// PoolOpts configures a Stockfish pool. Zero values fall back to sensible defaults.
type PoolOpts struct {
	Instances     int // number of UCI processes (parallel searches); default 2
	Threads       int // UCI "Threads" per instance; default 1
	HashMB        int // UCI "Hash" per instance in MB; default 64
	MoveTimeMs    int // default per-search movetime; default 300
	MaxMoveTimeMs int // upper clamp a caller may request; default 2000
	Version       string
}

// Stockfish is an Engine backed by a pool of long-lived UCI subprocesses. Each process
// runs at most one search at a time (UCI is single-search/stateful), so the pool is a
// channel of "slots": a nil slot means "spawn on demand", a non-nil slot is a ready proc.
type Stockfish struct {
	path                                            string
	threads, hashMB, moveTimeMs, maxMoveTimeMs      int
	version                                         string

	idle chan *proc

	mu   sync.Mutex
	live map[*proc]struct{} // every spawned, not-yet-killed proc (for Close)
}

var _ Engine = (*Stockfish)(nil)

const (
	handshakeTimeout = 10 * time.Second
	watchdogGrace    = 2 * time.Second // grace beyond movetime before a search is deemed hung
)

// NewStockfish starts a pool and verifies the binary speaks UCI by completing one
// handshake. It returns an error if the binary is missing or unresponsive, so callers can
// fall back to the Fake engine.
func NewStockfish(path string, opts PoolOpts) (*Stockfish, error) {
	if opts.Instances <= 0 {
		opts.Instances = 2
	}
	if opts.Threads <= 0 {
		opts.Threads = 1
	}
	if opts.HashMB <= 0 {
		opts.HashMB = 64
	}
	if opts.MoveTimeMs <= 0 {
		opts.MoveTimeMs = 300
	}
	if opts.MaxMoveTimeMs <= 0 {
		opts.MaxMoveTimeMs = 2000
	}
	s := &Stockfish{
		path:          path,
		threads:       opts.Threads,
		hashMB:        opts.HashMB,
		moveTimeMs:    opts.MoveTimeMs,
		maxMoveTimeMs: opts.MaxMoveTimeMs,
		version:       opts.Version,
		idle:          make(chan *proc, opts.Instances),
		live:          map[*proc]struct{}{},
	}
	// Spawn one eagerly to fail fast on a bad binary; the rest are lazy nil slots.
	first, err := s.spawn()
	if err != nil {
		return nil, err
	}
	s.idle <- first
	for i := 1; i < opts.Instances; i++ {
		s.idle <- nil
	}
	return s, nil
}

func (s *Stockfish) Name() string {
	if s.version != "" {
		return "stockfish:" + s.version
	}
	return "stockfish"
}

// Analyze checks out a slot, runs one search, and returns the slot to the pool. A search
// error poisons the proc: it is killed and the slot returns empty (a fresh proc is spawned
// on the next checkout), so the pool never serves a wedged process.
func (s *Stockfish) Analyze(ctx context.Context, fen string, opts Options) (Analysis, error) {
	var slot *proc
	select {
	case slot = <-s.idle:
	case <-ctx.Done():
		return Analysis{}, ctx.Err()
	}

	p := slot
	if p == nil {
		var err error
		p, err = s.spawn()
		if err != nil {
			s.idle <- nil // keep the slot; retry spawning next time
			return Analysis{}, err
		}
	}

	a, err := s.search(ctx, p, fen, opts)
	if err != nil {
		s.kill(p)
		s.idle <- nil
		return Analysis{}, err
	}
	s.idle <- p
	return a, nil
}

func (s *Stockfish) search(ctx context.Context, p *proc, fen string, opts Options) (Analysis, error) {
	movetime := opts.MoveTimeMs
	if movetime <= 0 {
		movetime = s.moveTimeMs
	}
	if movetime > s.maxMoveTimeMs {
		movetime = s.maxMoveTimeMs
	}
	multipv := opts.MultiPV
	if multipv <= 0 {
		multipv = 1
	}

	// Fresh search context (clears TT/heuristics) so analyses are independent.
	if err := p.send("ucinewgame"); err != nil {
		return Analysis{}, err
	}
	if err := p.send(fmt.Sprintf("setoption name MultiPV value %d", multipv)); err != nil {
		return Analysis{}, err
	}
	if err := p.send("position fen " + fen); err != nil {
		return Analysis{}, err
	}
	if err := p.readyOK(handshakeTimeout); err != nil {
		return Analysis{}, err
	}

	var goCmd string
	if opts.Depth > 0 {
		goCmd = fmt.Sprintf("go depth %d", opts.Depth)
	} else {
		goCmd = fmt.Sprintf("go movetime %d", movetime)
	}
	if err := p.send(goCmd); err != nil {
		return Analysis{}, err
	}

	lines := map[int]Line{}
	watchdog := time.NewTimer(time.Duration(movetime)*time.Millisecond + watchdogGrace + handshakeTimeout)
	defer watchdog.Stop()

	for {
		select {
		case <-ctx.Done():
			return Analysis{}, s.stopOrPoison(p, ctx.Err())
		case <-watchdog.C:
			return Analysis{}, s.stopOrPoison(p, fmt.Errorf("engine search timed out"))
		case line, ok := <-p.out:
			if !ok {
				return Analysis{}, fmt.Errorf("engine process exited during search")
			}
			if _, isBest := parseBestMove(line); isBest {
				return assemble(fen, lines), nil
			}
			if mpv, l, has := parseInfo(line); has {
				lines[mpv] = l
			}
		}
	}
}

// stopOrPoison sends "stop" and waits briefly for the bestmove so the proc can be reused;
// if it does not arrive in the grace window the proc is left to be killed by the caller.
// It always returns cause (the original cancellation/timeout error).
func (s *Stockfish) stopOrPoison(p *proc, cause error) error {
	_ = p.send("stop")
	grace := time.NewTimer(watchdogGrace)
	defer grace.Stop()
	for {
		select {
		case <-grace.C:
			return cause
		case line, ok := <-p.out:
			if !ok {
				return cause
			}
			if _, isBest := parseBestMove(line); isBest {
				return cause // drained cleanly; caller still treats the search as failed
			}
		}
	}
}

// assemble orders the collected per-MultiPV lines best-first by their index.
func assemble(fen string, lines map[int]Line) Analysis {
	idx := make([]int, 0, len(lines))
	for k := range lines {
		idx = append(idx, k)
	}
	sort.Ints(idx)
	out := make([]Line, 0, len(idx))
	for _, k := range idx {
		out = append(out, lines[k])
	}
	return Analysis{FEN: fen, Lines: out}
}

// Close terminates every live process.
func (s *Stockfish) Close() error {
	s.mu.Lock()
	procs := make([]*proc, 0, len(s.live))
	for p := range s.live {
		procs = append(procs, p)
	}
	s.mu.Unlock()
	for _, p := range procs {
		s.kill(p)
	}
	return nil
}

// ---- process ----

type proc struct {
	cmd   *exec.Cmd
	stdin *bufio.Writer
	out   chan string // every stdout line; closed when the reader goroutine exits
}

func (s *Stockfish) spawn() (*proc, error) {
	cmd := exec.Command(s.path)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start engine %q: %w", s.path, err)
	}
	p := &proc{
		cmd:   cmd,
		stdin: bufio.NewWriter(stdin),
		out:   make(chan string, 256),
	}
	go func() {
		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for sc.Scan() {
			p.out <- sc.Text()
		}
		close(p.out)
	}()

	if err := p.send("uci"); err != nil {
		_ = killCmd(cmd)
		return nil, err
	}
	if err := p.expect("uciok", handshakeTimeout); err != nil {
		_ = killCmd(cmd)
		return nil, err
	}
	_ = p.send(fmt.Sprintf("setoption name Threads value %d", s.threads))
	_ = p.send(fmt.Sprintf("setoption name Hash value %d", s.hashMB))
	if err := p.readyOK(handshakeTimeout); err != nil {
		_ = killCmd(cmd)
		return nil, err
	}

	s.mu.Lock()
	s.live[p] = struct{}{}
	s.mu.Unlock()
	return p, nil
}

func (s *Stockfish) kill(p *proc) {
	s.mu.Lock()
	_, tracked := s.live[p]
	delete(s.live, p)
	s.mu.Unlock()
	if tracked {
		_ = killCmd(p.cmd)
	}
}

func killCmd(cmd *exec.Cmd) error {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	return cmd.Wait()
}

func (p *proc) send(cmd string) error {
	if _, err := p.stdin.WriteString(cmd + "\n"); err != nil {
		return err
	}
	return p.stdin.Flush()
}

// expect reads lines until one has the given prefix, or the timeout elapses.
func (p *proc) expect(prefix string, timeout time.Duration) error {
	t := time.NewTimer(timeout)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			return fmt.Errorf("engine: timed out waiting for %q", prefix)
		case line, ok := <-p.out:
			if !ok {
				return fmt.Errorf("engine: process exited waiting for %q", prefix)
			}
			if strings.HasPrefix(line, prefix) {
				return nil
			}
		}
	}
}

// readyOK issues "isready" and waits for "readyok".
func (p *proc) readyOK(timeout time.Duration) error {
	if err := p.send("isready"); err != nil {
		return err
	}
	return p.expect("readyok", timeout)
}
