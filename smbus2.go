// Thia package seeks to provide a very similar API to the smbus2 python
// package, except written in pure Go. The point of the package is to simplify
// porting some python robotics code to go. This package is only expected to
// work on Linux, tested on Raspberry pi.
//
// A lot of the code is ported near-verbatim from python-smbus2; interested
// readers should view the original repository:
// https://github.com/kplindegaard/smbus2.git
//
// Variables and functions have been renamed to align with Go conventions.
package smbus2_go

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	// Commands from uapi/linux/i2c-dev.h
	// Use this slave address
	I2CSlave = 0x0703
	// Use this slave address, even if it already in use by a driver!
	I2CSlaveForce = 0x0706
	// Get the adapter functionality mask
	I2CFuncs = 0x0705
	// Combined R/W transfer (one STOP only)
	I2CRDWR = 0x0707
	// SMBus transfer. Takes pointer to i2c_smbus_ioctl_data
	I2CSMBus = 0x0720
	// != 0 to use PEC with SMBus
	I2CPEC = 0x0708

	// SMBus transfer read or write markers from uapi/linux/i2c.h
	I2CSMBusWrite = 0
	I2CSMBusRead  = 1

	// Size identifiers uapi/linux/i2c.h
	I2CSMBusQuick    = 0
	I2CSMBusByte     = 1
	I2CSMBusByteData = 2
	I2CSMBusWordData = 3
	I2CSMBusProcCall = 4
	// This isn't supported by Pure-I2C drivers with SMBUS emulation, like
	// those in Raspberry Pi, OrangePi, etc :(
	I2CSMBusBlockData = 5
	// Like I2CSMBusBlockData, it isn't supported by Pure-I2C drivers either.
	I2CSMBusBlockProcCall = 7
	I2CSMBusI2CBlockData  = 8
	I2CSMBusBlockMax      = 32

	// These flags identify the operations supported by an I2C/SMBus device.
	I2CFlag       = 0x00000001
	Addr10BitFlag = 0x00000002
	// I2C_M_IGNORE_NAK, etc.
	ProtocolManglingFlag = 0x00000004
	SMBUSPECFlag         = 0x00000008
	// I2C_M_NOSTART
	NoStartFlag             = 0x00000010
	SlaveFlag               = 0x00000020
	SMBusBlockProcCallFlag  = 0x00008000
	SMBusQuickFlag          = 0x00010000
	SMBusReadByteFlag       = 0x00020000
	SMBusWriteByteFlag      = 0x00040000
	SMBusReadByteDataFlag   = 0x00080000
	SMBusWriteByteDataFlag  = 0x00100000
	SMBusReadWordDataFlag   = 0x00200000
	SMBusWriteWordDataFlag  = 0x00400000
	SMBusProcCallFlag       = 0x00800000
	SMBusReadBlockDataFlag  = 0x01000000
	SMBusWriteBlockDataFlag = 0x02000000
	// I2C-like block transfer
	SMBusReadI2CBlockFlag = 0x04000000
	// With 1-byte register address
	SMBusWriteI2CBlockFlag = 0x08000000
	SMBusHostNotifyFlag    = 0x10000000

	SMBusByteFlag      = 0x00060000
	SMBusByteDataFlag  = 0x00180000
	SMBusWordDataFlag  = 0x00600000
	SMBusBlockDataFlag = 0x03000000
	SMBusI2CBlockFlag  = 0x0c000000
	SMBusEmulFlag      = 0x0eff0008

	// i2c_msg flags from uapi/linux/i2c.h
	I2CMRD = 0x0001
)

// These represent a bitfield indicating the capabilities of a bus.
type FunctionFlags uint32

func (f FunctionFlags) String() string {
	return fmt.Sprintf("0x%08x", uint32(f))
}

// Returns true if all of the bits are set in f.
func (f FunctionFlags) BitsSet(bits uint32) bool {
	return (uint32(f) & bits) == bits
}

