package squid

import (
	"fmt"

	"oktopus/internal/proxy/acl"
)

func foreachPatternLine(def Line, fn func(lineNo int, line string) error) error {
	lineNo := 0
	if def.ACLListBody != "" {
		return acl.ForEachPatternLine(def.ACLListBody, func(line string) error {
			lineNo++
			return fn(lineNo, line)
		})
	}
	for _, line := range def.ACLValues {
		lineNo++
		if err := fn(lineNo, line); err != nil {
			return err
		}
	}
	if lineNo == 0 {
		return fmt.Errorf("нет значений")
	}
	return nil
}
