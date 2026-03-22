package main

import (
	"os"
	"syscall"

	"github.com/openimsdk/gomake/mageutil"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "check" {
		mageutil.CheckAndReportBinariesStatus()
		return
	}
	mageutil.InitForSSC()
	var rlimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlimit); err == nil {
		rlimit.Max = uint64(mageutil.MaxFileDescriptors)
		rlimit.Cur = uint64(mageutil.MaxFileDescriptors)
		_ = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlimit)
	}
	mageutil.StartToolsAndServices(nil, nil)
}
