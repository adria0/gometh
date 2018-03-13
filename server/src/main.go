package main

/*

geth --dev console --ws --networkid 1337

*/

import (
	"gometh"

	"github.com/CrowdSurge/banner"
)

func main() {
	banner.Print("gometh")
	gometh.ExecuteCmd()
}
