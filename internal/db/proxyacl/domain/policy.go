package domain

import "time"

// Policy — текст squid-конфига (acl + http_access в конфиге).
type Policy struct {
	ConfigText string
	UpdatedAt  time.Time
}
