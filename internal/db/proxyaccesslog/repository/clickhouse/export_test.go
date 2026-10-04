package clickhouse

import "oktopus/internal/db/proxyaccesslog/repository"

// ListWhereForTest — SQL WHERE для unit-тестов.
func ListWhereForTest(f repository.ListFilter) (string, []any) {
	return listWhere(f)
}
