//go:build !windows

package library

import "context"

func restorePlatform(ctx context.Context, archive, target string, l RestoreLimits) error {
	return restoreWithLimitsLegacy(ctx, archive, target, l)
}
func restoreWorkerPlatform() (bool, int) { return false, 0 }
