//go:build !windows

package networkwatch

import "context"

func Watch(ctx context.Context, _ func()) error {
	<-ctx.Done()
	return nil
}
