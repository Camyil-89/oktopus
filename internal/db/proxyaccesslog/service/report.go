package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"oktopus/internal/db/proxyaccesslog/inspectlog"
	"oktopus/internal/db/proxyaccesslog/repository"
)

const maxReportWidgets = 12

// ReportSpec — декларативный отчёт (version 1).
type ReportSpec struct {
	Version int
	Time    ReportTimeRange
	Filters ReportSpecFilters
	Widgets []ReportWidget
}

type ReportTimeRange struct {
	From string
	To   string
}

type ReportSpecFilters struct {
	InstanceID    string
	User          string
	Source        string
	Destination   string
	URL           string
	SearchOnly    bool
	Action        *int32
	DecisionRuleRef string
	InspectRuleID   string
	InspectLog    map[string]string
	FieldNonempty []string
}

type ReportWidget struct {
	ID    string
	Type  string
	Title string
	Query ReportWidgetQuery
}

type ReportWidgetQuery struct {
	Metric      string
	GroupByTime string
	SplitBy     string
	GroupBy     string
	GroupByCols   []string
	SearchColumns []string
	Limit         int
	Order         string
	FieldNonempty []string
}

type ReportWidgetResult struct {
	ID    string
	Type  string
	Title string
	Data  repository.WidgetAnalyticsResult
}

type ReportResult struct {
	Widgets []ReportWidgetResult
}

// ReportTableParams — пагинация и поиск табличного виджета (не часть сохранённого spec).
type ReportTableParams struct {
	Page     int
	PageSize int
	Search   string
}

func (s *Service) RunReport(ctx context.Context, spec ReportSpec) (ReportResult, error) {
	if spec.Version != 1 {
		return ReportResult{}, fmt.Errorf("unsupported report version")
	}
	if len(spec.Widgets) == 0 {
		return ReportResult{}, fmt.Errorf("widgets required")
	}
	if len(spec.Widgets) > maxReportWidgets {
		return ReportResult{}, fmt.Errorf("at most %d widgets", maxReportWidgets)
	}

	from, to, err := parseReportTimeRange(spec.Time)
	if err != nil {
		return ReportResult{}, err
	}

	filters, err := buildReportFilters(spec.Filters)
	if err != nil {
		return ReportResult{}, err
	}

	out := ReportResult{Widgets: make([]ReportWidgetResult, 0, len(spec.Widgets))}
	for _, w := range spec.Widgets {
		wtype := strings.TrimSpace(w.Type)
		if wtype == "" {
			return ReportResult{}, fmt.Errorf("widget type required")
		}
		q := repository.WidgetQuery{
			Metric:        strings.TrimSpace(w.Query.Metric),
			GroupByTime:   strings.TrimSpace(w.Query.GroupByTime),
			SplitBy:       strings.TrimSpace(w.Query.SplitBy),
			GroupBy:       strings.TrimSpace(w.Query.GroupBy),
			GroupByCols:   w.Query.GroupByCols,
			SearchColumns: w.Query.SearchColumns,
			Limit:         w.Query.Limit,
			Order:         strings.TrimSpace(w.Query.Order),
			FieldNonempty: w.Query.FieldNonempty,
		}
		if err := applyFieldNonempty(&q.FieldNonempty, w.Query.FieldNonempty, "query.field_nonempty"); err != nil {
			return ReportResult{}, fmt.Errorf("widget %q: %w", w.ID, err)
		}
		if wtype == "table" {
			q.Page = 1
			q.PageSize = defaultTablePageSize(w.Query.Limit, 0)
		}
		if err := validateWidgetQuery(wtype, q); err != nil {
			return ReportResult{}, fmt.Errorf("widget %q: %w", w.ID, err)
		}
		data, err := s.repo.RunWidgetQuery(ctx, from, to, filters, q, wtype)
		if err != nil {
			return ReportResult{}, fmt.Errorf("widget %q: %w", w.ID, err)
		}
		out.Widgets = append(out.Widgets, ReportWidgetResult{
			ID:    w.ID,
			Type:  wtype,
			Title: strings.TrimSpace(w.Title),
			Data:  data,
		})
	}
	return out, nil
}

func (s *Service) RunReportTableWidget(
	ctx context.Context,
	spec ReportSpec,
	widget ReportWidget,
	params ReportTableParams,
) (ReportWidgetResult, error) {
	if spec.Version != 1 {
		return ReportWidgetResult{}, fmt.Errorf("unsupported report version")
	}
	wtype := strings.TrimSpace(widget.Type)
	if wtype != "table" {
		return ReportWidgetResult{}, fmt.Errorf("widget type must be table")
	}
	from, to, err := parseReportTimeRange(spec.Time)
	if err != nil {
		return ReportWidgetResult{}, err
	}
	filters, err := buildReportFilters(spec.Filters)
	if err != nil {
		return ReportWidgetResult{}, err
	}
	q := repository.WidgetQuery{
		Metric:        strings.TrimSpace(widget.Query.Metric),
		GroupBy:       strings.TrimSpace(widget.Query.GroupBy),
		GroupByCols:   widget.Query.GroupByCols,
		SearchColumns: widget.Query.SearchColumns,
		Limit:         widget.Query.Limit,
		Order:         strings.TrimSpace(widget.Query.Order),
		Page:          params.Page,
		PageSize:      defaultTablePageSize(widget.Query.Limit, params.PageSize),
		Search:        strings.TrimSpace(params.Search),
		FieldNonempty: widget.Query.FieldNonempty,
	}
	if err := applyFieldNonempty(&q.FieldNonempty, widget.Query.FieldNonempty, "query.field_nonempty"); err != nil {
		return ReportWidgetResult{}, fmt.Errorf("widget %q: %w", widget.ID, err)
	}
	if err := validateWidgetQuery(wtype, q); err != nil {
		return ReportWidgetResult{}, fmt.Errorf("widget %q: %w", widget.ID, err)
	}
	data, err := s.repo.RunWidgetQuery(ctx, from, to, filters, q, wtype)
	if err != nil {
		return ReportWidgetResult{}, fmt.Errorf("widget %q: %w", widget.ID, err)
	}
	return ReportWidgetResult{
		ID:    strings.TrimSpace(widget.ID),
		Type:  wtype,
		Title: strings.TrimSpace(widget.Title),
		Data:  data,
	}, nil
}

