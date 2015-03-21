package main

import (
	"flag"
	"github.com/whyrusleeping/go-tftp/server"
	"os"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	dir := flag.String("dir", cwd, "specify a directory to serve files from")
	port := flag.String("port", "6900", "specify a port to listen on")
	flag.Parse()

	srv := server.NewServer(*dir)
	panic(srv.Serve(":" + *port))
}
