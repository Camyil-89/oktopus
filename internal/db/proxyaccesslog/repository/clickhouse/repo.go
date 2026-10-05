package clickhouse

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"

	"oktopus/internal/db/proxyaccesslog/domain"
	"oktopus/internal/db/proxyaccesslog/extra"
	"oktopus/internal/db/proxyaccesslog/repository"
)

type Repository struct {
	conn driver.Conn
}

func New(conn driver.Conn) repository.Repository {
	return &Repository{conn: conn}
}

func (r *Repository) InsertBatch(ctx context.Context, entries []domain.Entry) error {
	if len(entries) == 0 {
		return nil
	}

	logBatch, err := r.conn.PrepareBatch(ctx, `INSERT INTO proxy_access_log (
		id, instance_id, created_at, source_address, destination_address, user_name,
		decide_duration_us, full_url, action, inspect_rule_id, denied_by,
		decision_rule_ref, search_engine, search_query, inspect_error
	)`)
	if err != nil {
		return fmt.Errorf("prepare access log batch: %w", err)
	}

	var kvRows []extra.KVRow
	for _, e := range entries {
		parsed := extra.ParseFromJSON(e.ID, e.Extra)
		if err := logBatch.Append(
			e.ID,
			e.InstanceID,
			e.CreatedAt.UTC(),
			e.SourceAddress,
			e.DestinationAddress,
			nullableString(e.UserName),
			e.DecideDurationUs,
			e.FullURL,
			uint8(e.Action),
			nullableUUID(e.InspectRuleID),
			nullableString(e.DeniedBy),
			e.DecisionRuleRef,
			parsed.SearchEngine,
			parsed.SearchQuery,
			parsed.InspectError,
		); err != nil {
			return fmt.Errorf("append access log: %w", err)
		}
		for _, row := range parsed.KV {
			kvRows = append(kvRows, row)
		}
	}
	if err := logBatch.Send(); err != nil {
		return fmt.Errorf("send access log batch: %w", err)
	}

	if len(kvRows) == 0 {
		return nil
	}

	kvBatch, err := r.conn.PrepareBatch(ctx, `INSERT INTO proxy_access_log_inspect_kv (
		log_id, created_at, inspect_rule_id, field_key, field_value
	)`)
	if err != nil {
		return fmt.Errorf("prepare inspect kv batch: %w", err)
	}
	createdByLog := make(map[uuid.UUID]time.Time, len(entries))
	for _, e := range entries {
		createdByLog[e.ID] = e.CreatedAt.UTC()
	}
	for _, row := range kvRows {
		at := createdByLog[row.LogID]
		if err := kvBatch.Append(
			row.LogID,
			at,
			row.InspectRuleID,
			row.FieldKey,
			row.FieldValue,
		); err != nil {
			return fmt.Errorf("append inspect kv: %w", err)
		}
	}
	if err := kvBatch.Send(); err != nil {
		return fmt.Errorf("send inspect kv batch: %w", err)
	}
	return nil
}

