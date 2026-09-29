package view

import (
	"encoding/json"
	"net/http"
	"strings"

	"oktopus/internal/api/platform/response"
	proxyaccesslogservice "oktopus/internal/db/proxyaccesslog/service"
)

func (h *AccessLogHandler) RunReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body reportSpecRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid json")
		return
	}

	spec := proxyaccesslogservice.ReportSpec{
		Version: body.Version,
		Time: proxyaccesslogservice.ReportTimeRange{
			From: body.Time.From,
			To:   body.Time.To,
		},
		Filters: proxyaccesslogservice.ReportSpecFilters{
			User:          body.Filters.User,
			Source:        body.Filters.Source,
			Destination:   body.Filters.Destination,
			URL:           body.Filters.URL,
			SearchOnly:    body.Filters.SearchOnly,
			DecisionRuleRef: body.Filters.DecisionRuleRef,
			InspectRuleID: body.Filters.InspectRuleID,
			InspectLog:    body.Filters.InspectLog,
			FieldNonempty: body.Filters.FieldNonempty,
		},
	}
	if body.Filters.Action != nil {
		spec.Filters.Action = body.Filters.Action
	}
	for _, w := range body.Widgets {
		spec.Widgets = append(spec.Widgets, proxyaccesslogservice.ReportWidget{
			ID:    w.ID,
			Type:  w.Type,
			Title: w.Title,
			Query: proxyaccesslogservice.ReportWidgetQuery{
				Metric:        w.Query.Metric,
				GroupByTime:   w.Query.GroupByTime,
				SplitBy:       w.Query.SplitBy,
				GroupBy:       w.Query.GroupBy,
				GroupByCols:   w.Query.GroupByCols,
				SearchColumns: w.Query.SearchColumns,
				Limit:         w.Query.Limit,
				Order:         w.Query.Order,
				FieldNonempty: w.Query.FieldNonempty,
			},
		})
	}

	result, err := h.logs.RunReport(r.Context(), spec)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "widget") ||
			strings.Contains(msg, "period") ||
			strings.Contains(msg, "unsupported") ||
			strings.Contains(msg, "required") ||
			strings.Contains(msg, "invalid") {
			response.Error(w, http.StatusBadRequest, msg)
			return
		}
		response.Error(w, http.StatusInternalServerError, "report failed")
		return
	}

	response.JSON(w, http.StatusOK, toReportResponse(result))
}

func (h *AccessLogHandler) RunReportTable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body reportTableRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid json")
		return
	}

	spec := proxyaccesslogservice.ReportSpec{
		Version: body.Version,
		Time: proxyaccesslogservice.ReportTimeRange{
			From: body.Time.From,
			To:   body.Time.To,
		},
		Filters: proxyaccesslogservice.ReportSpecFilters{
			User:          body.Filters.User,
			Source:        body.Filters.Source,
			Destination:   body.Filters.Destination,
			URL:           body.Filters.URL,
			SearchOnly:    body.Filters.SearchOnly,
			DecisionRuleRef: body.Filters.DecisionRuleRef,
			InspectRuleID: body.Filters.InspectRuleID,
			InspectLog:    body.Filters.InspectLog,
			FieldNonempty: body.Filters.FieldNonempty,
		},
	}
	if body.Filters.Action != nil {
		spec.Filters.Action = body.Filters.Action
	}

	widget := proxyaccesslogservice.ReportWidget{
		ID:    body.Widget.ID,
		Type:  body.Widget.Type,
		Title: body.Widget.Title,
		Query: proxyaccesslogservice.ReportWidgetQuery{
			Metric:        body.Widget.Query.Metric,
			GroupBy:       body.Widget.Query.GroupBy,
			GroupByCols:   body.Widget.Query.GroupByCols,
			SearchColumns: body.Widget.Query.SearchColumns,
			Limit:         body.Widget.Query.Limit,
			Order:         body.Widget.Query.Order,
			FieldNonempty: body.Widget.Query.FieldNonempty,
		},
	}

	result, err := h.logs.RunReportTableWidget(r.Context(), spec, widget, proxyaccesslogservice.ReportTableParams{
		Page:     body.Page,
		PageSize: body.PageSize,
		Search:   body.Search,
	})
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "widget") ||
			strings.Contains(msg, "period") ||
			strings.Contains(msg, "unsupported") ||
			strings.Contains(msg, "required") ||
			strings.Contains(msg, "invalid") ||
			strings.Contains(msg, "search_columns") {
			response.Error(w, http.StatusBadRequest, msg)
			return
		}
		response.Error(w, http.StatusInternalServerError, "report failed")
		return
	}

	response.JSON(w, http.StatusOK, toSingleWidgetResponse(result))
}

