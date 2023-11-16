package known_node

import (
	"sync/atomic"
)

type KnownNode struct {
	URL    string
	IsSeed bool
}

type KnownNodeScored struct {
	*KnownNode
	score int32 //use atomic
}

var (
	KNOWN_KNODE_SCORE_MINIMUM        = int32(-1000)
	KNOWN_KNODE_SCORE_MINIMUM_DELAY  = int32(-40)
	KNOWN_KNODE_SCORE_SERVER_MAXIMUM = int32(1000)
	KNOWN_KNODE_SCORE_CLIENT_MAXIMUM = int32(100)
)

func (self *KnownNodeScored) IncreaseScore(delta int32, isServer bool) (bool, int32) {

	newScore := atomic.AddInt32(&self.score, delta)

	if newScore > KNOWN_KNODE_SCORE_CLIENT_MAXIMUM && !isServer {
		atomic.StoreInt32(&self.score, KNOWN_KNODE_SCORE_CLIENT_MAXIMUM)
		return false, KNOWN_KNODE_SCORE_CLIENT_MAXIMUM
	}
	if newScore > KNOWN_KNODE_SCORE_SERVER_MAXIMUM && isServer {
		atomic.StoreInt32(&self.score, KNOWN_KNODE_SCORE_SERVER_MAXIMUM)
		return false, KNOWN_KNODE_SCORE_SERVER_MAXIMUM
	}

	return true, newScore
}

func (self *KnownNodeScored) DecreaseScore(delta int32, isServer bool) (bool, bool, int32) {

	newScore := atomic.AddInt32(&self.score, delta)
	if newScore < KNOWN_KNODE_SCORE_MINIMUM {
		if !self.IsSeed {
			return true, true, KNOWN_KNODE_SCORE_MINIMUM
		}
		atomic.StoreInt32(&self.score, KNOWN_KNODE_SCORE_MINIMUM)
		return true, false, KNOWN_KNODE_SCORE_MINIMUM
	}
	return true, false, newScore
}

func (self *KnownNodeScored) GetScore() int32 {
	return atomic.LoadInt32(&self.score)
}

func NewKnownNodeScored(node *KnownNode) *KnownNodeScored {
	return &KnownNodeScored{node, 0}
}
