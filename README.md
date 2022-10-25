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
	p.Start(true)                 // Start the PRU (RPMsg required)
	p.Send([]byte("Test string"))  // Send a message.
	time.Sleep(time.Second)        // Wait for message reply
}
```

## Accessing Shared Memory

The host CPU can access the various RAM blocks on the PRU subsystem, such as the PRU unit 0 and 1 8KB RAM
and the 12KB shared RAM. These RAM blocks are exported as byte slices (```[]byte```) initialised over the
RAM block as a byte array. The memory is accessed via ```/dev/mem```, so to use this facility, the program must
have r/w permission to this device. This can be done either by running the program via ```sudo```, or by changing
the ```/dev/mem``` permissions (e.g to 0660 and ensure that the program user account belongs to group ```kmem```).

There are a number of ways that applications can access the shared memory as structured access.
For ease of access, the package exports a variable ```Order``` (as a ```binary/encoding Order```).
This allows use of the ```binary/encoding``` package:

```
	p := pru.Open()
	pru.Order.PutUint32(p.Ram[0:], word1)
	pru.Order.PutUint32(p.Ram[4:], word2)
	pru.Order.PutUint16(p.Ram[offs:], word2)
	...
	v := pru.Order.Uint32(p.Ram[20:])
```

Of course, since the RAM is presented as a byte slice, any method that
uses a byte slice can work:

```
	f := os.Open("MyFile")
	f.Read(p.Ram[0x100:0x1FF])
	data := make([]byte, 0x200)
	copy(data, p.SharedRam[0x400:])
```

A Reader/Writer interface is available by using the ```Open``` method on any of the shared RAM fields:

```
	p := pru.Open()
	ram := p.Ram.Open()
	params := []interface{}{
		uint32(event),
		uint32(intrBit),
		uint16(2000),
		uint16(1000),
		uint32(0xDEADBEEF),
		uint32(in),
		uint32(out),
	}
	for _, v := range params {
		binary.Write(ram, pru.Order, v)
	}
	...
	ram.Seek(my_offset, io.SeekStart)
	fmt.Fprintf(ram, "Config string %d, %d", c1, c2)
	ram.WriteAt([]byte("A string to be written to PRU RAM"), 0x800)
	ram.Seek(0, io.SeekStart)
	b1 := ram.ReadByte()
	b2 := ram.ReadByte()
	...
```

A caveat is that the RAM is shared with the PRU, and Go does not have any explicit way
of indicating to the compiler that the memory is shared, so potentially there are patterns
of access where the compiler may optimise out accesses if care is not taken - the access may also
be subject to reordering.

If the memory access is done when the PRU units are disabled, then using the Reader/Writer interface or the
```binary/encoding``` methods described above should be sufficient.

For accesses that do rely on explicit ordering and reading or writing, it is recommended that the ```sync/ataomic```
and ```unsafe``` packages are used to access the memory:

```
	p := pru.Open()
	shared_rx := (*uint32)(unsafe.Pointer(&p.Ram[rx_offs]))
	shared_tx := (*uint32)(unsafe.Pointer(&p.Ram[tx_offs]))
	// Load and run PRU program ...
	for {
		n := atomic.LoadUint32(shared_rx)
		// process data from PRU
		...
		// Store word in PRU memory
		atomic.StoreUint32(shared_tx, 0xDEADBEEF)
	}
```

## Disclaimer

This is not an officially supported Google product.