// Returns the name associated with a single set of "flag" bits.
func getSingleFlagName(bits uint32) string {
	switch bits {
	case I2CFlag:
		return "I2C"
	case Addr10BitFlag:
		return "10-bit address"
	case ProtocolManglingFlag:
		return "Protocol mangling"
	case SMBUSPECFlag:
		return "SMBus PEC"
	case NoStartFlag:
		return "No start"
	case SlaveFlag:
		return "Slave"
	case SMBusBlockProcCallFlag:
		return "Block procedure call"
	case SMBusQuickFlag:
		return "Quick"
	case SMBusReadByteFlag:
		return "Read byte"
	case SMBusWriteByteFlag:
		return "Write byte"
	case SMBusReadByteDataFlag:
		return "Read byte data"
	case SMBusWriteByteDataFlag:
		return "Write byte data"
	case SMBusReadWordDataFlag:
		return "Read word data"
	case SMBusWriteWordDataFlag:
		return "Write word data"
	case SMBusProcCallFlag:
		return "Procedure call"
	case SMBusReadBlockDataFlag:
		return "Read block data"
	case SMBusWriteBlockDataFlag:
		return "Write block data"
	case SMBusReadI2CBlockFlag:
		return "Read I2C block"
	case SMBusWriteI2CBlockFlag:
		return "Write I2C block"
	case SMBusHostNotifyFlag:
		return "Host notify"
	case SMBusByteFlag:
		return "Byte"
	case SMBusByteDataFlag:
		return "Byte data"
	case SMBusWordDataFlag:
		return "Word data"
	case SMBusBlockDataFlag:
		return "Block data"
	case SMBusI2CBlockFlag:
		return "I2C block"
	case SMBusEmulFlag:
		return "Emulated"
	}
	return fmt.Sprintf("Unknown flag bits: 0x%08x", bits)
}

// Returns a list of string representations of each of the set flags.
func (f FunctionFlags) GetStringsList() []string {
	toReturn := make([]string, 0, 16)
	flags := []uint32{
		I2CFlag,
		Addr10BitFlag,
		ProtocolManglingFlag,
		SMBUSPECFlag,
		NoStartFlag,
		SlaveFlag,
		SMBusBlockProcCallFlag,
		SMBusQuickFlag,
		SMBusReadByteFlag,
		SMBusWriteByteFlag,
		SMBusReadByteDataFlag,
		SMBusWriteByteDataFlag,
		SMBusReadWordDataFlag,
		SMBusWriteWordDataFlag,
		SMBusProcCallFlag,
		SMBusReadBlockDataFlag,
		SMBusWriteBlockDataFlag,
		SMBusReadI2CBlockFlag,
		SMBusWriteI2CBlockFlag,
		SMBusHostNotifyFlag,
		SMBusByteFlag,
		SMBusByteDataFlag,
		SMBusWordDataFlag,
		SMBusBlockDataFlag,
		SMBusI2CBlockFlag,
		SMBusEmulFlag,
	}
	for _, flag := range flags {
		if f.BitsSet(flag) {
			toReturn = append(toReturn, getSingleFlagName(flag))
		}
	}
	return toReturn
}

// Provides a ioctl wrapper that works with the syscall library. Sorry for the
// unsafe usage.
func ioctl(fd int, cmd uintptr, arg uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), cmd, arg)
	if errno != 0 {
		return fmt.Errorf("Error making ioctl: %w", errno)
	}
	return nil
}

// Tracks state for a single open bus.
type SMBus struct {
	fd int
	// A bitfield indicating what functions are supported by the I2C device.
	Funcs             FunctionFlags
	forceSlaveAddress bool
	forceLast         bool
	pec               uint32
}

// Should be called when the SMBus connection is no longer needed. Closes the
// underlying file descriptor.
func (b *SMBus) Close() error {
	e := syscall.Close(b.fd)
	b.fd = -1
	return e
}

// Opens the given I2C bus. Requires the I2C bus number.
func NewSMBus(busID int) (*SMBus, error) {
	return NewSMBusWithPath(fmt.Sprintf("/dev/i2c-%d", busID))
}

// Like NewSMBus, but takes a path to an smbus device, i.e., "/dev/i2c-0".
func NewSMBusWithPath(path string) (*SMBus, error) {
	fd, e := syscall.Open(path, syscall.O_RDWR, 0666)
	if e != nil {
		return nil, fmt.Errorf("Error opening %s: %w", path, e)
	}
	funcs := uint32(0)
	e = ioctl(fd, I2CFuncs, uintptr(unsafe.Pointer(&funcs)))
	if e != nil {
		syscall.Close(fd)
		return nil, fmt.Errorf("Error getting funcs for %s: %w", path, e)
	}
	return &SMBus{
		fd:    fd,
		Funcs: FunctionFlags(funcs),
	}, nil
}