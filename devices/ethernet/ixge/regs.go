// Driver for Intel 10G Ethernet controllers.
package ixge

import (
	"github.com/platinasystems/elib/hw"

	"unsafe"
)

type reg hw.Reg32

func (r *reg) offset() uint         { return uint(uintptr(unsafe.Pointer(r)) - hw.RegsBaseAddress) }
func (r *reg) addr(d *dev) *uint32  { return (*uint32)(unsafe.Pointer(&d.mmaped_regs[r.offset()])) }
func (r *reg) get(d *dev) reg       { return reg(hw.LoadUint32(r.addr(d))) }
func (r *reg) set(d *dev, v reg)    { hw.StoreUint32(r.addr(d), uint32(v)) }
func (r *reg) or(d *dev, v reg)     { r.set(d, r.get(d)|v) }
func (r *reg) andnot(d *dev, v reg) { r.set(d, r.get(d)&^v) }

type dma_regs struct {
	// [31:7] 128 byte aligned.
	descriptor_address [2]reg

	n_descriptor_bytes reg

	// [5] rx/tx descriptor dca enable
	// [6] rx packet head dca enable
	// [7] rx packet tail dca enable
	// [9] rx/tx descriptor relaxed order
	// [11] rx/tx descriptor write back relaxed order
	// [13] rx/tx data write/read relaxed order
	// [15] rx head data write relaxed order
	// [31:24] apic id for cpu's cache.
	dca_control reg

	head_index reg

	// [4:0] tail buffer size (in 1k byte units)
	// [13:8] head buffer size (in 64 byte units)
	// [24:22] lo free descriptors threshold (units of 64 descriptors)
	// [27:25] descriptor type 0 = legacy, 1 = advanced one buffer (e.g. tail),
	//   2 = advanced header splitting (head + tail), 5 = advanced header splitting (head only).
	// [28] drop if no descriptors available.
	rx_split_control reg

	tail_index reg

	_ [0x28 - 0x1c]byte

	// [7:0] rx/tx prefetch threshold
	// [15:8] rx/tx host threshold
	// [24:16] rx/tx write back threshold
	// [25] rx/tx enable
	// [26] tx descriptor writeback flush
	// [30] rx strip vlan enable
	control reg

	rx_coallesce_control reg
}

type rx_dma_regs struct {
	dma_regs

	stats [3]reg

	_ reg
}

type tx_dma_regs struct {
	dma_regs

	_ [2]reg

	// [0] enables head write back.
	head_index_write_back_address [2]reg
}

// Only advanced descriptors are supported.
type rx_to_hw_descriptor struct {
	tail_buffer_address uint64
	head_buffer_address uint64
}

// Rx writeback descriptor format.
type rx_from_hw_descriptor struct {
	status [3]uint32

	n_bytes_this_descriptor uint16
	vlan_tag                uint16
}

type tx_descriptor struct {
	buffer_address      uint64
	n_bytes_this_buffer uint16
	status0             uint16
	status1             uint32
}

