package main

import (
	"flag"
	"io"
	"log"
	"os"
	"time"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/api/resource"
)

func main() {
	totalMem := flag.String("mem-total", "0", "memory that will be consumed")
	memStep := flag.String("mem-alloc-size", "4Ki", "amount of memory to be consumed in each allocation")
	memSleep := flag.Duration("mem-alloc-sleep", time.Millisecond, "sleep time between allocations")
	cpus := flag.Int("cpus", 0, "cpus to burn")
	flag.Parse()
	total := resource.MustParse(*totalMem)
	stepSize := resource.MustParse(*memStep)
	glog.Infof("Allocating %q memory, in %q chunks, with a %v sleep between allocations", total.String(), stepSize.String(), memSleep)
	takeMemory(stepSize, total, *memSleep)
	consumeCpus(*cpus)
	log.Printf("Allocated %q memory", total.String())
	select {}
}

func takeMemory(step, total resource.Quantity, howLong time.Duration) {
	var buffer [][]byte
	for i := int64(1); i*step.Value() <= total.Value(); i++ {
		newBuffer := make([]byte, step.Value())
		for i := range newBuffer {
			newBuffer[i] = 0
		}
		buffer = append(buffer, newBuffer)
		time.Sleep(howLong)
	}
}

func consumeCpus(howMany int) error {
	src, err := os.Open("/dev/zero")
	if err != nil {
		return err
	}
	for i := 0; i < howMany; i++ {
		log.Print("Spawning a go routine to consume CPU")
		go func() {
			_, err := io.Copy(io.Discard, src)
			if err != nil {
				panic(err)
			}
		}()
	}

	return nil
}
