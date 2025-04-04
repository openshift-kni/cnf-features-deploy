package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
	"k8s.io/klog/v2"
)

const (
	// MapHugeShift Shift for HugePage size
	MapHugeShift = 26
	// DefaultHugePageSize 1GB HugePage size
	DefaultHugePageSize = 1 * 1024 * 1024 * 1024
)

// MAP_HUGE_1GB 1GB HugePage mmap flag
const MAP_HUGE_1GB = 30 << MapHugeShift

type Args struct {
	TimeDuration time.Duration
	HugePageSize int
}

func main() {
	klog.InitFlags(nil)
	args := &Args{}
	flag.DurationVar(&args.TimeDuration, "time-duration", math.MaxInt64, "set the time duration for program to wait - wait forever by default")
	flag.IntVar(&args.HugePageSize, "hugepage-size", DefaultHugePageSize, "hugepage size to allocate - allocate 1G by default")

	flag.Parse()

	// Flags for HugePage allocation
	mmapFlags := unix.MAP_PRIVATE | unix.MAP_ANONYMOUS | unix.MAP_HUGETLB | MAP_HUGE_1GB
	// Use mmap to allocate memory
	addr, _, errno := unix.Syscall6(
		unix.SYS_MMAP,
		0,                          // Let the kernel choose the address
		uintptr(args.HugePageSize), // Size of the memory
		uintptr(unix.PROT_READ|unix.PROT_WRITE), // Read/Write permissions
		uintptr(mmapFlags),                      // mmap flags
		0,                                       // File descriptor (not used for anonymous memory)
		0,                                       // Offset
	)
	if errno != 0 {
		klog.ErrorS(fmt.Errorf("errno=%v", errno), "Failed to allocate HugePage")
		os.Exit(1)
	}
	memory := unsafe.Pointer(addr)
	// Write a byte to the allocated memory
	*(*byte)(memory) = 42

	klog.InfoS("Successfully allocated 1GB HugePage memory", "address", fmt.Sprintf("%p", unsafe.Pointer(addr)))

	// Cleanup: Unmap the memory
	defer func() {
		_, _, errno = unix.Syscall(unix.SYS_MUNMAP, addr, uintptr(args.HugePageSize), 0)
		if errno != 0 {
			klog.ErrorS(fmt.Errorf("errno=%v", errno), "Failed to unmap HugePage")
			os.Exit(2)
		}
		klog.InfoS("1GB HugePage memory unmapped successfully")
	}()
	wait(args.TimeDuration)
}

func wait(timeout time.Duration) {
	// Create a channel to listen for signals.
	signalChan := make(chan os.Signal, 1)

	// SIGINT handles Ctrl+C locally.
	// SIGTERM handles Cloud Run termination signal.
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	klog.InfoS("Waiting", "period", timeout.String())
	select {
	// Receive output from signalChan.
	case sig := <-signalChan:
		klog.InfoS("signal caught", "signal", sig)
	case <-time.After(timeout):
		klog.InfoS("Done")
		return
	}
}
