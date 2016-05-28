// generated by stringer -type=PhyInterface,IfSpanningTreeState,IfDuplex; DO NOT EDIT

package ethernet

import "fmt"

const _PhyInterface_name = "CAUICRCR2CR4GMIIINTERLAKENKRKR2KR4KXLRLR4MIIQSGMIIRGMIIRXAUISFISGMIISPAUISRSR10SR2SR4XAUIXFIXGMIIXLAUIXLAUI2ZR"

var _PhyInterface_index = [...]uint8{0, 4, 6, 9, 12, 16, 26, 28, 31, 34, 36, 38, 41, 44, 50, 55, 60, 63, 68, 73, 75, 79, 82, 85, 89, 92, 97, 102, 108, 110}

func (i PhyInterface) String() string {
	i -= 1
	if i < 0 || i >= PhyInterface(len(_PhyInterface_index)-1) {
		return fmt.Sprintf("PhyInterface(%d)", i+1)
	}
	return _PhyInterface_name[_PhyInterface_index[i]:_PhyInterface_index[i+1]]
}

const _IfSpanningTreeState_name = "DisableBlockListenLearnForward"

var _IfSpanningTreeState_index = [...]uint8{0, 7, 12, 18, 23, 30}

func (i IfSpanningTreeState) String() string {
	i -= 1
	if i < 0 || i >= IfSpanningTreeState(len(_IfSpanningTreeState_index)-1) {
		return fmt.Sprintf("IfSpanningTreeState(%d)", i+1)
	}
	return _IfSpanningTreeState_name[_IfSpanningTreeState_index[i]:_IfSpanningTreeState_index[i+1]]
}

const _IfDuplex_name = "FullHalf"

var _IfDuplex_index = [...]uint8{0, 4, 8}

func (i IfDuplex) String() string {
	i -= 1
	if i < 0 || i >= IfDuplex(len(_IfDuplex_index)-1) {
		return fmt.Sprintf("IfDuplex(%d)", i+1)
	}
	return _IfDuplex_name[_IfDuplex_index[i]:_IfDuplex_index[i+1]]
}