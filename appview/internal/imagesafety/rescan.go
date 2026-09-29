package imagesafety

import "time"

func NeedsRescan(result Result, required ScanKey, targeted bool, _ time.Time) bool {
	return targeted || !result.ReusableFor(required)
}
