package agentsession

import (
	"bufio"
	"os"
	"strings"
	"sync"
)

// liveResult is the outcome of streaming an entire .jsonl file looking for
// query matches. The hit count is exact; the snippet is a window centered
// on the first hit.
type liveResult struct {
	Hits    int
	Snippet string
}

// liveCache memoizes liveResult per (jsonl path, query) for the lifetime of
// the TUI process so retyping the same query — or refining it without
// changing the early prefix — does not redo I/O.
type liveCache struct {
	mu sync.RWMutex
	m  map[string]liveResult
}

func newLiveCache() *liveCache { return &liveCache{m: map[string]liveResult{}} }

func (c *liveCache) key(path, q string) string { return q + "\x00" + path }

func (c *liveCache) get(path, q string) (liveResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	r, ok := c.m[c.key(path, q)]
	return r, ok
}

func (c *liveCache) put(path, q string, r liveResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[c.key(path, q)] = r
}

// liveScan streams the .jsonl file and returns the exact number of times
// `query` (case-insensitive) appears anywhere in user/assistant message
// content, plus a short snippet centered on the first hit. Stops reading
// once the snippet is captured; counting continues to the end (cheap).
func liveScan(path, query string) liveResult {
	if query == "" {
		return liveResult{}
	}
	q := strings.ToLower(query)
	f, err := os.Open(path)
	if err != nil {
		return liveResult{}
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 16*1024*1024)

	var (
		hits    int
		snippet string
	)
	for scanner.Scan() {
		raw := scanner.Bytes()
		role, text := parseJsonlMessage(raw)
		if role == "" || text == "" {
			continue
		}
		lower := strings.ToLower(text)
		if !strings.Contains(lower, q) {
			continue
		}
		hits += strings.Count(lower, q)
		if snippet == "" {
			idx := strings.Index(lower, q)
			snippet = windowAround(text, idx, 140)
		}
	}
	return liveResult{Hits: hits, Snippet: snippet}
}

func windowAround(s string, hit, width int) string {
	if hit < 0 {
		return truncateText(s, width)
	}
	start := hit - width/3
	if start < 0 {
		start = 0
	}
	end := start + width
	if end > len(s) {
		end = len(s)
		start = end - width
		if start < 0 {
			start = 0
		}
	}
	out := strings.ReplaceAll(s[start:end], "\n", " ")
	if start > 0 {
		out = "…" + out
	}
	if end < len(s) {
		out += "…"
	}
	return out
}

// liveRank is the live-grep counterpart to rankLocal. It scans every
// indexed session's .jsonl file in parallel, returns candidates with at
// least one hit, and uses the per-file cache to avoid re-reading on
// repeated queries (e.g. as the user keeps typing).
//
// providerFilter behaves the same as rankLocal. Sort comes from the same
// SortMode set.
func liveRank(idx *searchIndex, cache *liveCache, query, providerFilter string, sort SortMode, workers int) []candidate {
	q := strings.TrimSpace(query)
	if q == "" {
		// No query → fall back to the cheap metadata ranker so the user
		// sees a sensible default list without paying for any I/O.
		return rankLocal(idx, "", providerFilter, sort)
	}
	if workers <= 0 {
		workers = 16
	}

	type job struct{ s *indexedSession }
	type result struct {
		s     *indexedSession
		live  liveResult
		score float64
	}

	jobs := make(chan job)
	results := make(chan result)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				if r, ok := cache.get(j.s.JsonlPath, q); ok {
					results <- result{s: j.s, live: r}
					continue
				}
				r := liveScan(j.s.JsonlPath, q)
				cache.put(j.s.JsonlPath, q, r)
				results <- result{s: j.s, live: r}
			}
		}()
	}

	go func() {
		for _, s := range idx.Sessions {
			if providerFilter != "" && providerFilter != "all" && s.Provider != providerFilter {
				continue
			}
			jobs <- job{s: s}
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	var out []candidate
	for r := range results {
		// Combine body hits with cheap metadata hits so name/path matches
		// still rank high even when body has nothing.
		_, metaHits := scoreSession(r.s, strings.ToLower(q))
		// metaHits double-counts body bytes from the cached excerpt, but
		// that excerpt is now small / disposable; what matters is that
		// metadata-only matches survive.
		bodyHits := r.live.Hits
		total := metaHits + bodyHits
		if total == 0 {
			continue
		}
		score := float64(bodyHits) + float64(metaHits)*1.5
		out = append(out, candidate{
			Session: r.s,
			Score:   score,
			Hits:    total,
			Source:  "live",
			Snippet: r.live.Snippet,
		})
	}
	sortCandidates(out, sort)
	return out
}
