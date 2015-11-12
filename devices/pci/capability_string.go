// generated by stringer -type=Capability,HeaderType; DO NOT EDIT

package pci

import "fmt"

const _Capability_name = "PowerManagementAGPVitalProductDataSlotIdentificationMSICompactPCIHotSwapPCIXHyperTransportVendorSpecificDebugPortCompactPciCentralControlPCIHotPlugControllerSSVIDAGP3SecureDevicePCIEMSIXSATAAdvancedFeatures"

var _Capability_index = [...]uint8{0, 15, 18, 34, 52, 55, 72, 76, 90, 104, 113, 137, 157, 162, 166, 178, 182, 186, 190, 206}

func (i Capability) String() string {
	i -= 1
	if i >= Capability(len(_Capability_index)-1) {
		return fmt.Sprintf("Capability(%d)", i+1)
	}
	return _Capability_name[_Capability_index[i]:_Capability_index[i+1]]
}

const _HeaderType_name = "NormalBridgeCardBus"

var _HeaderType_index = [...]uint8{0, 6, 12, 19}

func (i HeaderType) String() string {
	if i >= HeaderType(len(_HeaderType_index)-1) {
		return fmt.Sprintf("HeaderType(%d)", i)
	}
	return _HeaderType_name[_HeaderType_index[i]:_HeaderType_index[i+1]]
}
