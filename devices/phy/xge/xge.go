// Copyright 2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// 10 GIG E (XGE) PHY IEEE 802.3 clause 45 definitions.
package xge

const (
	PHY_CONTROL           = 0x0
	PHY_CONTROL_RESET     = 1 << 15
	PHY_CONTROL_LOOPBACK  = 1 << 14
	PHY_CONTROL_POWERDOWN = 1 << 11

	PHY_STATUS                   = 0x1
	PHY_STATUS_LOCAL_FAULT       = 1 << 7
	PHY_STATUS_LINK_UP           = 1 << 2
	PHY_STATUS_POWERDOWN_ABILITY = 1 << 1

	PHY_ID1 = 0x2
	PHY_ID2 = 0x3

	PHY_SPEED_ABILITY = 0x4

	// IEEE standard device types.
	PHY_DEV_TYPE_CLAUSE_22 = 0
	PHY_DEV_TYPE_PMA_PMD   = 1
	PHY_DEV_TYPE_WIS       = 2
	PHY_DEV_TYPE_PCS       = 3
	PHY_DEV_TYPE_PHY_XS    = 4
	PHY_DEV_TYPE_DTE_XS    = 5
	PHY_DEV_TYPE_VENDOR_1  = 30
	PHY_DEV_TYPE_VENDOR_2  = 31

	// 2 16 bit bitmaps of which devices are present.
	PHY_DEV_TYPES_PRESENT1 = 0x5
	PHY_DEV_TYPES_PRESENT2 = 0x6

	PHY_CONTROL2 = 7

	PHY_STATUS2 = 0x8

	PHY_PMD_TX_DISABLE = 0x9

	// [4:1] 4 lane status, [0] global signal detect.
	PHY_PMD_SIGNAL_DETECT            = 0xa
	PHY_PMD_SIGNAL_DETECT_GLOBAL     = 1 << 0
	PHY_PMD_SIGNAL_DETECT_LANE_SHIFT = 1

	PHY_PACKAGE_ID1 = 0xe
	PHY_PACKAGE_ID2 = 0xf

	// PCS specific.
	PHY_PCS_10G_BASE_X_STATUS = 0x18

	PHY_PCS_10G_BASE_R_STATUS                   = 0x20
	PHY_PCS_10G_BASE_R_STATUS_RX_LINK_STATUS    = 1 << 12
	PHY_PCS_10G_BASE_R_STATUS_HI_BIT_ERROR_RATE = 1 << 1
	PHY_PCS_10G_BASE_R_STATUS_BLOCK_LOCK        = 1 << 0

	PHY_PCS_10G_BASE_R_STATUS2                 = 0x21
	PHY_PCS_10G_BASE_R_JITTER_TEST_CONTROL     = 0x2a
	PHY_PCS_10G_BASE_R_JITTER_TEST_ERROR_COUNT = 0x2b

	// XS specific.
	PHY_XS_LANE_STATUS                    = 0x18
	PHY_XS_LANE_STATUS_TX_LANES_ALIGNED   = 1 << 12
	PHY_XS_LANE_STATUS_LANES_SYNCED_SHIFT = 0

	PHY_XS_TEST_CONTROL = 0x19
)

func PHY_PCS_10G_BASE_R_JITTER_TEST(ab, i int) int { return 0x22 + 4*ab + i }
