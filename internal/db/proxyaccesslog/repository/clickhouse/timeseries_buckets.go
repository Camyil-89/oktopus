package clickhouse

import (
	"fmt"
	"time"

	"oktopus/internal/db/proxyaccesslog/repository"
)

type timeseriesBucketKind int

const (
	bucketCalendar     timeseriesBucketKind = 0
	bucketHourOfDay    timeseriesBucketKind = 1
	bucketDayOfWeek    timeseriesBucketKind = 2
	bucketMonthOfYear  timeseriesBucketKind = 3
)

type timeseriesBucketConfig struct {
	sqlExpr string
	kind    timeseriesBucketKind
}

func timeseriesBucketConfigFor(bucket string) (timeseriesBucketConfig, error) {
	switch bucket {
	case "10m":
		return timeseriesBucketConfig{
			sqlExpr: "toStartOfInterval(al.created_at, INTERVAL 10 minute)",
			kind:    bucketCalendar,
		}, nil
	case "1h":
		return timeseriesBucketConfig{
			sqlExpr: "toStartOfHour(al.created_at)",
			kind:    bucketCalendar,
		}, nil
	case "1d":
		return timeseriesBucketConfig{
			sqlExpr: "toStartOfDay(al.created_at)",
			kind:    bucketCalendar,
		}, nil
	case "hour_of_day":
		return timeseriesBucketConfig{
			sqlExpr: "toHour(al.created_at)",
			kind:    bucketHourOfDay,
		}, nil
	case "day_of_week":
		return timeseriesBucketConfig{
			sqlExpr: "toDayOfWeek(al.created_at)",
			kind:    bucketDayOfWeek,
		}, nil
	case "month_of_year":
		return timeseriesBucketConfig{
			sqlExpr: "toMonth(al.created_at)",
			kind:    bucketMonthOfYear,
		}, nil
	default:
		return timeseriesBucketConfig{}, fmt.Errorf("unsupported group_by_time")
	}
}

func calendarStep(bucket string) (time.Duration, error) {
	switch bucket {
	case "10m":
		return 10 * time.Minute, nil
	case "1h":
		return time.Hour, nil
	case "1d":
		return 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported calendar bucket")
	}
}

func alignCalendarStart(t time.Time, bucket string) time.Time {
	t = t.UTC()
	switch bucket {
	case "10m":
		return t.Truncate(10 * time.Minute)
	case "1h":
		return t.Truncate(time.Hour)
	case "1d":
		y, m, d := t.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	default:
		return t
	}
}

func hourOfDayBucketTime(hour int) time.Time {
	if hour < 0 {
		hour = 0
	}
	if hour > 23 {
		hour = 23
	}
	return time.Date(2000, 1, 1, hour, 0, 0, 0, time.UTC)
}

// ClickHouse toDayOfWeek: 1 = Monday … 7 = Sunday.
func dayOfWeekBucketTime(day int) time.Time {
	if day < 1 {
		day = 1
	}
	if day > 7 {
		day = 7
	}
	// 2000-01-03 is Monday.
	return time.Date(2000, 1, 2+day, 0, 0, 0, 0, time.UTC)
}

func monthOfYearBucketTime(month int) time.Time {
	if month < 1 {
		month = 1
	}
	if month > 12 {
		month = 12
	}
	return time.Date(2000, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
}

func fillTimeseriesGaps(
	from, to time.Time,
	bucket string,
	kind timeseriesBucketKind,
	data *repository.TimeseriesData,
) {
	if data == nil {
		return
	}
	if len(data.Series) == 0 {
		data.Series = []repository.TimeseriesSeries{
			{Key: "", Label: "—", Points: []repository.TimeseriesPoint{}},
		}
	}
	for i := range data.Series {
		data.Series[i].Points = fillSeriesPoints(from, to, bucket, kind, data.Series[i].Points)
	}
}

func fillSeriesPoints(
	from, to time.Time,
	bucket string,
	kind timeseriesBucketKind,
	points []repository.TimeseriesPoint,
) []repository.TimeseriesPoint {
	byT := make(map[int64]float64, len(points))
	for _, p := range points {
		t := p.T.UTC()
		if kind == bucketCalendar {
			t = alignCalendarStart(t, bucket)
		}
		byT[t.Unix()] = p.Value
	}

	var slots []time.Time
	switch kind {
	case bucketCalendar:
		step, err := calendarStep(bucket)
		if err != nil {
			return points
		}
		start := alignCalendarStart(from, bucket)
		end := to.UTC()
		for t := start; !t.After(end); t = t.Add(step) {
			slots = append(slots, t.UTC())
		}
	case bucketHourOfDay:
		for h := 0; h < 24; h++ {
			slots = append(slots, hourOfDayBucketTime(h))
		}
	case bucketDayOfWeek:
		for d := 1; d <= 7; d++ {
			slots = append(slots, dayOfWeekBucketTime(d))
		}
	case bucketMonthOfYear:
		for m := 1; m <= 12; m++ {
			slots = append(slots, monthOfYearBucketTime(m))
		}
	default:
		return points
	}

	out := make([]repository.TimeseriesPoint, 0, len(slots))
	for _, t := range slots {
		v := byT[t.Unix()]
		out = append(out, repository.TimeseriesPoint{T: t, Value: v})
	}
	return out
}
