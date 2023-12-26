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

//go:generate make -C am335x/PRU_Halt
//go:generate sudo cp am335x/PRU_Halt/gen/PRU_Halt.out /lib/firmware/am335x-pru0-halt-fw

package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/aamcrae/pru-rp"
)

func main() {
	p, err := pru.Open(0)
	if err != nil {
		log.Fatalf("%s", err)
	}
	defer p.Close()
	err = p.Load("am335x-pru0-halt-fw")
	if err != nil {
		log.Fatalf("Load f/w: %v", err)
	}

	current := p.Status()
	log.Printf("PRU0 state: %s", current.String())
	p.Start(true)
	go func() {
		// Sleep between 1 and 4 seconds
		time.Sleep(time.Duration(rand.Int63n(3000)) * time.Millisecond + time.Second)
		p.Stop()
	}()
	now := time.Now()
	for i := 0; i < 1000; i++ {
		s := p.Status()
		if s != current {
			log.Printf("PRU0 state: %s", s.String())
			current = s
		}
		if s != pru.StatusRunning {
			log.Printf("PRU halted after %s", time.Now().Sub(now))
			break;
		}
		time.Sleep(10 * time.Millisecond)
	}
}
