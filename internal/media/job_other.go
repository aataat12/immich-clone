//go:build !windows

package media

import "context"

func Run(context.Context, string, []string, func(uint32)) (Result, error) {
	return Result{}, ErrUnavailable
}