type regs struct {
	/* [2] pcie master disable
	   [3] mac reset
	   [26] global device reset */
	control       reg
	control_alias reg

	/* [3:2] device id (0 or 1 for dual port chips)
	   [7] link is up
	   [17:10] num vfs
	   [18] io active
	   [19] pcie master enable status */
	status_read_only reg
	_                [0x18 - 0xc]byte

	/* [14] pf reset done
	   [17] relaxed ordering disable
	   [26] extended vlan enable
	   [28] driver loaded */
	extended_control reg
	_                [0x20 - 0x1c]byte

	/* software definable pins.
	   sdp_data [7:0]
	   sdp_is_output [15:8]
	   sdp_is_native [23:16]
	   sdp_function [31:24]. */
	sdp_control reg
	_           [0x28 - 0x24]byte

	/* [0] i2c clock in
	   [1] i2c clock out
	   [2] i2c data in
	   [3] i2c data out */
	i2c_control reg
	_           [0x4c - 0x2c]byte
	tcp_timer   reg

	_ [0x200 - 0x50]byte

	led_control reg

	_          [0x600 - 0x204]byte
	core_spare reg
	_          [0x700 - 0x604]byte

	pf_0 struct {
		vflr_events_clear        [4]reg
		mailbox_interrupt_status [4]reg
		mailbox_interrupt_enable [4]reg
		_                        [0x800 - 0x730]byte
	}

	interrupt struct {
		status_write_1_to_clear  reg
		_                        [0x808 - 0x804]byte
		status_write_1_to_set    reg
		_                        [0x810 - 0x80c]byte
		status_auto_clear_enable reg
		_                        [0x820 - 0x814]byte

		/* [11:3] minimum inter-interrupt interval
		     (2e-6 units 20e-6 units for fast ethernet).
		   [15] low-latency interrupt moderation enable
		   [20:16] low-latency interrupt credit
		   [27:21] interval counter
		   [31] write disable for credit and counter (write only). */
		throttle0 [24]reg

		enable_write_1_to_set   reg
		_                       [0x888 - 0x884]byte
		enable_write_1_to_clear reg
		_                       [0x890 - 0x88c]byte
		enable_auto_clear       reg
		msi_to_eitr_select      reg
		/* [3:0] spd 0-3 interrupt detection enable
		   [4] msi-x enable
		   [5] other clear disable (makes other bits in status not clear on read)
		   etc. */
		control reg
		_       [0x900 - 0x89c]byte

		/* Defines interrupt mapping for 128 rx + 128 tx queues.
		       64 x 4 8 bit entries.
		       For register [i]:
		         [5:0] bit in interrupt status for rx queue 2*i + 0
			 [7] valid bit
			 [13:8] bit for tx queue 2*i + 0
			 [15] valid bit
			 similar for rx 2*i + 1 and tx 2*i + 1. */
		queue_mapping [64]reg

		/* tcp timer [7:0] and other interrupts [15:8] */
		misc_mapping reg
		_            [0xa90 - 0xa04]byte

		/* 64 interrupts determined by mappings. */
		status1_write_1_to_clear  [4]reg
		enable1_write_1_to_set    [4]reg
		enable1_write_1_to_clear  [4]reg
		_                         [0xad0 - 0xac0]byte
		status1_enable_auto_clear [4]reg
		_                         [0x1000 - 0xae0]byte
	}

	rx_dma0 [64]rx_dma_regs

	_                             [0x2140 - 0x2000]byte
	dcb_rx_packet_plane_t4_config [8]reg
	dcb_rx_packet_plane_t4_status [8]reg
	_                             [0x2300 - 0x2180]byte

	/*  i defines mapping for 4 rx queues starting at 4*i + 0. */
	rx_queue_stats_mapping [32]reg
	rx_queue_stats_control reg

	_                        [0x2410 - 0x2384]byte
	fc_user_descriptor_ptr   [2]reg
	fc_buffer_control        reg
	_                        [0x2420 - 0x241c]byte
	fc_rx_dma                reg
	_                        [0x2430 - 0x2424]byte
	dcb_packet_plane_control reg
	_                        [0x2f00 - 0x2434]byte

	rx_dma_control                 reg
	pf_queue_drop_enable           reg
	_                              [0x2f20 - 0x2f08]byte
	rx_dma_descriptor_cache_config reg
	_                              [0x3000 - 0x2f24]byte

	/* 1 bit. */
	rx_enable reg
	_         [0x3008 - 0x3004]byte
	/* [15:0] ether type (little endian)
	   [31:16] opcode (big endian) */
	flow_control_control reg
	_                    [0x3020 - 0x300c]byte
	/* 3 bit traffic class for each of 8 priorities. */
	rx_priority_to_traffic_class     reg
	_                                [0x3028 - 0x3024]byte
	rx_coallesce_data_buffer_control reg
	_                                [0x3190 - 0x302c]byte
	rx_packet_buffer_flush_detect    reg
	_                                [0x3200 - 0x3194]byte
	flow_control_tx_timers           [4] /* 2 timer values */ reg
	_                                [0x3220 - 0x3210]byte
	flow_control_rx_threshold_lo     [8]reg
	_                                [0x3260 - 0x3240]byte
	flow_control_rx_threshold_hi     [8]reg
	_                                [0x32a0 - 0x3280]byte
	flow_control_refresh_threshold   reg
	_                                [0x3c00 - 0x32a4]byte
	/* For each of 8 traffic classes (units of bytes). */
	rx_packet_buffer_size [8]reg
	_                     [0x3d00 - 0x3c20]byte
	flow_control_config   reg
	_                     [0x4200 - 0x3d04]byte

	ge_mac struct {
		pcs_config                              reg
		_                                       [0x4208 - 0x4204]byte
		link_control                            reg
		link_status                             reg
		pcs_debug                               [2]reg
		auto_negotiation                        reg
		link_partner_ability                    reg
		auto_negotiation_tx_next_page           reg
		auto_negotiation_link_partner_next_page reg
		_                                       [0x4240 - 0x4228]byte
	}

	xge_mac struct {
		/* [0] tx crc enable
		   [2] enable frames up to max frame size register [31:16]
		   [10] pad frames < 64 bytes if specified by user
		   [15] loopback enable
		   [16] mdc hi speed
		   [17] turn off mdc between mdio packets */
		control reg

		/* [5] rx symbol error (all bits clear on read)
		   [6] rx illegal symbol
		   [7] rx idle error
		   [8] rx local fault
		   [9] rx remote fault */
		status reg

		pause_and_pace_control reg
		_                      [0x425c - 0x424c]byte
		phy_command            reg
		phy_data               reg
		_                      [0x4268 - 0x4264]byte

		/* [31:16] max frame size in bytes. */
		rx_max_frame_size reg
		_                 [0x4288 - 0x426c]byte

		/* [0]
		         [2] pcs receive link up? (latch lo)
			 [7] local fault
		       [1]
		         [0] pcs 10g base r capable
		         [1] pcs 10g base x capable
		         [2] pcs 10g base w capable
			 [10] rx local fault
			 [11] tx local fault
			 [15:14] 2 => device present at this address (else not present) */
		xgxs_status [2]reg

		base_x_pcs_status reg

		/* [0] pass unrecognized flow control frames
		   [1] discard pause frames
		   [2] rx priority flow control enable (only in dcb mode)
		   [3] rx flow control enable. */
		flow_control reg

		/* [3:0] tx lanes change polarity
		   [7:4] rx lanes change polarity
		   [11:8] swizzle tx lanes
		   [15:12] swizzle rx lanes
		   4 x 2 bit tx lane swap
		   4 x 2 bit rx lane swap. */
		serdes_control reg

		fifo_control reg

		/* [0] force link up
		       [1] autoneg ack2 bit to transmit
		       [6:2] autoneg selector field to transmit
		       [8:7] 10g pma/pmd type 0 => xaui, 1 kx4, 2 cx4
		       [9] 1g pma/pmd type 0 => sfi, 1 => kx/bx
		       [10] disable 10g on without main power
		       [11] restart autoneg on transition to dx power state
		       [12] restart autoneg
		       [15:13] link mode:
		         0 => 1g no autoneg
			 1 => 10g kx4 parallel link no autoneg
			 2 => 1g bx autoneg
			 3 => 10g sfi serdes
			 4 => kx4/kx/kr
			 5 => xgmii 1g/100m
			 6 => kx4/kx/kr 1g an
			 7 kx4/kx/kr sgmii.
		       [16] kr support
		       [17] fec requested
		       [18] fec ability
		       etc. */
		auto_negotiation_control reg

		/* [0] signal detect 1g/100m
		       [1] fec signal detect
		       [2] 10g serial pcs fec block lock
		       [3] 10g serial high error rate
		       [4] 10g serial pcs block lock
		       [5] kx/kx4/kr autoneg next page received
		       [6] kx/kx4/kr backplane autoneg next page received
		       [7] link status clear to read
		       [11:8] 10g signal detect (4 lanes) (for serial just lane 0)
		       [12] 10g serial signal detect
		       [16:13] 10g parallel lane sync status
		       [17] 10g parallel align status
		       [18] 1g sync status
		       [19] kx/kx4/kr backplane autoneg is idle
		       [20] 1g autoneg enabled
		       [21] 1g pcs enabled for sgmii
		       [22] 10g xgxs enabled
		       [23] 10g serial fec enabled (forward error detection)
		       [24] 10g kr pcs enabled
		       [25] sgmii enabled
		       [27:26] mac link mode
		         0 => 1g
			 1 => 10g parallel
			 2 => 10g serial
			 3 => autoneg
		       [29:28] link speed
		         1 => 100m
		         2 => 1g
		         3 => 10g
		       [30] link is up
		       [31] kx/kx4/kr backplane autoneg completed successfully. */
		link_status reg

		/* [17:16] pma/pmd for 10g serial
		   0 => kr, 2 => sfi
		 [18] disable dme pages */
		auto_negotiation_control2 reg

		_                      [0x42b0 - 0x42ac]byte
		link_partner_ability   [2]reg
		_                      [0x42d0 - 0x42b8]byte
		manageability_control  reg
		link_partner_next_page [2]reg
		_                      [0x42e0 - 0x42dc]byte
		kr_pcs_control         reg
		kr_pcs_status          reg
		fec_status             [2]reg
		_                      [0x4314 - 0x42f0]byte
		sgmii_control          reg
		_                      [0x4324 - 0x4318]byte
		link_status2           reg
		_                      [0x4900 - 0x4328]byte
	}

	tx_dcb_control                       reg
	tx_dcb_descriptor_plane_queue_select reg
	tx_dcb_descriptor_plane_t1_config    reg
	tx_dcb_descriptor_plane_t1_status    reg
	_                                    [0x4950 - 0x4910]byte

	/* For each TC in units of 1k bytes. */
	tx_packet_buffer_thresholds [8]reg
	_                           [0x4980 - 0x4970]byte

	dcb_tx_rate_scheduler struct {
		mmw        reg
		config     reg
		status     reg
		rate_drift reg
	}
	_                        [0x4a80 - 0x4990]byte
	tx_dma_control           reg
	_                        [0x4a88 - 0x4a84]byte
	tx_dma_tcp_flags_control [2]reg
	_                        [0x4b00 - 0x4a90]byte
	pf_mailbox               [64]reg
	_                        [0x5000 - 0x4c00]byte

	/* RX */
	checksum_control         reg
	_                        [0x5008 - 0x5004]byte
	rx_filter_control        reg
	_                        [0x5010 - 0x500c]byte
	management_vlan_tag      [8]reg
	management_udp_tcp_ports [8]reg
	_                        [0x5078 - 0x5050]byte
	/* little endian. */
	extended_vlan_ether_type reg
	_                        [0x5080 - 0x507c]byte
	/* [1] store/dma bad packets
	   [8] accept all multicast
	   [9] accept all unicast
	   [10] accept all broadcast. */
	filter_control reg
	_              [0x5088 - 0x5084]byte
	/* [15:0] vlan ethernet type (0x8100) little endian
	   [28] cfi bit expected
	   [29] drop packets with unexpected cfi bit
	   [30] vlan filter enable. */
	vlan_control reg
	_            [0x5090 - 0x508c]byte
	/* [1:0] hi bit of ethernet address for 12 bit index into multicast table
	   0 => 47, 1 => 46, 2 => 45, 3 => 43. */
	multicast_control  reg
	_                  [0x5100 - 0x5094]byte
	fcoe_rx_control    reg
	_                  [0x5108 - 0x5104]byte
	fc_flt_context     reg
	_                  [0x5110 - 0x510c]byte
	fc_filter_control  reg
	_                  [0x5120 - 0x5114]byte
	rx_message_type_lo reg
	_                  [0x5128 - 0x5124]byte
	/* [15:0] ethernet type (little endian)
	   [18:16] match pri in vlan tag
	   [19] priority match enable
	   [25:20] virtualization pool
	   [26] pool enable
	   [27] is fcoe
	   [30] ieee 1588 timestamp enable
	   [31] filter enable.
	   (See ethernet_type_queue_select.) */
	ethernet_type_queue_filter [8]reg
	_                          [0x5160 - 0x5148]byte
	/* [7:0] l2 ethernet type and
	   [15:8] l2 ethernet type or */
	management_decision_filters1     [8]reg
	vf_vm_tx_switch_loopback_enable  [2]reg
	rx_time_sync_control             reg
	_                                [0x5190 - 0x518c]byte
	management_ethernet_type_filters [4]reg
	rx_timestamp_attributes_lo       reg
	rx_timestamp_hi                  reg
	rx_timestamp_attributes_hi       reg
	_                                [0x51b0 - 0x51ac]byte
	pf_virtual_control               reg
	_                                [0x51d8 - 0x51b4]byte
	fc_offset_parameter              reg
	_                                [0x51e0 - 0x51dc]byte
	vf_rx_enable                     [2]reg
	rx_timestamp_lo                  reg
	_                                [0x5200 - 0x51ec]byte
	/* 12 bits determined by multicast_control
	   lookup bits in this vector. */
	multicast_enable [128]reg

	/* [0] ethernet address [31:0]
	   [1] [15:0] ethernet address [47:32]
	       [31] valid bit.
	   Index 0 is read from eeprom after reset. */
	rx_ethernet_address0 [16][2]reg

	_                               [0x5800 - 0x5480]byte
	wake_up_control                 reg
	_                               [0x5808 - 0x5804]byte
	wake_up_filter_control          reg
	_                               [0x5818 - 0x580c]byte
	multiple_rx_queue_command_82598 reg
	_                               [0x5820 - 0x581c]byte
	management_control              reg
	management_filter_control       reg
	_                               [0x5838 - 0x5828]byte
	wake_up_ip4_address_valid       reg
	_                               [0x5840 - 0x583c]byte
	wake_up_ip4_address_table       [4]reg
	management_control_to_host      reg
	_                               [0x5880 - 0x5854]byte
	wake_up_ip6_address_table       [4]reg

	/* unicast_and broadcast_and vlan_and ip_address_and
	   etc. */
	management_decision_filters [8]reg

	management_ip4_or_ip6_address_filters [4][4]reg
	_                                     [0x5900 - 0x58f0]byte
	wake_up_packet_length                 reg
	_                                     [0x5910 - 0x5904]byte
	management_ethernet_address_filters   [4][2]reg
	_                                     [0x5a00 - 0x5930]byte
	wake_up_packet_memory                 [32]reg
	_                                     [0x5c00 - 0x5a80]byte
	redirection_table_82598               [32]reg
	rss_random_keys_82598                 [10]reg
	_                                     [0x6000 - 0x5ca8]byte

	tx_dma [128]tx_dma_regs

	pf_vm_vlan_insert                  [64]reg
	tx_dma_tcp_max_alloc_size_requests reg
	_                                  [0x8110 - 0x8104]byte
	vf_tx_enable                       [2]reg
	_                                  [0x8120 - 0x8118]byte
	/* [0] dcb mode enable
	   [1] virtualization mode enable
	   [3:2] number of tcs/qs per pool. */
	multiple_tx_queues_command   reg
	_                            [0x8200 - 0x8124]byte
	pf_vf_anti_spoof             [8]reg
	pf_dma_tx_switch_control     reg
	_                            [0x82e0 - 0x8224]byte
	tx_strict_low_latency_queues [4]reg
	_                            [0x8600 - 0x82f0]byte
	tx_queue_stats_mapping_82599 [32]reg
	tx_queue_packet_counts       [32]reg
	tx_queue_byte_counts         [32][2]reg

	tx_security struct {
		control            reg
		status             reg
		buffer_almost_full reg
		_                  [0x8810 - 0x880c]byte
		buffer_min_ifg     reg
		_                  [0x8900 - 0x8814]byte
	}

	tx_ipsec struct {
		index reg
		salt  reg
		key   [4]reg
		_     [0x8a00 - 0x8918]byte
	}

	tx_link_security struct {
		capabilities reg
		control      reg
		tx_sci       [2]reg
		sa           reg
		sa_pn        [2]reg
		key          [2][4]reg
		/* untagged packets, encrypted packets, protected packets,
		   encrypted bytes, protected bytes */
		stats [5]reg
		_     [0x8c00 - 0x8a50]byte
	}

	tx_timesync struct {
		control                reg
		timestamp_value        [2]reg
		system_time            [2]reg
		increment_attributes   reg
		time_adjustment_offset [2]reg
		aux_control            reg
		target_time            [2][2]reg
		_                      [0x8c3c - 0x8c34]byte
		aux_time_stamp         [2][2]reg
		_                      [0x8d00 - 0x8c4c]byte
	}

	rx_security struct {
		control reg
		status  reg
		_       [0x8e00 - 0x8d08]byte
	}

	rx_ipsec struct {
		index      reg
		ip_address [4]reg
		spi        reg
		ip_index   reg
		key        [4]reg
		salt       reg
		mode       reg
		_          [0x8f00 - 0x8e34]byte
	}

	rx_link_security struct {
		capabilities reg
		control      reg
		sci          [2]reg
		sa           [2]reg
		sa_pn        [2]reg
		key          [2][4]reg
		/* see datasheet */
		stats [17]reg
		_     [0x9000 - 0x8f84]byte
	}

	/* 4 wake up, 2 management, 2 wake up. */
	flexible_filters [8][16][4]reg
	_                [0xa000 - 0x9800]byte

	/* 4096 bits. */
	vlan_filter [128]reg

	/* [0] ethernet address [31:0]
	   [1] [15:0] ethernet address [47:32]
	   [31] valid bit.
	   Index 0 is read from eeprom after reset. */
	rx_ethernet_address1 [128][2]reg

	/* select one of 64 pools for each rx address. */
	rx_ethernet_address_pool_select [128][2]reg
	_                               [0xc800 - 0xaa00]byte
	tx_priority_to_traffic_class    reg
	_                               [0xcc00 - 0xc804]byte

	/* In bytes units of 1k.  Total packet buffer is 160k. */
	tx_packet_buffer_size [8]reg

	_                             [0xcd10 - 0xcc20]byte
	tx_manageability_tc_mapping   reg
	_                             [0xcd20 - 0xcd14]byte
	dcb_tx_packet_plane_t2_config [8]reg
	dcb_tx_packet_plane_t2_status [8]reg
	_                             [0xce00 - 0xcd60]byte

	tx_flow_control_status reg
	_                      [0xd000 - 0xce04]byte

	rx_dma1 [64]rx_dma_regs

	ip4_filters struct {
		/* Bigendian ip4 src/dst address. */
		src_address [128]reg
		dst_address [128]reg

		/* TCP/UDP ports [15:0] src [31:16] dst bigendian. */
		tcp_udp_port [128]reg

		/* [1:0] protocol tcp, udp, sctp, other
		   [4:2] match priority (highest wins)
		   [13:8] pool
		   [25] src address match disable
		   [26] dst address match disable
		   [27] src port match disable
		   [28] dst port match disable
		   [29] protocol match disable
		   [30] pool match disable
		   [31] enable. */
		control [128]reg

		/* [12] size bypass
		   [19:13] must be 0x80
		   [20] low-latency interrupt
		   [27:21] rx queue. */
		interrupt [128]reg
	}

	_ [0xeb00 - 0xea00]byte
	/* 4 bit rss output index indexed by 7 bit hash.
	   128 8 bit fields = 32 registers. */
	redirection_table_82599 [32]reg

	rss_random_key_82599 [10]reg
	_                    [0xec00 - 0xeba8]byte
	/* [15:0] reserved
	   [22:16] rx queue index
	   [29] low-latency interrupt on match
	   [31] enable */
	ethernet_type_queue_select           [8]reg
	_                                    [0xec30 - 0xec20]byte
	syn_packet_queue_filter              reg
	_                                    [0xec60 - 0xec34]byte
	immediate_interrupt_rx_vlan_priority reg
	_                                    [0xec70 - 0xec64]byte
	rss_queues_per_traffic_class         reg
	_                                    [0xec90 - 0xec74]byte
	lli_size_threshold                   reg
	_                                    [0xed00 - 0xec94]byte

	fcoe_redirection struct {
		control reg
		_       [0xed10 - 0xed04]byte
		table   [8]reg
		_       [0xee00 - 0xed30]byte
	}

	flow_director struct {
		/* [1:0] packet buffer allocation 0 => disabled, else 64k*2^(f-1)
		   [3] packet buffer initialization done
		   [4] perfetch match mode
		   [5] report status in rss field of rx descriptors
		   [7] report status always
		   [14:8] drop queue
		   [20:16] flex 2 byte packet offset (units of 2 bytes)
		   [27:24] max linked list length
		   [31:28] full threshold. */
		control reg
		_       [0xee0c - 0xee04]byte

		data [8]reg

		/* [1:0] 0 => no action, 1 => add, 2 => remove, 3 => query.
		   [2] valid filter found by query command
		   [3] filter update override
		   [4] ip6 adress table
		   [6:5] l4 protocol reserved, udp, tcp, sctp
		   [7] is ip6
		   [8] clear head/tail
		   [9] packet drop action
		   [10] matched packet generates low-latency interrupt
		   [11] last in linked list
		   [12] collision
		   [15] rx queue enable
		   [22:16] rx queue
		   [29:24] pool. */
		command reg

		_ [0xee3c - 0xee30]byte
		/* ip4 dst/src address, tcp ports, udp ports.
		   set bits mean bit is ignored. */
		ip4_masks           [4]reg
		filter_length       reg
		usage_stats         reg
		failed_usage_stats  reg
		filters_match_stats reg
		filters_miss_stats  reg
		_                   [0xee68 - 0xee60]byte
		/* Lookup, signature. */
		hash_keys [2]reg
		/* [15:0] ip6 src address 1 bit per byte
		   [31:16] ip6 dst address. */
		ip6_mask reg
		/* [0] vlan id
		   [1] vlan priority
		   [2] pool
		   [3] ip protocol
		   [4] flex
		   [5] dst ip6. */
		other_mask reg
		_          [0xf000 - 0xee78]byte
	}

	pf_1 struct {
		l2_control              [64]reg
		vlan_pool_filter        [64]reg
		vlan_pool_filter_bitmap [128]reg
		dst_ethernet_address    [128]reg
		mirror_rule             [4]reg
		mirror_rule_vlan        [8]reg
		mirror_rule_pool        [8]reg
		_                       [0x10010 - 0xf650]byte
	}

	eeprom_flash_control reg
	/* [0] start
	   [1] done
	   [15:2] address
	   [31:16] read data. */
	eeprom_read            reg
	_                      [0x1001c - 0x10018]byte
	flash_access           reg
	_                      [0x10114 - 0x10020]byte
	flash_data             reg
	flash_control          reg
	flash_read_data        reg
	_                      [0x1013c - 0x10120]byte
	flash_opcode           reg
	software_semaphore     reg
	_                      [0x10148 - 0x10144]byte
	firmware_semaphore     reg
	_                      [0x10160 - 0x1014c]byte
	software_firmware_sync reg
	_                      [0x10200 - 0x10164]byte
	general_rx_control     reg
	_                      [0x11000 - 0x10204]byte

	pcie struct {
		control reg
		_       [0x11010 - 0x11004]byte
		/* [3:0] enable counters
		   [7:4] leaky bucket counter mode
		   [29] reset
		   [30] stop
		   [31] start. */
		counter_control reg
		/* [7:0],[15:8],[23:16],[31:24] event for counters 0-3.
		   event codes:
		   0x0 bad tlp
		   0x10 reqs that reached timeout
		   etc. */
		counter_event          reg
		_                      [0x11020 - 0x11018]byte
		counters_clear_on_read [4]reg
		counter_config         [4]reg
		indirect_access        struct {
			address reg
			data    reg
		}
		_                            [0x11050 - 0x11048]byte
		extended_control             reg
		_                            [0x11064 - 0x11054]byte
		mirrored_revision_id         reg
		_                            [0x11070 - 0x11068]byte
		dca_requester_id_information reg

		/* [0] global disable
		   [4:1] mode: 0 => legacy, 1 => dca 1.0. */
		dca_control reg
		_           [0x110b0 - 0x11078]byte
		/* [0] pci completion abort
		   [1] unsupported i/o address
		   [2] wrong byte enable
		   [3] pci timeout */
		pcie_interrupt_status reg
		_                     [0x110b8 - 0x110b4]byte
		pcie_interrupt_enable reg
		_                     [0x110c0 - 0x110bc]byte
		msi_x_pba_clear       [8]reg
		_                     [0x12300 - 0x110e0]byte
	}

	interrupt_throttle1 [128 - 24]reg
	_                   [0x14f00 - 0x124a0]byte

	core_analog_config reg
	_                  [0x14f10 - 0x14f04]byte
	core_common_config reg
	_                  [0x15f14 - 0x14f14]byte

	link_sec_software_firmware_interface reg
	_                                    [0x15f58 - 0x15f18]byte
	sfp_i2c                              struct {
		command reg
		params  reg
	}
}