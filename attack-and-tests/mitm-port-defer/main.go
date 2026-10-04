// PoC: CONNECT :443 (defer PORT) vs HTTP URL на запрещённый порт.
package main

import "oktopus/attack-and-tests/poclib"

func main() {
	poclib.MainCLI("MITM port defer bypass", cmdAll, cmdLab, cmdRun)
}
