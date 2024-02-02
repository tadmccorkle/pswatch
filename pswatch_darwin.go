package main

// #include <libproc.h>
// #include <pwd.h>
import "C"
import (
	"fmt"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
)

func getProcs() []Proc {
	kinfoProcs, err := unix.SysctlKinfoProcSlice("kern.proc.all")
	if err != nil {
		fmt.Printf("Error getting processes: %v\n", err)
		return make([]Proc, 0)
	}

	procs := make([]Proc, 0, len(kinfoProcs))
	cPath := make([]C.char, C.PROC_PIDPATHINFO_MAXSIZE)

	for _, r := range kinfoProcs {
		ret, err := C.proc_pidpath(C.int(r.Proc.P_pid), unsafe.Pointer(&cPath[0]), C.PROC_PIDPATHINFO_MAXSIZE)
		if ret < 0 {
			fmt.Printf("Error getting process path for '%v': %v\n", r.Proc.P_pid, err)
			continue
		}

		path := C.GoString(&cPath[0])
		exeIdx := strings.LastIndex(path, "/") + 1
		pwuid := C.getpwuid(C.uint(r.Eproc.Ucred.Uid))

		procs = append(procs, Proc{
			Pid:    int(r.Proc.P_pid),
			Path:   path,
			Exe:    path[exeIdx:],
			Parent: int(r.Eproc.Ppid),
			User:   C.GoString(pwuid.pw_name),
		})
	}
	return procs
}
