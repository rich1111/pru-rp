# pru-rp
Go library for accessing the TI PRU using the
[RemoteProc](https://software-dl.ti.com/processor-sdk-linux/esd/docs/08_00_00_21/linux/Foundational_Components/PRU-ICSS/Linux_Drivers/RemoteProc.html)
framework. This API is used in more recent kernels.

godoc for this package is [available](https://pkg.go.dev/github.com/aamcrae/pru-rp).

## Examples

The examples use firmware from the [TI PRU Software Support package](https://git.ti.com/cgit/pru-software-support-package)
examples.
The firmware can be built from the examples and stored in /lib/firmware.
The examples used are:

| Example | Firmware file | Example source from package |
|---------|---------------|-----------------------------|
| echo | `am335x-pru0-echo0-fw` | `examples/am335x/PRU_RPMsg_Echo_Interrupt0` |

## Sample skeleton application

```
import "github.com/aamcrae/pru-rp"

func main() {
	p, _ := pru.Open(0)            // Open PRU0
	defer p.Close()
	p.Load("am335x-pru0-echo0-fw") // Load the firmware from /lib/firmware
	p.Callback(func (msg []byte) { // Add callback
		fmt.Printf("msg = [%s]\n", buf)
    })
	p.Start()                      // Start the PRU
	p.Send([]byte("Test string"))  // Send a message.
	time.Sleep(time.Second)        // Wait for message reply
}
```

## Disclaimer

This is not an officially supported Google product.
