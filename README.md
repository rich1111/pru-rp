# pru-rp
Go library for accessing the TI PRU using the
[RemoteProc](https://software-dl.ti.com/processor-sdk-linux/esd/docs/08_00_00_21/linux/Foundational_Components/PRU-ICSS/Linux_Drivers/RemoteProc.html)
framework. This API is used in more recent kernels.

## Examples

The examples use firmware from the [TI PRU Software Support package](https://git.ti.com/cgit/pru-software-support-package)
examples.
The firmware can be built from the examples and stored in /lib/firmware.
The examples used are:

| Example | Firmware file | Example source from package |
|---------|---------------|-----------------------------|
| echo | `am335x-pru0-echo0-fw` | `examples/am335x/PRU_RPMsg_Echo_Interrupt0` |

## Disclaimer

This is not an officially supported Google product.