type reportSpecRequest struct {
	Version int                   `json:"version"`
	Time    reportTimeRequest     `json:"time"`
	Filters reportFiltersRequest  `json:"filters"`
	Widgets []reportWidgetRequest `json:"widgets"`
}

type reportTimeRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type reportFiltersRequest struct {
	User          string `json:"user"`
	Source        string `json:"source"`
	Destination   string `json:"destination"`
	URL           string `json:"url"`
	SearchOnly    bool   `json:"search_only"`
	Action        *int32 `json:"action"`
	DecisionRuleRef string            `json:"decision_rule_ref"`
	InspectRuleID string            `json:"inspect_rule_id"`
	InspectLog    map[string]string `json:"inspect_log"`
	FieldNonempty []string          `json:"field_nonempty"`
}

type reportWidgetRequest struct {
	ID    string                `json:"id"`
	Type  string                `json:"type"`
	Title string                `json:"title"`
	Query reportWidgetQueryBody `json:"query"`
}

type reportWidgetQueryBody struct {
	Metric        string   `json:"metric"`
	GroupByTime   string   `json:"group_by_time"`
	SplitBy       string   `json:"split_by"`
	GroupBy       string   `json:"group_by"`
	GroupByCols   []string `json:"group_by_cols"`
	SearchColumns []string `json:"search_columns"`
	Limit         int      `json:"limit"`
	Order         string   `json:"order"`
	FieldNonempty []string `json:"field_nonempty"`
}

type reportTableRequest struct {
	Version  int                   `json:"version"`
	Time     reportTimeRequest     `json:"time"`
	Filters  reportFiltersRequest  `json:"filters"`
	Widget   reportWidgetRequest   `json:"widget"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
	Search   string                `json:"search"`
}

type reportResponse struct {
	Widgets []reportWidgetResponse `json:"widgets"`
}

type reportWidgetResponse struct {
	ID    string          `json:"id"`
	Type  string          `json:"type"`
	Title string          `json:"title"`
	Data  json.RawMessage `json:"data"`
}

func toReportResponse(result proxyaccesslogservice.ReportResult) reportResponse {
	widgets := make([]reportWidgetResponse, 0, len(result.Widgets))
	for _, w := range result.Widgets {
		data := map[string]interface{}{}
		switch w.Type {
		case "timeseries":
			if w.Data.Timeseries != nil {
				series := make([]map[string]interface{}, 0, len(w.Data.Timeseries.Series))
				for _, s := range w.Data.Timeseries.Series {
					points := make([]map[string]interface{}, 0, len(s.Points))
					for _, p := range s.Points {
						points = append(points, map[string]interface{}{
							"t":     p.T.UTC().Format("2006-01-02T15:04:05Z"),
							"value": p.Value,
						})
					}
					series = append(series, map[string]interface{}{
						"key":    s.Key,
						"label":  s.Label,
						"points": points,
					})
				}
				data["series"] = series
			}
		case "bar":
			items := make([]map[string]interface{}, 0, len(w.Data.Bar))
			for _, it := range w.Data.Bar {
				items = append(items, map[string]interface{}{
					"label": it.Label,
					"value": it.Value,
				})
			}
			data["items"] = items
		case "stat":
			if w.Data.Stat != nil {
				data["value"] = w.Data.Stat.Value
				data["label"] = w.Data.Stat.Label
			}
		case "table":
			if w.Data.Table != nil {
				data["columns"] = w.Data.Table.Columns
				data["rows"] = w.Data.Table.Rows
				data["values"] = w.Data.Table.Values
				data["total"] = w.Data.Table.Total
				data["page"] = w.Data.Table.Page
				data["page_size"] = w.Data.Table.PageSize
			}
		}
		raw, _ := json.Marshal(data)
		widgets = append(widgets, reportWidgetResponse{
			ID:    w.ID,
			Type:  w.Type,
			Title: w.Title,
			Data:  raw,
		})
	}
	return reportResponse{Widgets: widgets}
}

func toSingleWidgetResponse(w proxyaccesslogservice.ReportWidgetResult) reportWidgetResponse {
	full := toReportResponse(proxyaccesslogservice.ReportResult{Widgets: []proxyaccesslogservice.ReportWidgetResult{w}})
	if len(full.Widgets) > 0 {
		return full.Widgets[0]
	}
	return reportWidgetResponse{ID: w.ID, Type: w.Type, Title: w.Title, Data: json.RawMessage("{}")}
}
