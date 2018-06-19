package main

import (
	"flag"
	"github.com/whyrusleeping/go-tftp/server"
	"io"
	"os"
)

func reader(path string) (r io.Reader, err error) {
	r, err = os.Open(path)
	return
}

func writer(path string) (w io.Writer, err error) {
	w, err = os.Create(path)
	return
}

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	dir := flag.String("dir", cwd, "specify a directory to serve files from")
	port := flag.String("port", "6900", "specify a port to listen on")
	address := flag.String("address", "", "specify address to listen on")
	flag.Parse()

	srv := server.NewServer(*dir, reader, writer)
	panic(srv.Serve(*address + ":" + *port))
}
