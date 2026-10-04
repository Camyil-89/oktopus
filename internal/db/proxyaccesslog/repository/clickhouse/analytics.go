package clickhouse

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"oktopus/internal/db/proxyaccesslog/inspectlog"
	"oktopus/internal/db/proxyaccesslog/repository"
)

const maxReportRange = 31 * 24 * time.Hour

var groupBySQL = map[string]string{
	"instance_id":         "toString(al.instance_id)",
	"destination_address": "al.destination_address",
	"source_address":      "al.source_address",
	"user":                "coalesce(al.user_name, '')",
	"action":              "toString(al.action)",
	"denied_by":           "coalesce(al.denied_by, '')",
	"decision_rule_ref":   "al.decision_rule_ref",
	"search_engine":       "al.search_engine",
}

var splitBySQL = map[string]string{
	"action":    "toString(al.action)",
	"denied_by": "coalesce(al.denied_by, '')",
}

func (r *Repository) RunWidgetQuery(
	ctx context.Context,
	from, to time.Time,
	f repository.ReportFilters,
	q repository.WidgetQuery,
	widgetType string,
) (repository.WidgetAnalyticsResult, error) {
	if to.Before(from) {
		return repository.WidgetAnalyticsResult{}, fmt.Errorf("period end must not be before start")
	}
	if to.Sub(from) > maxReportRange {
		return repository.WidgetAnalyticsResult{}, fmt.Errorf("period must not exceed 31 days")
	}
	switch widgetType {
	case "timeseries":
		data, err := r.queryTimeseries(ctx, from, to, f, q)
		if err != nil {
			return repository.WidgetAnalyticsResult{}, err
		}
		return repository.WidgetAnalyticsResult{Timeseries: data}, nil
	case "bar":
		items, err := r.queryGroupBy(ctx, from, to, f, q)
		if err != nil {
			return repository.WidgetAnalyticsResult{}, err
		}
		return repository.WidgetAnalyticsResult{Bar: items}, nil
	case "stat":
		val, err := r.queryStat(ctx, from, to, f, q)
		if err != nil {
			return repository.WidgetAnalyticsResult{}, err
		}
		return repository.WidgetAnalyticsResult{Stat: &repository.LabeledValue{Label: "", Value: val}}, nil
	case "table":
		tbl, err := r.queryTable(ctx, from, to, f, q)
		if err != nil {
			return repository.WidgetAnalyticsResult{}, err
		}
		return repository.WidgetAnalyticsResult{Table: tbl}, nil
	default:
		return repository.WidgetAnalyticsResult{}, fmt.Errorf("unsupported widget type")
	}
}

func metricExpr(metric string) (string, error) {
	switch metric {
	case "", "count":
		return "count()", nil
	case "avg_decide_duration_us":
		return "avg(al.decide_duration_us)", nil
	default:
		return "", fmt.Errorf("unsupported metric")
	}
}

func timeBucketExpr(bucket string) (string, error) {
	switch bucket {
	case "10m":
		return "toStartOfInterval(al.created_at, INTERVAL 10 minute)", nil
	case "1h":
		return "toStartOfHour(al.created_at)", nil
	case "1d":
		return "toStartOfDay(al.created_at)", nil
	default:
		return "", fmt.Errorf("unsupported group_by_time")
	}
}

