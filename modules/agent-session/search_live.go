package agentsession

import (
	"bufio"
	"os"
	"strings"
	"sync"
)

// Role filter values for search.
const (
	roleAll       = "all"
	roleUser      = "user"
	roleAssistant = "assistant"
)

// roleMatches reports whether a message role passes the filter.
func roleMatches(filter, role string) bool {
	return filter == "" || filter == roleAll || filter == role
}

// liveResult is the outcome of streaming an entire .jsonl file. Hits is the
// positive-term occurrence count in the (role-filtered) body; seen records
// which query terms appeared in that body.
type liveResult struct {
	Hits    int
	Snippet string
	seen    map[string]bool
}

// liveCache memoizes liveResult per (jsonl path, query+role) for the lifetime
// of the TUI so retyping the same query does not redo I/O.
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

// liveScan streams the whole .jsonl file (honoring the role filter): records
// which query terms appear in the body, counts positive-term occurrences, and
// captures a snippet centered on the first positive hit.
func liveScan(path string, pq parsedQuery, role string) liveResult {
	res := liveResult{seen: make(map[string]bool, len(pq.terms))}
	f, err := os.Open(path)
	if err != nil {
		return res
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 16*1024*1024)

	for scanner.Scan() {
		msgRole, text := parseJsonlMessage(scanner.Bytes())
		if msgRole == "" || text == "" || !roleMatches(role, msgRole) {
			continue
		}
		lower := strings.ToLower(text)
		for _, t := range pq.terms {
			if !res.seen[t] && strings.Contains(lower, t) {
				res.seen[t] = true
			}
		}
		for _, t := range pq.positive {
			n := strings.Count(lower, t)
			if n == 0 {
				continue
			}
			res.Hits += n
			if res.Snippet == "" {
				res.Snippet = windowAround(text, strings.Index(lower, t), 140)
			}
		}
	}
	return res
}

// liveRank scans every session's full .jsonl file in parallel and keeps those
// the boolean query matches. Body matching is complete (whole file, role
// filtered); an all-roles search also matches session metadata, so it stays a
// superset of the role-filtered searches. Results are cached per query+role.
func liveRank(idx *searchIndex, cache *liveCache, query, providerFilter, folder, role string, sort SortMode, workers int) []candidate {
	raw := strings.TrimSpace(query)
	pq := parseQuery(raw)
	if pq.empty() {
		return rankLocal(idx, "", providerFilter, folder, sort)
	}
	if workers <= 0 {
		workers = 16
	}
	cacheKey := raw + "|" + role
	roleAny := role == "" || role == roleAll

	type job struct{ s *indexedSession }
	type result struct {
		s    *indexedSession
		live liveResult
	}
	jobs := make(chan job)
	results := make(chan result)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				if r, ok := cache.get(j.s.JsonlPath, cacheKey); ok {
					results <- result{s: j.s, live: r}
					continue
				}
				r := liveScan(j.s.JsonlPath, pq, role)
				cache.put(j.s.JsonlPath, cacheKey, r)
				results <- result{s: j.s, live: r}
			}
		}()
	}
	go func() {
		for _, s := range idx.Sessions {
			if providerFilter != "" && providerFilter != "all" && s.Provider != providerFilter {
				continue
			}
			if !matchFolder(s.Path, folder) {
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
		s := r.s
		// Only all-roles search consults metadata; a role filter is body-only.
		meta := ""
		if roleAny {
			meta = strings.ToLower(s.Name + " " + s.Description + " " + s.Path + " " + s.Provider)
		}
		present := func(field, term string) bool {
			switch field {
			case "name":
				return strings.Contains(strings.ToLower(s.Name), term)
			case "desc", "description":
				return strings.Contains(strings.ToLower(s.Description), term)
			case "path", "folder", "dir":
				return strings.Contains(strings.ToLower(s.Path), term)
			case "provider":
				return strings.Contains(strings.ToLower(s.Provider), term)
			default:
				return r.live.seen[term] || (meta != "" && strings.Contains(meta, term))
			}
		}
		if !pq.evalMatch(present) {
			continue
		}
		metaHits := 0
		if meta != "" {
			for _, t := range pq.positive {
				metaHits += strings.Count(meta, t)
			}
		}
		out = append(out, candidate{
			Session: s,
			Score:   float64(r.live.Hits) + float64(metaHits)*1.5,
			Hits:    r.live.Hits + metaHits,
			Source:  "live",
			Snippet: r.live.Snippet,
		})
	}
	sortCandidates(out, sort)
	return out
}

// allMatchSnippets streams the selected session's .jsonl file and returns up
// to `max` windowed snippets (role-prefixed) around lines that contain any
// positive term. Used on demand when the user expands a result (tab).
func allMatchSnippets(path string, pq parsedQuery, role string, max int) []string {
	if len(pq.positive) == 0 || max <= 0 {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 16*1024*1024)

	var out []string
	for scanner.Scan() && len(out) < max {
		msgRole, text := parseJsonlMessage(scanner.Bytes())
		if msgRole == "" || text == "" || !roleMatches(role, msgRole) {
			continue
		}
		lower := strings.ToLower(text)
		first := -1
		for _, t := range pq.positive {
			if idx := strings.Index(lower, t); idx >= 0 && (first < 0 || idx < first) {
				first = idx
			}
		}
		if first < 0 {
			continue
		}
		out = append(out, msgRole+": "+windowAround(text, first, 160))
	}
	return out
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

// matchFolder reports whether a session path is inside the folder filter.
// Empty folder means "global" (match everything).
func matchFolder(sessionPath, folder string) bool {
	if folder == "" {
		return true
	}
	return sessionPath == folder || strings.HasPrefix(sessionPath, folder+"/")
}
