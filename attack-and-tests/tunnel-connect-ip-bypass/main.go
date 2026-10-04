// PoC: CONNECT по литералу IP в обход acl dstdomain (нужен deny dst).
package main

import "oktopus/attack-and-tests/poclib"

func main() {
	poclib.MainCLI("CONNECT IP bypass dstdomain", cmdAll, cmdLab, cmdRun)
}
