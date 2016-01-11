package main

import (
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"time"

	"k8s.io/kubernetes/cmd/hype-apiserver/app"
	"k8s.io/kubernetes/pkg/util"
	"k8s.io/kubernetes/pkg/version/verflag"

	"github.com/spf13/pflag"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	rand.Seed(time.Now().UTC().UnixNano())

	s := app.NewAPIServer()
	s.AddFlags(pflag.CommandLine)

	util.InitFlags()
	util.InitLogs()
	defer util.FlushLogs()

	verflag.PrintAndExitIfRequested()

	if err := s.Run(pflag.CommandLine.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
