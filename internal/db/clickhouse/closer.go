package clickhouse

// Closer — соединение ClickHouse (закрытие при остановке процесса).
type Closer interface {
	Close() error
}
