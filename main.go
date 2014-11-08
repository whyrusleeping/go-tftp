package main

import (
	"github.com/whyrusleeping/tftp/server"
	"os"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	srv := server.NewServer(cwd)
	panic(srv.Serve(":6969"))
}
