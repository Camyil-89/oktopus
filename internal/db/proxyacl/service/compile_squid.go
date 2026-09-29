package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"

	"oktopus/internal/db/proxyacl/domain"
	"oktopus/internal/id"
	"oktopus/internal/proxy/acl"
	"oktopus/internal/proxy/acl/squid"
)

func compileSquidPolicy(configText string, lists []domain.NamedList) (*acl.Engine, string, acl.SNIPatternIndexReport, error) {
	inputs := domainListsToSquid(lists)
	rev := revisionFromSquid(configText, lists)
	res := squid.Analyze(configText, inputs)
	if !res.OK {
		return nil, rev, acl.SNIPatternIndexReport{}, compileErrorFromAnalyze(res)
	}
	if res.Engine == nil {
		return acl.EmptyEngine(), rev, res.SNIIndexReport, nil
	}
	return res.Engine, rev, res.SNIIndexReport, nil
}

func compileSquidFromStore(ctx context.Context, repo loadSquidSource) (*acl.Engine, string, acl.SNIPatternIndexReport, error) {
	pol, err := repo.GetPolicy(ctx)
	if err != nil {
		return nil, "", acl.SNIPatternIndexReport{}, err
	}
	lists, err := referencedNamedLists(ctx, repo, pol.ConfigText)
	if err != nil {
		return nil, "", acl.SNIPatternIndexReport{}, err
	}
	return compileSquidPolicy(pol.ConfigText, lists)
}

type loadSquidSource interface {
	GetPolicy(ctx context.Context) (domain.Policy, error)
	ListNamedListsByNames(ctx context.Context, names []string) ([]domain.NamedList, error)
}

func referencedNamedLists(ctx context.Context, repo loadSquidSource, configText string) ([]domain.NamedList, error) {
	cfg, _ := squid.ParsePolicyConfig(configText)
	names := squid.ListNamesToMergeFromDB(cfg)
	if len(names) == 0 {
		return nil, nil
	}
	return repo.ListNamedListsByNames(ctx, names)
}

func filterNamedListsByNames(lists []domain.NamedList, names []string) []domain.NamedList {
	if len(names) == 0 || len(lists) == 0 {
		return nil
	}
	want := make(map[string]struct{}, len(names))
	for _, n := range names {
		want[strings.ToLower(strings.TrimSpace(n))] = struct{}{}
	}
	out := make([]domain.NamedList, 0, len(names))
	for _, l := range lists {
		if _, ok := want[strings.ToLower(strings.TrimSpace(l.Name))]; ok {
			out = append(out, l)
		}
	}
	return out
}

func domainListsToSquid(lists []domain.NamedList) []squid.ListInput {
	out := make([]squid.ListInput, 0, len(lists))
	for _, l := range lists {
		out = append(out, squid.ListInput{
			Name:     l.Name,
			ListType: l.ListType,
			Body:     l.Body,
		})
	}
	return out
}

