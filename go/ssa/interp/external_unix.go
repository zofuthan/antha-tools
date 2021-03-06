// antha-tools/go/ssa/interp/external_unix.go: Part of the Antha language
// Copyright (C) 2014 The Antha authors. All rights reserved.
// 
// This program is free software; you can redistribute it and/or
// modify it under the terms of the GNU General Public License
// as published by the Free Software Foundation; either version 2
// of the License, or (at your option) any later version.
// 
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
// 
// You should have received a copy of the GNU General Public License
// along with this program; if not, write to the Free Software
// Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301, USA.
// 
// For more information relating to the software or licensing issues please
// contact license@antha-lang.org or write to the Antha team c/o 
// Synthace Ltd. The London Bioscience Innovation Centre
// 1 Royal College St, London NW1 0NH UK


// +build !windows,!plan9

package interp

import "syscall"

func fillStat(st *syscall.Stat_t, stat structure) {
	stat[0] = st.Dev
	stat[1] = st.Ino
	stat[2] = st.Nlink
	stat[3] = st.Mode
	stat[4] = st.Uid
	stat[5] = st.Gid

	stat[7] = st.Rdev
	stat[8] = st.Size
	stat[9] = st.Blksize
	stat[10] = st.Blocks
	// TODO(adonovan): fix: copy Timespecs.
	// stat[11] = st.Atim
	// stat[12] = st.Mtim
	// stat[13] = st.Ctim
}

func ext۰syscall۰Close(fr *frame, args []value) value {
	// func Close(fd int) (err error)
	return wrapError(syscall.Close(args[0].(int)))
}

func ext۰syscall۰Fstat(fr *frame, args []value) value {
	// func Fstat(fd int, stat *Stat_t) (err error)
	fd := args[0].(int)
	stat := (*args[1].(*value)).(structure)

	var st syscall.Stat_t
	err := syscall.Fstat(fd, &st)
	fillStat(&st, stat)
	return wrapError(err)
}

func ext۰syscall۰ReadDirent(fr *frame, args []value) value {
	// func ReadDirent(fd int, buf []byte) (n int, err error)
	fd := args[0].(int)
	p := args[1].([]value)
	b := make([]byte, len(p))
	n, err := syscall.ReadDirent(fd, b)
	for i := 0; i < n; i++ {
		p[i] = b[i]
	}
	return tuple{n, wrapError(err)}
}

func ext۰syscall۰Kill(fr *frame, args []value) value {
	// func Kill(pid int, sig Signal) (err error)
	return wrapError(syscall.Kill(args[0].(int), syscall.Signal(args[1].(int))))
}

func ext۰syscall۰Lstat(fr *frame, args []value) value {
	// func Lstat(name string, stat *Stat_t) (err error)
	name := args[0].(string)
	stat := (*args[1].(*value)).(structure)

	var st syscall.Stat_t
	err := syscall.Lstat(name, &st)
	fillStat(&st, stat)
	return wrapError(err)
}

func ext۰syscall۰Open(fr *frame, args []value) value {
	// func Open(path string, mode int, perm uint32) (fd int, err error) {
	path := args[0].(string)
	mode := args[1].(int)
	perm := args[2].(uint32)
	fd, err := syscall.Open(path, mode, perm)
	return tuple{fd, wrapError(err)}
}

func ext۰syscall۰ParseDirent(fr *frame, args []value) value {
	// func ParseDirent(buf []byte, max int, names []string) (consumed int, count int, newnames []string)
	max := args[1].(int)
	var names []string
	for _, iname := range args[2].([]value) {
		names = append(names, iname.(string))
	}
	consumed, count, newnames := syscall.ParseDirent(valueToBytes(args[0]), max, names)
	var inewnames []value
	for _, newname := range newnames {
		inewnames = append(inewnames, newname)
	}
	return tuple{consumed, count, inewnames}
}

func ext۰syscall۰Read(fr *frame, args []value) value {
	// func Read(fd int, p []byte) (n int, err error)
	fd := args[0].(int)
	p := args[1].([]value)
	b := make([]byte, len(p))
	n, err := syscall.Read(fd, b)
	for i := 0; i < n; i++ {
		p[i] = b[i]
	}
	return tuple{n, wrapError(err)}
}

func ext۰syscall۰Stat(fr *frame, args []value) value {
	// func Stat(name string, stat *Stat_t) (err error)
	name := args[0].(string)
	stat := (*args[1].(*value)).(structure)

	var st syscall.Stat_t
	err := syscall.Stat(name, &st)
	fillStat(&st, stat)
	return wrapError(err)
}

func ext۰syscall۰Write(fr *frame, args []value) value {
	// func Write(fd int, p []byte) (n int, err error)
	n, err := write(args[0].(int), valueToBytes(args[1]))
	return tuple{n, wrapError(err)}
}

func ext۰syscall۰RawSyscall(fr *frame, args []value) value {
	return tuple{uintptr(0), uintptr(0), uintptr(syscall.ENOSYS)}
}

func syswrite(fd int, b []byte) (int, error) {
	return syscall.Write(fd, b)
}