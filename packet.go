package vnet

// Generic packet type
type PacketType int

const (
	IP4 PacketType = 1 + iota
	IP6
	MPLS_UNICAST
	MPLS_MULTICAST
)