func (r *Repository) List(ctx context.Context, f repository.ListFilter, page, pageSize int) (repository.Page, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	where, args := listWhere(f)
	countSQL := `SELECT count() FROM proxy_access_log WHERE ` + where
	var total uint64
	if err := r.conn.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return repository.Page{}, fmt.Errorf("count access log: %w", err)
	}

	offset := (page - 1) * pageSize
	listSQL := `SELECT
		id, instance_id, created_at, source_address, destination_address, user_name,
		decide_duration_us, full_url, action, inspect_rule_id, denied_by,
		decision_rule_ref, search_engine, search_query, inspect_error
	FROM proxy_access_log
	WHERE ` + where + `
	ORDER BY created_at DESC, id DESC
	LIMIT ? OFFSET ?`
	listArgs := append(append([]any{}, args...), pageSize, offset)

	rows, err := r.conn.Query(ctx, listSQL, listArgs...)
	if err != nil {
		return repository.Page{}, fmt.Errorf("list access log: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Entry, 0, pageSize)
	ids := make([]uuid.UUID, 0, pageSize)
	rowMeta := make([]struct {
		searchEngine string
		searchQuery  string
		inspectError string
	}, 0, pageSize)

	for rows.Next() {
		var (
			id                 uuid.UUID
			instanceID         uuid.UUID
			createdAt          time.Time
			source             string
			destination        string
			user               *string
			decideUs           int64
			fullURL            string
			action             uint8
			inspectRule        *uuid.UUID
			deniedBy           *string
			decisionRef        string
			searchEngine       string
			searchQuery        string
			inspectError       string
		)
		if err := rows.Scan(
			&id, &instanceID, &createdAt, &source, &destination, &user,
			&decideUs, &fullURL, &action, &inspectRule, &deniedBy,
			&decisionRef, &searchEngine, &searchQuery, &inspectError,
		); err != nil {
			return repository.Page{}, fmt.Errorf("scan access log: %w", err)
		}
		ids = append(ids, id)
		rowMeta = append(rowMeta, struct {
			searchEngine string
			searchQuery  string
			inspectError string
		}{searchEngine, searchQuery, inspectError})
		items = append(items, domain.Entry{
			ID:                 id,
			InstanceID:         instanceID,
			CreatedAt:          createdAt.UTC(),
			SourceAddress:      source,
			DestinationAddress: destination,
			UserName:           user,
			DecideDurationUs:   decideUs,
			FullURL:            fullURL,
			Action:             int16(action),
			InspectRuleID:      inspectRule,
			DeniedBy:           deniedBy,
			DecisionRuleRef:    decisionRef,
		})
	}
	if err := rows.Err(); err != nil {
		return repository.Page{}, err
	}

	kvByLog, err := r.loadKVForLogs(ctx, ids)
	if err != nil {
		return repository.Page{}, err
	}
	for i := range items {
		items[i].Extra = extra.BuildJSON(
			rowMeta[i].searchEngine,
			rowMeta[i].searchQuery,
			rowMeta[i].inspectError,
			kvByLog[items[i].ID],
		)
	}

	return repository.Page{
		Items: items,
		Total: int64(total),
		Page:  page,
		Size:  pageSize,
	}, nil
}

func (r *Repository) loadKVForLogs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID][]extra.KVRow, error) {
	out := make(map[uuid.UUID][]extra.KVRow)
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.conn.Query(ctx, `SELECT log_id, inspect_rule_id, field_key, field_value
		FROM proxy_access_log_inspect_kv
		WHERE log_id IN ?`, ids)
	if err != nil {
		return nil, fmt.Errorf("load inspect kv: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var logID, ruleID uuid.UUID
		var key, value string
		if err := rows.Scan(&logID, &ruleID, &key, &value); err != nil {
			return nil, err
		}
		out[logID] = append(out[logID], extra.KVRow{
			LogID:         logID,
			InspectRuleID: ruleID,
			FieldKey:      key,
			FieldValue:    value,
		})
	}
	return out, rows.Err()
}

func (r *Repository) DeleteBefore(ctx context.Context, before time.Time) error {
	before = before.UTC()
	if err := r.conn.Exec(ctx, `DELETE FROM proxy_access_log_inspect_kv WHERE created_at < ?`, before); err != nil {
		return fmt.Errorf("delete inspect kv before: %w", err)
	}
	if err := r.conn.Exec(ctx, `DELETE FROM proxy_access_log WHERE created_at < ?`, before); err != nil {
		return fmt.Errorf("delete access log before: %w", err)
	}
	return nil
}

func (r *Repository) DeleteBetween(ctx context.Context, from, to time.Time) (int64, error) {
	from = from.UTC()
	to = to.UTC()
	var n uint64
	if err := r.conn.QueryRow(ctx,
		`SELECT count() FROM proxy_access_log WHERE created_at >= ? AND created_at <= ?`,
		from, to,
	).Scan(&n); err != nil {
		return 0, fmt.Errorf("count delete range: %w", err)
	}
	if err := r.conn.Exec(ctx,
		`DELETE FROM proxy_access_log_inspect_kv WHERE created_at >= ? AND created_at <= ?`,
		from, to,
	); err != nil {
		return 0, fmt.Errorf("delete inspect kv between: %w", err)
	}
	if err := r.conn.Exec(ctx,
		`DELETE FROM proxy_access_log WHERE created_at >= ? AND created_at <= ?`,
		from, to,
	); err != nil {
		return 0, fmt.Errorf("delete access log between: %w", err)
	}
	return int64(n), nil
}

func nullableString(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

func nullableUUID(id *uuid.UUID) *uuid.UUID {
	return id
}
