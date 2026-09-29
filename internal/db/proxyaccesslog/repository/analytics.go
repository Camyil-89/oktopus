package repository

import "time"

// ReportFilters — подмножество фильтров списка журнала для агрегаций.
type ReportFilters struct {
	User          string
	Source        string
	Destination   string
	URL           string
	SearchOnly    bool
	Action        int32 // -1 любое, 0 deny, 1 allow
	DecisionRuleRef string
	InspectRuleID string
	InspectLog    map[string]string
	// FieldNonempty — только записи, где поле не пустая строка (колонка или inspect_log.*).
	FieldNonempty []string
}

// WidgetQuery — один запрос виджета (whitelist полей на уровне репозитория).
type WidgetQuery struct {
	Metric      string   // count, avg_decide_duration_us
	GroupByTime string   // 10m, 1h, 1d
	SplitBy     string   // action, denied_by
	GroupBy     string   // одно поле
	GroupByCols []string // несколько полей для table
	SearchColumns []string // подмножество group_by_cols: поиск по подстроке в таблице
	Limit       int
	Order       string // asc, desc
	Page        int    // table: номер страницы, с 1
	PageSize    int    // table: строк на странице, макс. 100
	Search      string // table: подстрока ILIKE по search_columns
	// FieldNonempty — дополнительно к filters.field_nonempty для одного виджета.
	FieldNonempty []string
}

// TimeseriesPoint — точка ряда.
type TimeseriesPoint struct {
	T     time.Time
	Value float64
}

// TimeseriesSeries — ряд с ключом разбиения.
type TimeseriesSeries struct {
	Key   string
	Label string
	Points []TimeseriesPoint
}

// LabeledValue — пара подпись / значение для bar и stat.
type LabeledValue struct {
	Label string
	Value float64
}

// TableResult — табличная агрегация.
type TableResult struct {
	Columns  []string
	Rows     [][]string
	Values   []float64
	Total    int
	Page     int
	PageSize int
}

// WidgetAnalyticsResult — результат одного виджета.
type WidgetAnalyticsResult struct {
	Timeseries *TimeseriesData
	Bar        []LabeledValue
	Stat       *LabeledValue
	Table      *TableResult
}

type TimeseriesData struct {
	Series []TimeseriesSeries
}
