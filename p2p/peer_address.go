package p2p

import (
	"github.com/libp2p/go-libp2p-peerstore/addr"
	"github.com/libp2p/go-libp2p/core/peer"
)

func (c *Communication) ExportPeerAddress() map[peer.ID]addr.AddrList {
	peerStore := c.dht.Host().Peerstore()
	peers := peerStore.Peers()
	addressBook := make(map[peer.ID]addr.AddrList)
	for _, el := range peers {
		addrs := peerStore.Addrs(el)
		addressBook[el] = addrs
	}
	return addressBook
}
