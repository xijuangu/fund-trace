// Package quotes orchestrates real-time fund quote fetching and validation.
package quotes

import (
	"log/slog"
	"sync"
	"time"

	"fund-trace/internal/fetcher"
	"fund-trace/internal/model"
	"fund-trace/internal/store"
)

const (
	// metaRetryInterval keeps a permanently failing fund from triggering a
	// benchmark lookup on every refresh cycle.
	metaRetryInterval = 10 * time.Minute
	metaFetchWorkers  = 4
)

// Service fetches real-time fund quotes and reconciles them with each fund's
// benchmark index. Benchmark metadata is cached in the database, so the extra
// lookups only happen once per fund.
type Service struct {
	client    *fetcher.Client
	store     *store.Store
	tolerance float64

	mu    sync.Mutex
	metas map[string]model.FundMeta
	tried map[string]time.Time
}

// New builds a Service. st may be nil, in which case metadata is kept in memory
// only.
func New(client *fetcher.Client, st *store.Store) *Service {
	s := &Service{
		client:    client,
		store:     st,
		tolerance: fetcher.DefaultAnchorTolerance,
		metas:     make(map[string]model.FundMeta),
		tried:     make(map[string]time.Time),
	}
	if st != nil {
		if metas, err := st.FundMetas(); err == nil {
			// Rows with a zero weight have never been fetched; keep them out
			// of the cache so they are looked up on first use.
			s.metas = make(map[string]model.FundMeta, len(metas))
			for code, meta := range metas {
				if meta.IndexWeight > 0 {
					s.metas[code] = meta
				}
			}
		} else {
			slog.Warn("load fund meta cache failed", "error", err)
		}
	}
	return s
}

// FundQuotes returns real-time estimates for the given fund codes, with stale
// or inconsistent upstream values corrected by the benchmark index.
func (s *Service) FundQuotes(codes []string) map[string]*model.RealTimeFund {
	funds := s.client.FetchAllRealTime(codes)

	metas := s.metasFor(codes)
	changes := s.indexChanges(metas)
	if n := fetcher.ApplyIndexAnchor(funds, metas, changes, s.tolerance); n > 0 {
		slog.Info("index anchor corrected stale fund estimates", "funds", n)
	}
	return funds
}

// metasFor returns cached metadata for the requested codes, fetching and
// caching anything that is missing.
func (s *Service) metasFor(codes []string) map[string]model.FundMeta {
	metas := make(map[string]model.FundMeta, len(codes))

	s.mu.Lock()
	var missing []string
	for _, code := range codes {
		if meta, ok := s.metas[code]; ok {
			metas[code] = meta
			continue
		}
		if at, ok := s.tried[code]; ok && time.Since(at) < metaRetryInterval {
			continue
		}
		missing = append(missing, code)
	}
	s.mu.Unlock()

	if len(missing) > 0 {
		s.fetchMetas(missing)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, code := range codes {
		if meta, ok := s.metas[code]; ok {
			metas[code] = meta
		}
	}
	return metas
}

func (s *Service) fetchMetas(codes []string) {
	type result struct {
		code string
		meta *model.FundMeta
		err  error
	}

	results := make(chan result, len(codes))
	sem := make(chan struct{}, metaFetchWorkers)
	var wg sync.WaitGroup

	for _, code := range codes {
		wg.Add(1)
		go func(code string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			meta, err := s.client.FetchFundMeta(code)
			results <- result{code: code, meta: meta, err: err}
		}(code)
	}
	wg.Wait()
	close(results)

	s.mu.Lock()
	defer s.mu.Unlock()
	for r := range results {
		if r.err != nil {
			slog.Warn("fetch fund meta failed", "code", r.code, "error", r.err)
			s.tried[r.code] = time.Now()
			continue
		}
		s.metas[r.code] = *r.meta
		delete(s.tried, r.code)
		if s.store != nil {
			if err := s.store.SaveFundMeta(*r.meta); err != nil {
				slog.Warn("cache fund meta failed", "code", r.code, "error", err)
			}
		}
	}
}

// indexChanges fetches the intraday move of every benchmark index in use.
func (s *Service) indexChanges(metas map[string]model.FundMeta) map[string]float64 {
	secids := make([]string, 0, len(metas))
	for _, meta := range metas {
		if meta.DeriveOK && meta.IndexSecID != "" {
			secids = append(secids, meta.IndexSecID)
		}
	}
	if len(secids) == 0 {
		return nil
	}

	changes, err := s.client.FetchIndexChanges(secids)
	if err != nil {
		slog.Warn("fetch index quotes failed", "error", err)
		return nil
	}
	return changes
}
