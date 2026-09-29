package metrics

import (
	"time"

	"oktopus/internal/proxy/bytecount"
)

type byteTotals struct {
	upAllow   int64
	upDeny    int64
	downAllow int64
	downDeny  int64
}

func init() {
	bytecount.SetRotateHook(func(sec int64, upAllow, upDeny, downAllow, downDeny uint64) {
		defaultCollector.ingestBytes(sec, upAllow, upDeny, downAllow, downDeny)
	})
}

func (c *Collector) ingestBytes(sec int64, upAllow, upDeny, downAllow, downDeny uint64) {
	if upAllow == 0 && upDeny == 0 && downAllow == 0 && downDeny == 0 {
		return
	}
	c.mu.Lock()
	bt := c.bytesBySec[sec]
	bt.upAllow += int64(upAllow)
	bt.upDeny += int64(upDeny)
	bt.downAllow += int64(downAllow)
	bt.downDeny += int64(downDeny)
	c.bytesBySec[sec] = bt
	c.mu.Unlock()
}

func (c *Collector) mergeBytesIntoBucketsLocked(now time.Time, out []TrafficBucket) {
	cutoffSec := now.Add(-window).Unix()
	nowSec := now.Unix()
	endBucket := nowSec / seriesBucketSec
	startBucket := endBucket - int64(seriesBucketCount-1)

	liveSec, liveUpAllow, liveUpDeny, liveDownAllow, liveDownDeny := bytecount.Live()
	for sec, bt := range c.bytesBySec {
		if sec < cutoffSec {
			continue
		}
		if sec == liveSec {
			continue
		}
		bucketIdx := sec / seriesBucketSec
		if bucketIdx < startBucket || bucketIdx > endBucket {
			continue
		}
		i := int(bucketIdx - startBucket)
		if i < 0 || i >= seriesBucketCount {
			continue
		}
		mergeByteTotalsIntoBucket(&out[i], bt)
	}
	if liveSec >= cutoffSec {
		bucketIdx := liveSec / seriesBucketSec
		if bucketIdx >= startBucket && bucketIdx <= endBucket {
			i := int(bucketIdx - startBucket)
			if i >= 0 && i < seriesBucketCount {
				out[i].BytesUpAllow += int64(liveUpAllow)
				out[i].BytesUpDeny += int64(liveUpDeny)
				out[i].BytesDownAllow += int64(liveDownAllow)
				out[i].BytesDownDeny += int64(liveDownDeny)
				out[i].BytesUp += int64(liveUpAllow + liveUpDeny)
				out[i].BytesDown += int64(liveDownAllow + liveDownDeny)
			}
		}
	}
}

func mergeByteTotalsIntoBucket(b *TrafficBucket, bt byteTotals) {
	b.BytesUpAllow += bt.upAllow
	b.BytesUpDeny += bt.upDeny
	b.BytesDownAllow += bt.downAllow
	b.BytesDownDeny += bt.downDeny
	b.BytesUp += bt.upAllow + bt.upDeny
	b.BytesDown += bt.downAllow + bt.downDeny
}
