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
	"fmt"
	"os"
	"time"
)

// Device paths etc.
const (
	RpBufSize = 512
	rpmDevBase = "/dev/rpmsg_pru3%d"
	rpBase = "/sys/class/remoteproc/remoteproc%d/%s"

    waitTimeout = 2 * time.Second
)

type PRU struct {
	unit int
	tx *os.File
	open bool
	running bool
	cb func ([]byte)
}

var prus = [...]PRU {
{ unit: 0, },
{ unit: 1, },
}

// Open initialises the PRU.
func Open(unit int) (* PRU, error) {
	if unit < 0 || unit >= len(prus) {
		return nil, fmt.Errorf("illegal unit")
	}
	p := &prus[unit]
	if !p.open {
		// On first open, ensure the PRU is stopped.
		p.open = true
		p.Stop()
	}
	return p, nil
}

// Close shuts down the PRU
func (p *PRU) Close() {
	p.Stop()
	p.open = false
}

// Stop writes the stop command to the PRU
func (p* PRU) Stop() error {
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
// the RPMsg device (if created).
func (p* PRU) Start() error {
	err := p.write("state", "start")
	if err == nil {
		p.running = true
		// Check for a RPMsg device being created.
		f, err := waitForPermission(fmt.Sprintf(rpmDevBase, p.unit))
		if err != nil {
			p.tx = nil
			if p.cb != nil {
				p.Stop()
				return fmt.Errorf("callback set, but no RPMsg device present")
			}
		} else {
			p.tx = f
			if p.cb != nil {
				// If a callback is set, start a go routine to read it.
				go func(cb func([]byte)) {
					defer f.Close()
					buf := make([]byte, RpBufSize)
					for {
						n, err := f.Read(buf)
						if err != nil {
							break;
						}
						cb(buf[0:n])
					}
				}(p.cb)
			}
		}
	}
	return err
}

// Send sends a message to this PRU via RPMsg
func (p *PRU) Send(buf []byte) error {
	if p.tx == nil {
		return fmt.Errorf("no RPMsg device opened")
	}
	if len(buf) >= RpBufSize {
		return fmt.Errorf("RPMsg buffer size too big")
	}
	_, err := p.tx.Write(buf)
	return err
}

// Callback sets the callback for any events
// This must be set before the PRU is started.
func (p *PRU) Callback(f func ([]byte)) error {
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
	f := fmt.Sprintf(rpBase, p.unit + 1, name)
	fd, err := os.OpenFile(f, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer fd.Close()
	_, err = fd.WriteString(command)
	return err
}

// After the RPMsg is created, there is a short time before the
// permissions get set correctly, so wait for the file to become
// writable.
func waitForPermission(name string) (*os.File, error) {
	var tout time.Duration
	var err error
	var f *os.File
	sl := time.Millisecond
	for tout = 0; tout < waitTimeout; tout += sl {
		f, err = os.OpenFile(name, os.O_RDWR, 0)
		if err == nil || !os.IsPermission(err) {
			break;
		}
		time.Sleep(sl)
	}
	return f, err
}
