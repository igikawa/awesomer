package mutation

import (
	"errors"
	"testing"

	"golang.org/x/sys/unix"
)

func TestSetPRlimitUsesInjectedSyscall(t *testing.T) {
	orig := prlimitFn
	defer func() { prlimitFn = orig }()

	var (
		gotPID   int
		gotLimit int
		gotCur   uint64
		gotMax   uint64
	)

	prlimitFn = func(pid int, resource int, newlimit, oldlimit *unix.Rlimit) error {
		gotPID = pid
		gotLimit = resource
		gotCur = newlimit.Cur
		gotMax = newlimit.Max
		return nil
	}

	if err := SetPRlimit(10, unix.RLIMIT_NOFILE, 12, 14); err != nil {
		t.Fatalf("SetPRlimit() error = %v", err)
	}
	if gotPID != 10 || gotLimit != unix.RLIMIT_NOFILE || gotCur != 12 || gotMax != 14 {
		t.Fatalf("captured values pid=%d limit=%d cur=%d max=%d", gotPID, gotLimit, gotCur, gotMax)
	}
}

func TestSetCPUaffinityUsesInjectedSyscall(t *testing.T) {
	orig := schedSetaffinityFn
	defer func() { schedSetaffinityFn = orig }()

	var (
		gotPID int
		has1   bool
		has3   bool
	)

	schedSetaffinityFn = func(pid int, set *unix.CPUSet) error {
		gotPID = pid
		has1 = set.IsSet(1)
		has3 = set.IsSet(3)
		return nil
	}

	if err := SetCPUaffinity(99, []int{1, 3}); err != nil {
		t.Fatalf("SetCPUaffinity() error = %v", err)
	}
	if gotPID != 99 || !has1 || !has3 {
		t.Fatalf("captured pid=%d has1=%v has3=%v", gotPID, has1, has3)
	}
}

func TestSetCPUattrUsesInjectedSyscall(t *testing.T) {
	orig := schedSetAttrFn
	defer func() { schedSetAttrFn = orig }()

	attr := &unix.SchedAttr{Policy: 7, Nice: 5}
	var gotPID int
	var capturedAttr *unix.SchedAttr

	schedSetAttrFn = func(pid int, passedAttr *unix.SchedAttr, flags uint) error {
		gotPID = pid
		capturedAttr = passedAttr
		return nil
	}

	if err := SetCPUattr(55, attr); err != nil {
		t.Fatalf("SetCPUattr() error = %v", err)
	}
	if gotPID != 55 || capturedAttr != attr {
		t.Fatalf("captured pid=%d attr=%p want attr=%p", gotPID, capturedAttr, attr)
	}
}

func TestSetCPUattrPropagatesErrors(t *testing.T) {
	orig := schedSetAttrFn
	defer func() { schedSetAttrFn = orig }()

	schedSetAttrFn = func(pid int, gotAttr *unix.SchedAttr, flags uint) error {
		return errors.New("boom")
	}

	if err := SetCPUattr(1, &unix.SchedAttr{}); err == nil {
		t.Fatal("SetCPUattr() error = nil, want non-nil")
	}
}

func TestGetPRlimitUsesInjectedSyscall(t *testing.T) {
	orig := prlimitFn
	defer func() { prlimitFn = orig }()

	prlimitFn = func(pid int, resource int, newlimit, oldlimit *unix.Rlimit) error {
		oldlimit.Cur = 128
		oldlimit.Max = 256
		return nil
	}

	cur, max, err := GetPRlimit(3, unix.RLIMIT_NOFILE)
	if err != nil {
		t.Fatalf("GetPRlimit() error = %v", err)
	}
	if cur != 128 || max != 256 {
		t.Fatalf("GetPRlimit() = (%d, %d), want (128, 256)", cur, max)
	}
}

func TestGetCPUaffinityUsesInjectedSyscall(t *testing.T) {
	orig := schedGetaffinityFn
	defer func() { schedGetaffinityFn = orig }()

	schedGetaffinityFn = func(pid int, set *unix.CPUSet) error {
		set.Set(0)
		set.Set(2)
		set.Set(5)
		return nil
	}

	cores, err := GetCPUaffinity(7)
	if err != nil {
		t.Fatalf("GetCPUaffinity() error = %v", err)
	}
	if len(cores) != 3 || cores[0] != 0 || cores[1] != 2 || cores[2] != 5 {
		t.Fatalf("GetCPUaffinity() = %v, want [0 2 5]", cores)
	}
}
