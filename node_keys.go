// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package config

// Luxd config keys referenced by netrunner/cli.
const (
	HTTPHostKey                  = "http-host"
	BootstrapNodesKey            = "bootstrap-nodes"
	BootstrapIPsKey              = "bootstrap-ips"
	BootstrapIDsKey              = "bootstrap-ids"
	DBPathKey                    = "db-dir"
	LogsDirKey                   = "log-dir"
	TrackChainsKey               = "track-chains"
	ChainConfigDirKey            = "chain-config-dir"
	NetConfigDirKey              = "net-config-dir"
	GenesisFileKey               = "genesis-file"
	StakingTLSKeyPathKey         = "staking-tls-key-file"
	StakingCertPathKey           = "staking-tls-cert-file"
	StakingSignerKeyPathKey      = "staking-signer-key-file"
	SybilProtectionEnabledKey    = "sybil-protection-enabled"
	IndexEnabledKey              = "index-enabled"
	IndexAllowIncompleteKey      = "index-allow-incomplete"
	NetworkAllowPrivateIPsKey    = "network-allow-private-ips"
	PartialSyncPrimaryNetworkKey = "partial-sync-primary-network"
)

// Luxd environment variables referenced by cli helpers.
const (
	LuxNodeDataDirVar = "LUXD_DATA_DIR"
)
