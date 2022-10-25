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

//go:generate make

package main

import (
	"fmt"
	"log"
	"sync"

	"github.com/aamcrae/pru-rp"
)

func main() {
	p, err := pru.Open(0)
	if err != nil {
		log.Fatalf("%s", err)
	}
	defer p.Close()
	err = p.Load("am335x-pru0-echo0-fw")
	if err != nil {
		log.Fatalf("Load: %v", err)
	}

	var counter sync.WaitGroup
	p.Callback(func(msg []byte) {
		log.Printf("Rx bytes = [%s]", msg)
		counter.Done()
	})
	p.Start(true)
	for i := 0; i < 10; i++ {
		err := p.Send([]byte(fmt.Sprintf("test %d", i)))
		if err != nil {
			log.Printf("Send: %v", err)
		} else {
			counter.Add(1)
		}
	}
	counter.Wait()
}
