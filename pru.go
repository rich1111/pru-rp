// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pru

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

type Status int

// Status values
const (
	StatusOffline Status = iota
	StatusRunning
	StatusUnknown
)

// Device paths etc.
const (
	RpBufSize  = 512
	rpmDevBase = "/dev/rpmsg_pru3%d"
	rpBase     = "/sys/class/remoteproc/remoteproc%d/%s"
	unitOffset = 2 // Remoteproc units start at 2 for TI AM62x PRU
)

// AM3xx
// Memory values
const (
	am3xxPru0Ram       = 0x00000000
	am3xxPru1Ram       = 0x00002000
	am3xxSharedRam     = 0x00010000
	am3xxRamSize       = 8 * 1024
	am3xxSharedRamSize = 12 * 1024

	am3xxAddress = 0x4A300000
	am3xxSize    = 0x80000
)

const (
	waitTimeout = 5 * time.Second
)

var Order = binary.LittleEndian

type ram []byte

type PRU struct {
	unit    int
	tx      *os.File
	open    bool
	running bool
	cb      func([]byte)

	// These are set if /dev/mem is accessible
	mmapFile *os.File
	mem      []byte
}

var prus = [...]PRU{
	{unit: 0},
	{unit: 1},
}

// Open initialises the PRU.
func Open(unit int) (*PRU, error) {
	if unit < 0 || unit >= len(prus) {
		return nil, fmt.Errorf("illegal unit")
	}
	p := &prus[unit]
	if !p.open {
		// On first open, ensure the PRU is stopped, and set up
		// the shared memory mappings (if accessible).
		p.Stop()
		p.open = true
		p.mmapFile = nil
		p.mem = nil
	}
	return p, nil
}

// Ram creates a type that can use a Reader/Writer interface to the
// underlying byte array of this PRU's RAM.
func (p *PRU) Ram() (*RamIO, error) {
	if err := p.openRam(); err != nil {
		return nil, err
	}
	var base ram
	if p.unit == 0 {
		base = p.mem[am3xxPru0Ram : am3xxPru0Ram+am3xxRamSize]
	} else {
		base = p.mem[am3xxPru1Ram : am3xxPru1Ram+am3xxRamSize]
	}
	return &RamIO{Memory: base, max: cap(base)}, nil
}

// SharedRam creates a type that can use a Reader/Writer interface to the
// underlying byte array for the shared PRU RAM.
func (p *PRU) SharedRam() (*RamIO, error) {
	if err := p.openRam(); err != nil {
		return nil, err
	}
	base := p.mem[am3xxSharedRam : am3xxSharedRam+am3xxSharedRamSize]
	return &RamIO{Memory: base, max: cap(base)}, nil
}

// openRam initialises the access to the PRU RAM and shared RAM.
func (p *PRU) openRam() error {
	// Check whether the RAM is already open.
	if p.mem != nil {
		return nil
	}
	// Attempt to access the shared memory via /dev/mem.
	m, err := os.OpenFile("/dev/mem", os.O_RDWR|os.O_SYNC, 0660)
	if err != nil {
		return err
	}
	mem, err := unix.Mmap(int(m.Fd()), am3xxAddress, am3xxSize, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		return err
	}
	p.mmapFile = m
	p.mem = mem
	return nil
}

// Close shuts down the PRU
func (p *PRU) Close() {
	p.Stop()
	p.open = false
	if p.mmapFile != nil {
		unix.Munmap(p.mem)
		p.mmapFile.Close()
		p.mmapFile = nil
		p.mem = nil
	}
}

// Status returns the current status of the PRU
func (p *PRU) Status() Status {
	f := fmt.Sprintf(rpBase, p.unit+unitOffset, "state")
	fd, err := os.OpenFile(f, os.O_RDONLY, 0)
	if err != nil {
		return StatusUnknown
	}
	defer fd.Close()
	s, err := io.ReadAll(fd)
	if err != nil {
		return StatusUnknown
	}
	switch string(s) {
	case "offline\n":
		return StatusOffline
	case "running\n":
		return StatusRunning
	default:
		return StatusUnknown
	}
}

// Stop writes the stop command to the PRU
func (p *PRU) Stop() error {
	if p.tx != nil {
		p.tx.Close()
		p.tx = nil
	}
	err := p.write("state", "stop")
	if err == nil {
		p.running = false
	}
	return err
}

// Start writes the start command to the PRU, and sets up
// the RPMsg device (if required). rpmsg is set if
// the PRU requires the RPMsg virtual device.
func (p *PRU) Start(rpmsg bool) error {
	err := p.write("state", "start")
	if err == nil {
		p.tx = nil
		if rpmsg {
			// Check for a RPMsg device being created.
			name := fmt.Sprintf(rpmDevBase, p.unit)
			f, err := waitForPermission(name)
			if err != nil {
				return fmt.Errorf("rpmsg %s: %v", name, err)
			}
			p.tx = f
			if p.cb != nil {
				// If a callback is set, start a go routine to read it.
				go func(cb func([]byte)) {
					buf := make([]byte, RpBufSize)
					for {
						n, err := f.Read(buf)
						if err != nil {
							break
						}
						cb(buf[0:n])
					}
				}(p.cb)
			}
		}
		p.running = true
	}
	return err
}

// Send sends a message to this PRU via RPMsg
func (p *PRU) Send(buf []byte) error {
	if p.tx == nil {
		return fmt.Errorf("no RPMsg device")
	}
	if len(buf) >= RpBufSize {
		return fmt.Errorf("RPMsg buffer size too big")
	}
	_, err := p.tx.Write(buf)
	return err
}

// Callback sets the callback for any events
// This must be set before the PRU is started.
func (p *PRU) Callback(f func([]byte)) error {
	if p.running {
		return fmt.Errorf("Cannot install callback after PRU has started")
	}
	p.cb = f
	return nil
}

// Load writes the name of the firmware to be loaded.
// This is a file that resides in /lib/firmware.
// If the PRU is currently running, it is stopped first.
func (p *PRU) Load(name string) error {
	if p.running {
		p.Stop()
	}
	return p.write("firmware", name)
}

// write sends the string to the remoteproc filename
func (p *PRU) write(name, command string) error {
	f := fmt.Sprintf(rpBase, p.unit+unitOffset, name)
	fd, err := os.OpenFile(f, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer fd.Close()
	_, err = fd.WriteString(command)
	return err
}

// After the RPMsg vdev is created, there is a short time before the
// permissions get set correctly, so wait for the device to become writable.
func waitForPermission(name string) (*os.File, error) {
	var tout time.Duration
	var err error
	var f *os.File
	sl := time.Millisecond
	for tout = 0; tout < waitTimeout; tout += sl {
		f, err = os.OpenFile(name, os.O_RDWR, 0)
		if err == nil || !os.IsPermission(err) {
			break
		}
		time.Sleep(sl)
	}
	return f, err
}

// String returns string value of status.
func (s Status) String() string {
	return [...]string{"Offline", "Running", "Unknown"}[s]
}
