# pru-rp
Go library for accessing the TI PRU using the
[RemoteProc](https://software-dl.ti.com/processor-sdk-linux/esd/docs/08_00_00_21/linux/Foundational_Components/PRU-ICSS/Linux_Drivers/RemoteProc.html)
framework. This API is used in more recent kernels.

godoc for this package is [available](https://pkg.go.dev/github.com/aamcrae/pru-rp).

## Examples

The examples use firmware from the [TI PRU Software Support package](https://git.ti.com/cgit/pru-software-support-package)
examples.
The firmware can be built from the example, and the relevant .out files copied
to /lib/firmware:

```
cd examples/am335x/PRU_RPMsg_Echo_Interrupt0
make
sudo cp gen/PRU_RPMsg_Echo_Interrupt0.out /lib/firmware/am335x-pru0-echo0-fw
```

Build and install the am335x-pru1-echo1-fw firmware in a similar way.
The examples used are:

| Example | Firmware file | Example source from package |
|---------|---------------|-----------------------------|
| echo | `am335x-pru{0,1}-echo{0,1}-fw` | `examples/am335x/PRU_RPMsg_Echo_Interrupt{0,1}` |

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
RAM block as a byte array.

### Permissions

The PRU memory is accessible via the ```/dev/mem``` device, so to read or write to the memory,
the program must have read/write access to this device.
The easiest way is by running the program as root, or using ```sudo```.
With earlier kernels, it was possible to change the ```/dev/mem``` permissions
to 0660 (to allow group r/w access), and add the user account to group ```kmem```, but
this is no longer enough as the CAP_SYS_RAWIO capability is also required to access this device.

### Shared Memory API

There are a number of ways that applications can access the shared memory as structured access.
For ease of access, the package exports a variable ```Order``` (as a ```binary/encoding Order```).
This allows use of the ```binary/encoding``` package:

```
	p := pru.Open()
	r := pru.Ram()
	pru.Order.PutUint32(r.Ram[0:], word1)
	pru.Order.PutUint32(r.Ram[4:], word2)
	pru.Order.PutUint16(r.Ram[offs:], word2)
	...
	v := pru.Order.Uint32(r.Ram[20:])
```

Of course, since the RAM is presented as a byte slice, any method that
uses a byte slice can work:

```
	r := pru.Ram()
	s := pru.SharedRam()
	f := os.Open("MyFile")
	f.Read(r.Ram[0x100:0x1FF])
	data := make([]byte, 0x200)
	copy(data, s.Ram[0x400:])
```

The RAM objects support the Reader/Writer interface:

```
	p := pru.Open()
	r := p.Ram()
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
		binary.Write(r, pru.Order, v)
	}
	...
	r.Seek(my_offset, io.SeekStart)
	fmt.Fprintf(r, "Config string %d, %d", c1, c2)
	r.WriteAt([]byte("A string to be written to PRU RAM"), 0x800)
	r.Seek(0, io.SeekStart)
	b1 := r.ReadByte()
	b2 := r.ReadByte()
	...
```

### Synchronisation

A caveat is that the RAM is shared with the PRU, and Go does not have any explicit way
of indicating to the compiler that the memory is shared, so potentially there are patterns
of access where the compiler may optimise out accesses if care is not taken - reads and writes may also
be subject to reordering.

If the memory is accessed when the PRU units are disabled, then using the Reader/Writer interface or the
```binary/encoding``` methods described above should be sufficient.

For accesses that do rely on explicit ordering for reading or writing,
or for when memory accesses may be concurrent with PRU access,
it is recommended that the ```sync/ataomic```
and ```unsafe``` packages are used to access the memory:

```
	p := pru.Open()
	r := pru.Ram()
	shared_rx := (*uint32)(unsafe.Pointer(&r.Ram[rx_offs]))
	shared_tx := (*uint32)(unsafe.Pointer(&r.Ram[tx_offs]))
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