func buildReportFilters(f ReportSpecFilters) (repository.ReportFilters, error) {
	filters := repository.ReportFilters{
		InstanceID:    strings.TrimSpace(f.InstanceID),
		User:          strings.TrimSpace(f.User),
		Source:        strings.TrimSpace(f.Source),
		Destination:   strings.TrimSpace(f.Destination),
		URL:           strings.TrimSpace(f.URL),
		SearchOnly:    f.SearchOnly,
		Action:        -1,
		DecisionRuleRef: strings.TrimSpace(f.DecisionRuleRef),
		InspectRuleID:   strings.TrimSpace(f.InspectRuleID),
	}
	if f.Action != nil {
		a := *f.Action
		if a != 0 && a != 1 {
			return repository.ReportFilters{}, fmt.Errorf("invalid filter action")
		}
		filters.Action = a
	}
	if len(f.InspectLog) > 0 {
		filters.InspectLog = make(map[string]string, len(f.InspectLog))
		for k, v := range f.InspectLog {
			key := strings.TrimSpace(k)
			if key == "" {
				continue
			}
			if err := inspectlog.ValidateFieldKey(key); err != nil {
				return repository.ReportFilters{}, fmt.Errorf("invalid inspect_log filter key %q", key)
			}
			val := strings.TrimSpace(v)
			if val == "" {
				continue
			}
			filters.InspectLog[key] = val
		}
	}
	if inst := filters.InstanceID; inst != "" {
		if _, err := uuid.Parse(inst); err != nil {
			return repository.ReportFilters{}, fmt.Errorf("invalid filter instance_id")
		}
	}
	if err := applyFieldNonempty(&filters.FieldNonempty, f.FieldNonempty, "filters.field_nonempty"); err != nil {
		return repository.ReportFilters{}, err
	}
	return filters, nil
}

func applyFieldNonempty(dst *[]string, raw []string, ctx string) error {
	if len(raw) == 0 {
		return nil
	}
	out := make([]string, 0, len(raw))
	seen := make(map[string]bool)
	for i, name := range raw {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if err := repository.ValidateReportFieldName(name); err != nil {
			return fmt.Errorf("%s[%d]: %w", ctx, i, err)
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	*dst = out
	return nil
}

func defaultTablePageSize(specLimit int, requested int) int {
	ps := requested
	if ps <= 0 {
		if specLimit > 0 {
			ps = specLimit
		} else {
			ps = 20
		}
	}
	if ps > 100 {
		ps = 100
	}
	return ps
}

func parseReportTimeRange(tr ReportTimeRange) (time.Time, time.Time, error) {
	fromRaw := strings.TrimSpace(tr.From)
	toRaw := strings.TrimSpace(tr.To)
	if fromRaw == "" || toRaw == "" {
		return time.Time{}, time.Time{}, fmt.Errorf("time.from and time.to required")
	}
	from, err := time.Parse(time.RFC3339, fromRaw)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid time.from")
	}
	to, err := time.Parse(time.RFC3339, toRaw)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid time.to")
	}
	return from.UTC(), to.UTC(), nil
}

func validateWidgetQuery(wtype string, q repository.WidgetQuery) error {
	switch wtype {
	case "timeseries":
		if q.GroupByTime == "" {
			return fmt.Errorf("query.group_by_time required")
		}
		return nil
	case "bar":
		if q.GroupBy == "" {
			return fmt.Errorf("query.group_by required")
		}
		return nil
	case "stat":
		return nil
	case "table":
		if q.GroupBy == "" && len(q.GroupByCols) == 0 {
			return fmt.Errorf("query.group_by or group_by_cols required")
		}
		cols := q.GroupByCols
		if len(cols) == 0 && q.GroupBy != "" {
			cols = []string{q.GroupBy}
		}
		colSet := make(map[string]bool, len(cols))
		for _, c := range cols {
			colSet[c] = true
		}
		for _, sc := range q.SearchColumns {
			sc = strings.TrimSpace(sc)
			if sc == "" {
				return fmt.Errorf("search_columns: empty name")
			}
			if !colSet[sc] {
				return fmt.Errorf("search_columns: %q must be in group_by_cols", sc)
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported widget type")
	}
}