func revisionFromSquid(configText string, lists []domain.NamedList) string {
	h := sha256.New()
	h.Write([]byte("policy\x00"))
	h.Write([]byte(configText))
	h.Write([]byte{0})
	sorted := append([]domain.NamedList(nil), lists...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})
	for _, l := range sorted {
		h.Write([]byte(l.Name))
		h.Write([]byte{0})
		h.Write([]byte(l.ListType))
		h.Write([]byte{0})
		h.Write([]byte(l.Body))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func validateSquidCompile(ctx context.Context, repo loadSquidSource) error {
	pol, err := repo.GetPolicy(ctx)
	if err != nil {
		return err
	}
	lists, err := referencedNamedLists(ctx, repo, pol.ConfigText)
	if err != nil {
		return err
	}
	res := squid.AnalyzeValidate(pol.ConfigText, domainListsToSquid(lists))
	if !res.OK {
		return compileErrorFromAnalyze(res)
	}
	return nil
}

func compileErrorFromAnalyze(res squid.AnalyzeResult) error {
	if len(res.Diagnostics) == 0 {
		return errors.New("compile failed")
	}
	return &squid.CompileErrors{Diagnostics: res.Diagnostics}
}

// DiagnosticsFromError извлекает диагностики компиляции squid ACL.
func DiagnosticsFromError(err error) []squid.Diagnostic {
	var ce *squid.CompileErrors
	if errors.As(err, &ce) {
		return ce.Diagnostics
	}
	return nil
}

type SyncNamedListInput struct {
	ID                  *string
	Name                string
	ListType            string
	Body                string
	SourceMode          string
	SourceURL           string
	PollIntervalMinutes int
}

func normalizeNamedLists(inputs []SyncNamedListInput) ([]domain.NamedList, error) {
	out := make([]domain.NamedList, 0, len(inputs))
	seen := make(map[string]int)
	for i, in := range inputs {
		name := strings.TrimSpace(in.Name)
		if name == "" {
			return nil, fmt.Errorf("list %d: name is required", i)
		}
		lt := strings.ToLower(strings.TrimSpace(in.ListType))
		switch lt {
		case "src", "port":
		case "dstdomain", "sni":
			if lt == "sni" {
				lt = "dstdomain"
			}
		default:
			return nil, fmt.Errorf("list %q: list_type must be src, dstdomain or port", name)
		}
		if idx, dup := seen[strings.ToLower(name)]; dup {
			return nil, fmt.Errorf("list %q: duplicate name (also at index %d)", name, idx)
		}
		seen[strings.ToLower(name)] = i
		var listID uuid.UUID
		if in.ID != nil && strings.TrimSpace(*in.ID) != "" {
			parsed, err := uuid.Parse(strings.TrimSpace(*in.ID))
			if err != nil {
				return nil, fmt.Errorf("list %q: invalid id", name)
			}
			listID = parsed
		} else {
			listID = id.MustNew()
		}
		sourceMode, err := normalizeListSourceMode(in.SourceMode)
		if err != nil {
			return nil, fmt.Errorf("list %q: %w", name, err)
		}
		body := strings.TrimSpace(in.Body)
		sourceURL := strings.TrimSpace(in.SourceURL)
		pollMin := in.PollIntervalMinutes
		if sourceMode == domain.ListSourceModeManual {
			if body == "" {
				return nil, fmt.Errorf("list %q: body is required", name)
			}
			sourceURL = ""
			pollMin = 60
		} else {
			if err := validateRemoteSourceURL(sourceURL); err != nil {
				return nil, fmt.Errorf("list %q: %w", name, err)
			}
			if pollMin < 1 {
				pollMin = 60
			}
		}
		out = append(out, domain.NamedList{
			ID:                  listID,
			Name:                name,
			ListType:            lt,
			Body:                body,
			SourceMode:          sourceMode,
			SourceURL:           sourceURL,
			PollIntervalMinutes: pollMin,
		})
	}
	return out, nil
}

func (s *Service) preserveRemoteListBodies(ctx context.Context, lists []domain.NamedList) ([]domain.NamedList, error) {
	out := append([]domain.NamedList(nil), lists...)
	for i, l := range out {
		if l.SourceMode != domain.ListSourceModeRemote {
			continue
		}
		if strings.TrimSpace(l.Body) != "" {
			continue
		}
		existing, err := s.repo.GetNamedList(ctx, l.ID)
		if err != nil {
			if errors.Is(err, domain.ErrListNotFound) {
				continue
			}
			return nil, err
		}
		out[i].Body = existing.Body
	}
	return out, nil
}

func (s *Service) ensureRemoteListBodies(ctx context.Context, lists []domain.NamedList) ([]domain.NamedList, error) {
	out := append([]domain.NamedList(nil), lists...)
	for i, l := range out {
		if l.SourceMode != domain.ListSourceModeRemote {
			continue
		}
		if strings.TrimSpace(l.Body) != "" {
			continue
		}
		body, err := fetchListBodyFromURL(l.SourceURL)
		if err != nil {
			return nil, fmt.Errorf("list %q: fetch remote source: %w", l.Name, err)
		}
		if body == "" {
			return nil, fmt.Errorf("list %q: remote source returned empty body", l.Name)
		}
		out[i].Body = body
	}
	return out, nil
}

func normalizeListSourceMode(mode string) (string, error) {
	m := strings.ToLower(strings.TrimSpace(mode))
	if m == "" {
		return domain.ListSourceModeManual, nil
	}
	switch m {
	case domain.ListSourceModeManual, domain.ListSourceModeRemote:
		return m, nil
	default:
		return "", fmt.Errorf("source_mode must be manual or remote")
	}
}

func countSquidSources(lists []domain.NamedList, policy domain.Policy) int {
	n := 0
	if strings.TrimSpace(policy.ConfigText) != "" {
		n++
	}
	n += len(lists)
	return n
}
