//go:build !darwin || !cgo

package capture

import "context"

func platformListWindows(ctx context.Context) ([]WindowInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, ErrUnsupported
}
