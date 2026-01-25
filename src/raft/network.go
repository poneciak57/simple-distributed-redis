package raft

// Very simple network abstraction might be extended later

import (
	. "main/src/config"
)

type Peer struct {
	ID        string
	Address   string
	Available bool
}

type Network struct {
	peers []Peer
	me    string // my ID in the network
}

func NewNetwork(peerInfos NetworkConfig) *Network {
	peers := make([]Peer, 0, len(peerInfos.Peers))
	for _, p := range peerInfos.Peers {
		peers = append(peers, Peer{
			ID:        p.ID,
			Address:   p.Address,
			Available: true,
		})
	}
	return &Network{
		peers: peers,
		me:    peerInfos.Self.ID,
	}
}

// Iterator over available peers
func (n *Network) AvailablePeersIterator(excludeSelf bool) func(func(Peer) bool) {
	return func(yield func(Peer) bool) {
		for _, peer := range n.peers {
			if peer.Available && (!excludeSelf || peer.ID != n.me) {
				if !yield(peer) {
					break
				}
			}
		}
	}
}

func (n *Network) GetMe() string {
	return n.me
}

func (n *Network) IsMe(id string) bool {
	return n.me == id
}
