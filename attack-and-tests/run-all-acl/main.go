package main

import (
	"log"
	"os"

	"oktopus/attack-and-tests/aclrun"
)

func main() {
	log.SetFlags(0)
	args := os.Args[1:]
	failed := false

	for _, item := range aclrun.AllSuites() {
		if item.Eval != nil {
			s := *item.Eval
			log.Printf("=== %s ===", s.Name)
			if aclrun.RunEvaluateSuite(s, args) != 0 {
				failed = true
			}
			log.Println("")
			continue
		}
		if item.Compile != nil {
			c := *item.Compile
			log.Printf("=== %s ===", c.Name)
			if aclrun.RunCompileSuite(c, args) != 0 {
				failed = true
			}
			log.Println("")
			continue
		}
		if item.Auth != nil {
			a := *item.Auth
			log.Printf("=== %s ===", a.Name)
			if aclrun.RunProxyAuthSuite(a, args) != 0 {
				failed = true
			}
			log.Println("")
			continue
		}
		if item.Inspect != nil {
			ins := *item.Inspect
			log.Printf("=== %s ===", ins.Name)
			if aclrun.RunInspectSuite(ins, args) != 0 {
				failed = true
			}
			log.Println("")
		}
	}

	if failed {
		log.Println("RESULT: FAIL")
		os.Exit(1)
	}
	log.Println("RESULT: OK")
}
