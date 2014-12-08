package main

import (
	"flag"
	"fmt"
	client "github.com/whyrusleeping/go-tftp/client"
	"io"
	"runtime"
	"sync"
	"time"
)

func benchReads(server, file string, threads, loops int) {
	wg := &sync.WaitGroup{}

	bwcollect := make(chan int, 32)
	before := time.Now()

	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cli, err := client.NewTftpClient(server)
			if err != nil {
				panic(err)
			}

			for j := 0; j < loops; j++ {
				nbytes, _, err := cli.GetFile(file)
				if err != nil {
					panic(err)
				}
				bwcollect <- nbytes

			}
		}()
	}

	go func() {
		wg.Wait()
		close(bwcollect)
	}()

	sum := 0
	for bw := range bwcollect {
		sum += bw
	}
	took := time.Now().Sub(before)

	fmt.Printf("Total Transferred: %d\n", sum)
	fmt.Printf("Overall Bandwidth: %.0f Bps\n", float64(sum)/took.Seconds())
}

func benchWrites(server string, threads, loops, nbytes int) {
	wg := &sync.WaitGroup{}

	bwcollect := make(chan int, 32)
	before := time.Now()

	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func(thr int) {
			defer wg.Done()
			cli, err := client.NewTftpClient(server)
			if err != nil {
				panic(err)
			}

			for j := 0; j < loops; j++ {
				read := io.LimitReader(DataReader{}, int64(nbytes))
				nbytes, err := cli.PutFile(fmt.Sprintf("file%d-%d", thr, j), read)
				if err != nil {
					panic(err)
				}
				bwcollect <- nbytes

			}
		}(i)
	}

	go func() {
		wg.Wait()
		close(bwcollect)
	}()

	sum := 0
	for bw := range bwcollect {
		sum += bw
	}
	took := time.Now().Sub(before)

	fmt.Printf("Total Transferred: %d\n", sum)
	fmt.Printf("Overall Bandwidth: %.0f Bps\n", float64(sum)/took.Seconds())
}

func main() {
	nprocs := flag.Int("procs", 1, "number of procs to run")
	nthreads := flag.Int("threads", 1, "number of threads to run")
	nloops := flag.Int("loops", 1, "number of operations per thread")
	serv := flag.String("serv", "127.0.0.1:6900", "address of server to benchmark")
	filename := flag.String("file", "", "name of file to work with (for reads only)")
	upload := flag.Int("upload", -1, "size of data for upload testing")

	flag.Parse()

	runtime.GOMAXPROCS(*nprocs)
	_ = nthreads
	_ = upload

	fmt.Printf("Testing Server: '%s'\n", *serv)

	if *upload > 0 {
		benchWrites(*serv, *nthreads, *nloops, *upload)
	} else {
		benchReads(*serv, *filename, *nthreads, *nloops)
	}
}
