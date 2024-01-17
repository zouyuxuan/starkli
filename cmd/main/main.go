package main

import (
	"github.com/buildpacks/libcnb"
	"github.com/paketo-buildpacks/libpak/bard"
	"os"
	"starkli/starkli"
)

func main() {
	libcnb.Main(
		starkli.Detect{},
		starkli.Build{Logger: bard.NewLogger(os.Stdout)},
	)
}
