package metrics

import (
	"time"

	"github.com/google/uuid"

	"oktopus/internal/proxy/bytecount"
)

type byteTotals struct {
	upAllow   int64
	upDeny    int64
	downAllow int64
	downDeny  int64
}

func init() {
	bytecount.SetRecordHooks(recordBytesUp, recordBytesDown)
}

func recordBytesUp(id uuid.UUID, n int, denied bool) {
	collectorForID(id).recordBytesUp(n, denied)
}

func recordBytesDown(id uuid.UUID, n int, denied bool) {
	collectorForID(id).recordBytesDown(n, denied)
}

func collectorForID(id uuid.UUID) *Collector {
	if id == uuid.Nil {
		return defaultCollector
	}
	return DefaultRegistry().Collector(id)
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

	liveSec := c.liveByteSec
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
				out[i].BytesUpAllow += c.liveUpAllow
				out[i].BytesUpDeny += c.liveUpDeny
				out[i].BytesDownAllow += c.liveDownAllow
				out[i].BytesDownDeny += c.liveDownDeny
				out[i].BytesUp += c.liveUpAllow + c.liveUpDeny
				out[i].BytesDown += c.liveDownAllow + c.liveDownDeny
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
