package pci

// Linux PCI code

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"syscall"
)

var rootDir string = "/sys/bus/pci/devices"

func (d *Device) Map(r *Resource) (err error) {
	fn := fmt.Sprintf("%s/%s/resource%d", rootDir, d.Addr.String(), r.Index)
	fd, err := syscall.Open(fn, syscall.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("open %s: %v", fn, err)
	}
	defer func() { syscall.Close(fd) }()
	r.Mem, err = syscall.Mmap(fd, 0, int(r.Size), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return fmt.Errorf("mmap %s: %v", fn, err)
	}
	return
}

func Probe() (err error) {
	fis, err := ioutil.ReadDir(rootDir)
	if perr, ok := err.(*os.PathError); ok && perr.Err == syscall.ENOENT {
		return
	}
	if err != nil {
		return
	}
	for _, fi := range fis {
		var d Device
		n := fi.Name()
		if _, err = fmt.Sscanf(n, "%x:%x:%x.%x", &d.Addr.Domain, &d.Addr.Bus, &d.Addr.Slot, &d.Addr.Fn); err != nil {
			return
		}

		devDir := rootDir + "/" + n

		d.configBytes, err = ioutil.ReadFile(devDir + "/config")
		if err != nil {
			return
		}

		r := bytes.NewReader(d.configBytes)
		binary.Read(r, binary.LittleEndian, &d.Config)
		if d.Config.Hdr.Type() != Normal {
			continue
		}

		var hw Hardware
		var ok bool
		if hw, ok = registeredDevs[d.Config.Hdr.DeviceID]; !ok {
			continue
		}

		for i := range d.Config.BaseAddress {
			bar := d.Config.BaseAddress[i]
			if bar == 0 {
				continue
			}
			var rfi os.FileInfo
			rfi, err = os.Stat(fmt.Sprintf("%s/resource%d", devDir, i))
			if err != nil {
				return
			}
			d.Resources = append(d.Resources, Resource{
				Index: uint32(i),
				BAR:   bar,
				Base:  uint64(bar.Addr()),
				Size:  uint64(rfi.Size()),
			})
		}

		err = hw.DeviceMatch(&d)
		if err != nil {
			return
		}
	}
	return
}
