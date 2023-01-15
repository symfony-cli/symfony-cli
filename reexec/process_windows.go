/*
 * Copyright (c) 2021-present Fabien Potencier <fabien@symfony.com>
 *
 * This file is part of Symfony CLI project
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <http://www.gnu.org/licenses/>.
 */

package reexec

import (
	"os"
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/symfony-cli/terminal"
)

func Getppid() int {
	ppid := os.Getppid()

	// MinGW is so shitty that it messes with process tree
	// to get the parent PID we actually need to look at great-parent one
	// otherwise we get a process that is already exited!
	if terminal.IsCygwinTTY(os.Stdout.Fd()) {
		parent, err := findProcess(ppid)
		if err != nil {
			panic(err)
		}
		ppid = parent.PPid()
	}

	return ppid
}

func waitForProcess(ps *os.Process) {
	ps.Wait()

	return
}

// The following code is *highly* inspired from
// https://github.com/mitchellh/go-ps/blob/4fdf99ab29366514c69ccccddab5dc58b8d84062/process_windows.go

// Windows API functions
var (
	modKernel32                  = syscall.NewLazyDLL("kernel32.dll")
	procCloseHandle              = modKernel32.NewProc("CloseHandle")
	procCreateToolhelp32Snapshot = modKernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32First           = modKernel32.NewProc("Process32FirstW")
	procProcess32Next            = modKernel32.NewProc("Process32NextW")
)

// Some constants from the Windows API
const (
	max_path = 260
)

// processentry32 is the Windows API structure that contains a process's
// information.
type processentry32 struct {
	Size              uint32
	CntUsage          uint32
	ProcessID         uint32
	DefaultHeapID     uintptr
	ModuleID          uint32
	CntThreads        uint32
	ParentProcessID   uint32
	PriorityClassBase int32
	Flags             uint32
	ExeFile           [max_path]uint16
}

// windowsProcess is an implementation of Process for Windows.
type windowsProcess struct {
	pid  int
	ppid int
	exe  string
}

func (p *windowsProcess) Pid() int {
	return p.pid
}

func (p *windowsProcess) PPid() int {
	return p.ppid
}

func (p *windowsProcess) Executable() string {
	return p.exe
}

func newWindowsProcess(e *processentry32) *windowsProcess {
	// Find when the string ends for decoding
	end := 0
	for {
		if e.ExeFile[end] == 0 {
			break
		}
		end++
	}

	return &windowsProcess{
		pid:  int(e.ProcessID),
		ppid: int(e.ParentProcessID),
		exe:  syscall.UTF16ToString(e.ExeFile[:end]),
	}
}

func findProcess(pid int) (*windowsProcess, error) {
	ps, err := processes()
	if err != nil {
		return nil, err
	}

	for _, p := range ps {
		if p.Pid() == pid {
			return p, nil
		}
	}

	return nil, nil
}

func processes() ([]*windowsProcess, error) {
	handle, _, _ := procCreateToolhelp32Snapshot.Call(
		0x00000002,
		0)
	if handle < 0 {
		return nil, errors.WithStack(syscall.GetLastError())
	}
	defer procCloseHandle.Call(handle)

	var entry processentry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	ret, _, _ := procProcess32First.Call(handle, uintptr(unsafe.Pointer(&entry)))
	if ret == 0 {
		return nil, errors.Errorf("Error retrieving process info.")
	}

	results := make([]*windowsProcess, 0, 50)
	for {
		results = append(results, newWindowsProcess(&entry))

		ret, _, _ := procProcess32Next.Call(handle, uintptr(unsafe.Pointer(&entry)))
		if ret == 0 {
			break
		}
	}

	return results, nil
}
