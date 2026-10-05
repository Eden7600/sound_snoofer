//go:build !windows

package windowsaudio

import "context"

func run(ctx context.Context, requests <-chan Request, results chan Result) {
	defer close(results)
	results <- Result{"Windows defaults unsupported"}
	<-ctx.Done()
}