func (r *Repository) queryTimeseries(
	ctx context.Context,
	from, to time.Time,
	f repository.ReportFilters,
	q repository.WidgetQuery,
) (*repository.TimeseriesData, error) {
	bucketExpr, err := timeBucketExpr(q.GroupByTime)
	if err != nil {
		return nil, err
	}
	metric, err := metricExpr(q.Metric)
	if err != nil {
		return nil, err
	}
	where, args := buildReportWhere(from, to, f, q)

	selectSplit := "'' AS split_key"
	groupSplit := ""
	orderSplit := ""
	if q.SplitBy != "" {
		splitExpr, ok := splitBySQL[q.SplitBy]
		if !ok {
			return nil, fmt.Errorf("unsupported split_by")
		}
		selectSplit = splitExpr + " AS split_key"
		groupSplit = ", split_key"
		orderSplit = ", split_key"
	}

	sql := fmt.Sprintf(`
SELECT %s AS bucket, %s, toFloat64(%s) AS value
FROM proxy_access_log AS al
WHERE %s
GROUP BY bucket%s
ORDER BY bucket%s`, bucketExpr, selectSplit, metric, where, groupSplit, orderSplit)

	rows, err := r.conn.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("timeseries query: %w", err)
	}
	defer rows.Close()

	seriesMap := map[string]*repository.TimeseriesSeries{}
	seriesOrder := []string{}

	for rows.Next() {
		var bucket time.Time
		var splitKey string
		var value float64
		if err := rows.Scan(&bucket, &splitKey, &value); err != nil {
			return nil, fmt.Errorf("timeseries scan: %w", err)
		}
		if math.IsNaN(value) {
			value = 0
		}
		s, ok := seriesMap[splitKey]
		if !ok {
			s = &repository.TimeseriesSeries{
				Key:   splitKey,
				Label: splitLabel(q.SplitBy, splitKey),
			}
			seriesMap[splitKey] = s
			seriesOrder = append(seriesOrder, splitKey)
		}
		s.Points = append(s.Points, repository.TimeseriesPoint{T: bucket.UTC(), Value: value})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := repository.TimeseriesData{Series: make([]repository.TimeseriesSeries, 0, len(seriesOrder))}
	for _, k := range seriesOrder {
		out.Series = append(out.Series, *seriesMap[k])
	}
	return &out, nil
}

func splitLabel(splitBy, key string) string {
	if splitBy == "action" {
		if key == "1" {
			return "allow"
		}
		if key == "0" {
			return "deny"
		}
	}
	if key == "" {
		return "—"
	}
	return key
}

func (r *Repository) queryStat(
	ctx context.Context,
	from, to time.Time,
	f repository.ReportFilters,
	q repository.WidgetQuery,
) (float64, error) {
	metric, err := metricExpr(q.Metric)
	if err != nil {
		return 0, err
	}
	where, args := buildReportWhere(from, to, f, q)
	sql := fmt.Sprintf(
		`SELECT toFloat64(ifNull(%s, 0)) AS value FROM proxy_access_log AS al WHERE %s`,
		metric,
		where,
	)
	var value float64
	if err := r.conn.QueryRow(ctx, sql, args...).Scan(&value); err != nil {
		return 0, fmt.Errorf("stat query: %w", err)
	}
	if math.IsNaN(value) {
		value = 0
	}
	return value, nil
}

func (r *Repository) queryGroupBy(
	ctx context.Context,
	from, to time.Time,
	f repository.ReportFilters,
	q repository.WidgetQuery,
) ([]repository.LabeledValue, error) {
	col, err := resolveGroupBy(q.GroupBy)
	if err != nil {
		return nil, err
	}
	metric, err := metricExpr(q.Metric)
	if err != nil {
		return nil, err
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	order := "DESC"
	if strings.EqualFold(q.Order, "asc") {
		order = "ASC"
	}

	where, args := buildReportWhere(from, to, f, q)
	joins, err := inspectJoinClauses(q.GroupBy)
	if err != nil {
		return nil, err
	}
	sql := fmt.Sprintf(`
SELECT %s AS label, toFloat64(%s) AS value
FROM proxy_access_log AS al
%s
WHERE %s
GROUP BY label
ORDER BY value %s
LIMIT %d`, col, metric, joins, where, order, limit)

	rows, err := r.conn.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("group by query: %w", err)
	}
	defer rows.Close()

	var out []repository.LabeledValue
	for rows.Next() {
		var label string
		var value float64
		if err := rows.Scan(&label, &value); err != nil {
			return nil, err
		}
		if math.IsNaN(value) {
			value = 0
		}
		if label == "" {
			label = "—"
		}
		out = append(out, repository.LabeledValue{Label: label, Value: value})
	}
	return out, rows.Err()
}

func tablePageSize(q repository.WidgetQuery) int {
	ps := q.PageSize
	if ps <= 0 {
		if q.Limit > 0 {
			ps = q.Limit
		} else {
			ps = 20
		}
	}
	if ps > 100 {
		ps = 100
	}
	return ps
}

func tablePage(q repository.WidgetQuery) int {
	if q.Page < 1 {
		return 1
	}
	return q.Page
}

