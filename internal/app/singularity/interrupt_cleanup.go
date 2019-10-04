// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"context"
	"os"

	"github.com/sylabs/singularity/internal/pkg/sylog"
)

// interruptCleanup will watch for a interrupt signal, if there's
// one detected, then it will remove all the specified file(s)
func interruptCleanup(ctx context.Context, done <-chan struct{}, files ...string) {
	select {
	case <-ctx.Done():
		for _, f := range files {
			sylog.Debugf("Removing file: %q because of receiving termination signal", f)
			err := os.Remove(f)
			if !os.IsNotExist(err) && err != nil {
				sylog.Errorf("unable to remove: %s: %v", f, err)
			}
		}
		os.Exit(1)

	case <-done:
	}
}
