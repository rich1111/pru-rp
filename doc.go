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

/*

Package pru-rp provides a Go library to access the Programmable Real-time Units (PRU)
of the TI AM335x (https://www.ti.com/processors/sitara-arm/applications/industrial-communications.html)
The commonly available product with this part is the Beaglebone Black (https://beagleboard.org/black)

The RemoteProc (https://software-dl.ti.com/processor-sdk-linux/esd/docs/08_00_00_21/linux/Foundational_Components/PRU-ICSS/Linux_Drivers/RemoteProc.html)
framework is used, which is standard on recent Linux kernels, and optional on
earlier (<4.19) kernels

This package does not include any support for developing or building the PRU firmware;
for that, the standard TI PRU S/W support package should be used
(https://git.ti.com/cgit/pru-software-support-package).

Complete documentation is available via https://github.com/aamcrae/pru-rp, and through godoc at
https://pkg.go.dev/github.com/aamcrae/pru-rp

*/
package pru