func (r *Repository) queryTable(
	ctx context.Context,
	from, to time.Time,
	f repository.ReportFilters,
	q repository.WidgetQuery,
) (*repository.TableResult, error) {
	cols := q.GroupByCols
	if len(cols) == 0 && q.GroupBy != "" {
		cols = []string{q.GroupBy}
	}
	if len(cols) == 0 {
		return nil, fmt.Errorf("group_by required for table")
	}
	if len(cols) > 4 {
		return nil, fmt.Errorf("at most 4 group_by columns")
	}
	metric, err := metricExpr(q.Metric)
	if err != nil {
		return nil, err
	}
	pageSize := tablePageSize(q)
	page := tablePage(q)
	offset := (page - 1) * pageSize
	order := "DESC"
	if strings.EqualFold(q.Order, "asc") {
		order = "ASC"
	}

	selectParts := make([]string, len(cols))
	groupParts := make([]string, len(cols))
	colAliases := make([]string, len(cols))
	for i, c := range cols {
		expr, err := resolveGroupBy(c)
		if err != nil {
			return nil, err
		}
		alias := "c" + fmt.Sprint(i)
		colAliases[i] = alias
		selectParts[i] = expr + " AS " + alias
		groupParts[i] = expr
	}

	where, args := buildReportWhere(from, to, f, q)
	joins, err := inspectJoinClauses(cols...)
	if err != nil {
		return nil, err
	}
	search := strings.TrimSpace(q.Search)
	searchableIdx := searchableColumnIndices(cols, q.SearchColumns)
	outerWhere := ""
	if search != "" && len(searchableIdx) > 0 {
		orParts := make([]string, 0, len(searchableIdx))
		for _, idx := range searchableIdx {
			orParts = append(orParts, fmt.Sprintf("toString(%s) ILIKE ?", colAliases[idx]))
			args = append(args, "%"+search+"%")
		}
		outerWhere = "WHERE (" + strings.Join(orParts, " OR ") + ")"
	}

	innerSelect := strings.Join(selectParts, ", ")
	sql := fmt.Sprintf(`
WITH grouped AS (
	SELECT %s, toFloat64(%s) AS value
	FROM proxy_access_log AS al
	%s
	WHERE %s
	GROUP BY %s
)
SELECT %s, value, count() OVER() AS total_count
FROM grouped
%s
ORDER BY value %s
LIMIT %d OFFSET %d`,
		innerSelect,
		metric,
		joins,
		where,
		strings.Join(groupParts, ", "),
		strings.Join(colAliases, ", "),
		outerWhere,
		order,
		pageSize,
		offset,
	)

	rows, err := r.conn.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("table query: %w", err)
	}
	defer rows.Close()

	result := &repository.TableResult{
		Columns:  cols,
		Rows:     [][]string{},
		Values:   []float64{},
		Page:     page,
		PageSize: pageSize,
	}
	var total uint64
	for rows.Next() {
		strCols := make([]string, len(cols))
		scans := make([]interface{}, len(cols)+2)
		for i := range cols {
			scans[i] = &strCols[i]
		}
		var value float64
		scans[len(cols)] = &value
		scans[len(cols)+1] = &total
		if err := rows.Scan(scans...); err != nil {
			return nil, err
		}
		if math.IsNaN(value) {
			value = 0
		}
		for i, s := range strCols {
			if s == "" {
				strCols[i] = "—"
			}
		}
		result.Rows = append(result.Rows, strCols)
		result.Values = append(result.Values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result.Total = int(total)
	return result, nil
}

func searchableColumnIndices(cols []string, searchColumns []string) []int {
	if len(searchColumns) == 0 {
		return nil
	}
	nameToIdx := make(map[string]int, len(cols))
	for i, c := range cols {
		nameToIdx[c] = i
	}
	var out []int
	seen := make(map[int]bool)
	for _, sc := range searchColumns {
		sc = strings.TrimSpace(sc)
		if sc == "" {
			continue
		}
		idx, ok := nameToIdx[sc]
		if !ok || seen[idx] {
			continue
		}
		seen[idx] = true
		out = append(out, idx)
	}
	return out
}

func resolveGroupBy(name string) (string, error) {
	if expr, ok := groupBySQL[name]; ok {
		return expr, nil
	}
	if key, ok := inspectlog.ParseGroupByField(name); ok {
		return inspectlog.ValueExprCH(key)
	}
	return "", fmt.Errorf("unsupported group_by")
}

func inspectJoinClauses(groupByNames ...string) (string, error) {
	seen := make(map[string]bool)
	var clauses []string
	for _, name := range groupByNames {
		key, ok := inspectlog.ParseGroupByField(name)
		if !ok {
			continue
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		clause, err := inspectlog.CHJoinClause(key)
		if err != nil {
			return "", err
		}
		clauses = append(clauses, clause)
	}
	return strings.Join(clauses, "\n"), nil
}

func buildReportWhere(from, to time.Time, f repository.ReportFilters, q repository.WidgetQuery) (string, []any) {
	conditions := []string{
		"al.created_at >= ?",
		"al.created_at <= ?",
	}
	args := []any{from.UTC(), to.UTC()}

	addILIKE := func(field string, val string) {
		val = strings.TrimSpace(val)
		if val == "" {
			return
		}
		conditions = append(conditions, field+" ILIKE ?")
		args = append(args, "%"+val+"%")
	}

	addILIKE("al.user_name", f.User)
	addILIKE("al.source_address", f.Source)
	addILIKE("al.destination_address", f.Destination)
	addILIKE("al.full_url", f.URL)

	if f.SearchOnly {
		conditions = append(conditions, "al.search_engine != ''")
	}
	if f.Action == 0 || f.Action == 1 {
		conditions = append(conditions, "al.action = ?")
		args = append(args, uint8(f.Action))
	}
	if inst := strings.TrimSpace(f.InstanceID); inst != "" {
		parsed, err := uuid.Parse(inst)
		if err == nil {
			conditions = append(conditions, "al.instance_id = ?")
			args = append(args, parsed)
		}
	}
	if ref := strings.TrimSpace(f.DecisionRuleRef); ref != "" {
		conditions = append(conditions, "al.decision_rule_ref = ?")
		args = append(args, ref)
	}
	if id := strings.TrimSpace(f.InspectRuleID); id != "" {
		parsed, err := uuid.Parse(id)
		if err == nil {
			conditions = append(conditions, "al.inspect_rule_id = ?")
			args = append(args, parsed)
		}
	}

	for field, val := range f.InspectLog {
		val = strings.TrimSpace(val)
		if val == "" {
			continue
		}
		if err := inspectlog.ValidateFieldKey(field); err != nil {
			continue
		}
		conditions = append(conditions, `al.id IN (
  SELECT log_id FROM proxy_access_log_inspect_kv
  WHERE field_key = ? AND field_value ILIKE ?
)`)
		args = append(args, field, "%"+val+"%")
	}

	for _, name := range repository.MergeFieldNonempty(f.FieldNonempty, q.FieldNonempty) {
		cond, condArgs, err := fieldNonemptyConditionCH(name, q)
		if err != nil {
			continue
		}
		conditions = append(conditions, cond)
		args = append(args, condArgs...)
	}

	return strings.Join(conditions, " AND "), args
}

func fieldNonemptyConditionCH(name string, q repository.WidgetQuery) (string, []any, error) {
	name = strings.TrimSpace(name)
	if key, ok := inspectlog.ParseGroupByField(name); ok {
		if err := inspectlog.ValidateFieldKey(key); err != nil {
			return "", nil, err
		}
		if expr, ok := inspectNonemptyFromJoin(key, q); ok {
			return fmt.Sprintf("(%s) != ''", expr), nil, nil
		}
		return `al.id IN (
  SELECT log_id FROM proxy_access_log_inspect_kv
  WHERE field_key = ? AND field_value != ''
)`, []any{key}, nil
	}
	expr, ok := groupBySQL[name]
	if !ok {
		return "", nil, fmt.Errorf("unsupported field_nonempty %q", name)
	}
	return fmt.Sprintf("(%s) != ''", expr), nil, nil
}

func inspectNonemptyFromJoin(key string, q repository.WidgetQuery) (string, bool) {
	names := append([]string{}, q.GroupByCols...)
	if q.GroupBy != "" {
		names = append(names, q.GroupBy)
	}
	for _, name := range names {
		k, ok := inspectlog.ParseGroupByField(name)
		if !ok || k != key {
			continue
		}
		expr, err := inspectlog.ValueExprCH(key)
		if err != nil {
			return "", false
		}
		return expr, true
	}
	return "", false
}
