package main

import (
	"os"
	"runtime/pprof"

	"orcaton/internal/cli"
)

func main() {
	if p := os.Getenv("ORCATON_CPUPROFILE"); p != "" {
		if f, err := os.Create(p); err == nil {
			_ = pprof.StartCPUProfile(f)
			code := cli.Run(os.Args[1:], os.Stdout, os.Stderr)
			pprof.StopCPUProfile()
			f.Close()
			os.Exit(code)
		}
	}
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
