package sfp

type SfpId uint8

const (
	Unknown SfpId = iota
	Gbic
	OnMotherboard
	Sfp
	Xbi
	Xenpak
	Xfp
	Xff
	XfpE
	Xpak
	X2
	DwdmSfp
	Qsfp
)

// Lower memory map.
// Everything in network byte order.
// Bytes 0-85 are read only; 86-128 are read/write.
type Qsfp struct {
	Id SfpId

	// [0] data not ready, [1] interrupt active low pin value
	Status [2]byte

	ChannelStatusInterrupt  [3]byte
	ModuleMonitorInterrupt  [3]byte
	ChannelMonitorInterrupt [4]byte
	_                       [9]byte
	// Module Monitoring Values.
	Temperature   [2]byte // signed 16 bit, units of degrees Celsius/256
	_             [2]byte
	SupplyVoltage [2]byte // 16 unsigned; units of 100e-6 Volts
	_             [6]byte
	// Channel Monitoring Values.
	RxPower       [4][2]byte // unsigned 16 bit, units of 1e-7 Watts
	TxBiasCurrent [4][2]byte // unsigned 16 bit, units of 2e-6 Amps
	_             [86 - 50]byte

	// Bytes 86 through 128 are all read/write.
	TxDisable           byte // [3:0] per channel laser disable
	RxRateSelect        byte
	TxRateSelect        byte
	RxApplicationSelect [4]byte
	// [1] low power enable [0] override
	PowerControl        byte
	TxApplicationSelect [4]byte
	_                   [2]byte
	InterruptDisable    [7]byte
	_                   [12]byte
	PasswordEntryChange [4]byte
	PasswordEntry       [4]byte
	PageSelect          byte
}

// Upper memory map (page select 0)
// Read only.
type Sfp struct {
	Id                           SfpId
	ExtendedId                   byte
	ConnectorType                byte
	Compatibility                [8]byte
	Encoding                     byte
	NominalBitRate100MbitsPerSec Byte
	_                            byte
	LinkLength                   [5]byte
	_                            byte
	VendorName                   [16]byte
	_                            byte
	VendorOui                    [3]byte
	VendorPartNumber             [16]byte
	VendorRevision               [4]byte
	LaserWavelengthInNm          [2]byte
	_                            byte
	checksum_0_to_62             byte
	Options                      [2]byte
	MaxBitRateMarginPercent      byte
	MinBitRateMarginPercent      byte
	VendorSerialNumber           [16]byte
	VendorDateCode               [8]byte
	_                            [3]byte
	checksum_63_to_94            byte
	VendorSpecific               [32]byte
}

type qsfpHighLow struct {
	High, Low [2]byte
}

type qsfpThreshold struct {
	Alarm, Warning qsfpHighLow
}

// Upper memory map (page select 3)
type QsfpThresholds struct {
	Temperature   qsfpThreshold
	_             [8]byte
	Vcc           qsfpThreshold
	_             [176 - 152]byte
	RxPower       qsfpThreshold
	TxBiasCurrent qsfpThreshold
	_             [226 - 192]byte

	// Bytes 226-255 are read/write.
	VendorChannelControls     [14]byte
	OptionalChannelControlls  [2]byte
	ThresholdInterruptDisable [4]byte
	_                         [256 - 246]byte
}
