package metrics

import "time"

func (c *Collector) recordBytesUp(n int, denied bool) {
	if n <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.advanceByteSecLocked(time.Now().Unix())
	if denied {
		c.liveUpDeny += int64(n)
		c.totalUpDeny += uint64(n)
	} else {
		c.liveUpAllow += int64(n)
		c.totalUpAllow += uint64(n)
	}
}

func (c *Collector) recordBytesDown(n int, denied bool) {
	if n <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.advanceByteSecLocked(time.Now().Unix())
	if denied {
		c.liveDownDeny += int64(n)
		c.totalDownDeny += uint64(n)
	} else {
		c.liveDownAllow += int64(n)
		c.totalDownAllow += uint64(n)
	}
}

func (c *Collector) advanceByteSecLocked(sec int64) {
	if c.liveByteSec == 0 {
		c.liveByteSec = sec
		return
	}
	if sec <= c.liveByteSec {
		return
	}
	c.flushLiveBytesToSecLocked(c.liveByteSec)
	c.liveByteSec = sec
	c.liveUpAllow = 0
	c.liveUpDeny = 0
	c.liveDownAllow = 0
	c.liveDownDeny = 0
}

func (c *Collector) flushLiveBytesToSecLocked(sec int64) {
	if c.liveUpAllow == 0 && c.liveUpDeny == 0 && c.liveDownAllow == 0 && c.liveDownDeny == 0 {
		return
	}
	bt := c.bytesBySec[sec]
	bt.upAllow += c.liveUpAllow
	bt.upDeny += c.liveUpDeny
	bt.downAllow += c.liveDownAllow
	bt.downDeny += c.liveDownDeny
	c.bytesBySec[sec] = bt
}
