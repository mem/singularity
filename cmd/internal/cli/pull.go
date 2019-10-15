// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"context"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/sylabs/scs-library-client/client"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/app/singularity"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	ociclient "github.com/sylabs/singularity/internal/pkg/client/oci"
	scs "github.com/sylabs/singularity/internal/pkg/remote"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/uri"
	net "github.com/sylabs/singularity/pkg/client/net"
	"github.com/sylabs/singularity/pkg/cmdline"
	"github.com/sylabs/singularity/pkg/signing"
)

const (
	// LibraryProtocol holds the sylabs cloud library base URI,
	// for more info refer to https://cloud.sylabs.io/library.
	LibraryProtocol = "library"
	// ShubProtocol holds singularity hub base URI,
	// for more info refer to https://singularity-hub.org/
	ShubProtocol = "shub"
	// HTTPProtocol holds the remote http base URI.
	HTTPProtocol = "http"
	// HTTPSProtocol holds the remote https base URI.
	HTTPSProtocol = "https"
	// OrasProtocol holds the oras URI.
	OrasProtocol = "oras"
)

var (
	// pullLibraryURI holds the base URI to a Sylabs library API instance.
	pullLibraryURI string
	// pullImageName holds the name to be given to the pulled image.
	pullImageName string
	// keyServerURL server URL.
	keyServerURL = "https://keys.sylabs.io"
	// unauthenticatedPull when true; wont ask to keep a unsigned container after pulling it.
	unauthenticatedPull bool
	// pullDir is the path that the containers will be pulled to, if set.
	pullDir string
	// pullArch is the architecture for which containers will be pulled from the
	// SCS library.
	pullArch string
)

// --arch
var pullArchFlag = cmdline.Flag{
	ID:           "pullArchFlag",
	Value:        &pullArch,
	DefaultValue: runtime.GOARCH,
	Name:         "arch",
	Usage:        "architecture to pull from library",
	EnvKeys:      []string{"PULL_ARCH"},
}

// --library
var pullLibraryURIFlag = cmdline.Flag{
	ID:           "pullLibraryURIFlag",
	Value:        &pullLibraryURI,
	DefaultValue: "https://library.sylabs.io",
	Name:         "library",
	Usage:        "download images from the provided library",
	EnvKeys:      []string{"LIBRARY"},
}

// --name
var pullNameFlag = cmdline.Flag{
	ID:           "pullNameFlag",
	Value:        &pullImageName,
	DefaultValue: "",
	Name:         "name",
	Hidden:       true,
	Usage:        "specify a custom image name",
	EnvKeys:      []string{"PULL_NAME"},
}

// --dir
var pullDirFlag = cmdline.Flag{
	ID:           "pullDirFlag",
	Value:        &pullDir,
	DefaultValue: "",
	Name:         "dir",
	Usage:        "download images to the specific directory",
	EnvKeys:      []string{"PULLDIR", "PULLFOLDER"},
}

// --disable-cache
var pullDisableCacheFlag = cmdline.Flag{
	ID:           "pullDisableCacheFlag",
	Value:        &disableCache,
	DefaultValue: false,
	Name:         "disable-cache",
	Usage:        "dont use cached images/blobs and dont create them",
	EnvKeys:      []string{"DISABLE_CACHE"},
}

// -U|--allow-unsigned
var pullAllowUnsignedFlag = cmdline.Flag{
	ID:           "pullAllowUnauthenticatedFlag",
	Value:        &unauthenticatedPull,
	DefaultValue: false,
	Name:         "allow-unsigned",
	ShortHand:    "U",
	Usage:        "do not require a signed container",
	EnvKeys:      []string{"ALLOW_UNSIGNED"},
	Deprecated:   `pull no longer exits with an error code in case of unsigned image. Now the flag only suppress warning message.`,
}

// --allow-unauthenticated
var pullAllowUnauthenticatedFlag = cmdline.Flag{
	ID:           "pullAllowUnauthenticatedFlag",
	Value:        &unauthenticatedPull,
	DefaultValue: false,
	Name:         "allow-unauthenticated",
	ShortHand:    "",
	Usage:        "do not require a signed container",
	EnvKeys:      []string{"ALLOW_UNAUTHENTICATED"},
	Hidden:       true,
}

func init() {
	cmdManager.RegisterCmd(PullCmd)

	cmdManager.RegisterFlagForCmd(&commonForceFlag, PullCmd)
	cmdManager.RegisterFlagForCmd(&pullLibraryURIFlag, PullCmd)
	cmdManager.RegisterFlagForCmd(&pullNameFlag, PullCmd)
	cmdManager.RegisterFlagForCmd(&commonNoHTTPSFlag, PullCmd)
	cmdManager.RegisterFlagForCmd(&commonTmpDirFlag, PullCmd)
	cmdManager.RegisterFlagForCmd(&pullDisableCacheFlag, PullCmd)
	cmdManager.RegisterFlagForCmd(&pullDirFlag, PullCmd)

	cmdManager.RegisterFlagForCmd(&dockerUsernameFlag, PullCmd)
	cmdManager.RegisterFlagForCmd(&dockerPasswordFlag, PullCmd)
	cmdManager.RegisterFlagForCmd(&dockerLoginFlag, PullCmd)

	cmdManager.RegisterFlagForCmd(&buildNoCleanupFlag, PullCmd)
	cmdManager.RegisterFlagForCmd(&pullAllowUnsignedFlag, PullCmd)
	cmdManager.RegisterFlagForCmd(&pullAllowUnauthenticatedFlag, PullCmd)
	cmdManager.RegisterFlagForCmd(&pullArchFlag, PullCmd)
}

// PullCmd singularity pull
var PullCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args:                  cobra.RangeArgs(1, 2),
	PreRun:                sylabsToken,
	Run:                   pullRun,
	Use:                   docs.PullUse,
	Short:                 docs.PullShort,
	Long:                  docs.PullLong,
	Example:               docs.PullExample,
}

func setupContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	signalCh := make(chan os.Signal)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGKILL)

	go func() {
		select {
		case <-signalCh:
			sylog.Debugf("Got cancellation signal, propagating cancellation.")
			cancel()

		case <-ctx.Done():
			sylog.Debugf("Request context done.")
		}
	}()

	return ctx, func() {
		sylog.Debugf("Done. Cleaning up.")
		signal.Stop(signalCh)
		cancel()
	}
}

func pullRun(cmd *cobra.Command, args []string) {
	ctx, cancel := setupContext()
	defer cancel()

	imgCache := getCacheHandle(cache.Config{Disable: disableCache})
	if imgCache == nil {
		sylog.Fatalf("Failed to create an image cache handle")
	}

	pullFrom := args[len(args)-1]
	transport, ref := uri.Split(pullFrom)
	if ref == "" {
		sylog.Errorf("Bad URI %s", pullFrom)
		return
	}

	pullTo := pullImageName
	if pullTo == "" {
		pullTo = args[0]
		if len(args) == 1 {
			if transport == "" {
				pullTo = uri.GetName("library://" + pullFrom)
			} else {
				// TODO: If not library/shub & no name specified, simply put to cache
				pullTo = uri.GetName(pullFrom)
			}
		}
	}

	if pullDir != "" {
		pullTo = filepath.Join(pullDir, pullTo)
	}

	checkOverwrite := func(filename string) bool {
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			return true
		}

		// image already exists, it was either already here or
		// it showed up since the last time we checked
		if !forceOverwrite {
			sylog.Errorf(`Image file %s already exists, will not overwrite`, filename)
			return false
		}

		sylog.Debugf("Overwriting existing file: %s", filename)

		return true
	}

	if ok := checkOverwrite(pullTo); !ok {
		return
	}

	tmpFile, err := ioutil.TempFile(filepath.Dir(pullTo), "")
	if err != nil {
		sylog.Errorf("Cannot create temporary file for download: %s", err)
		return
	}

	tmpDst := tmpFile.Name()

	// FIXME(mem): this is a bad use of the API, it's not only
	// necessary to close the file handle, but it's also necessary
	// to remove it before proceding because some of the functions
	// error out if the file already exists.
	tmpFile.Close()
	os.Remove(tmpDst)

	sylog.Debugf("Downloading image to temporary location %s", tmpDst)

	// remove temporary file when returning
	defer func(filename string) {
		switch err := os.Remove(filename); {
		case os.IsNotExist(err):
			sylog.Debugf(`Temporary file "%s" not found`, filename)

		case err == nil:
			sylog.Debugf(`Removed temporary file "%s"`, filename)

		default:
			sylog.Debugf(`Cannot remove temporary file "%s"`, filename)
		}
	}(tmpDst)

	switch transport {
	case LibraryProtocol, "":
		handlePullFlags(cmd)

		libraryConfig := &client.Config{
			BaseURL:   pullLibraryURI,
			AuthToken: authToken,
		}
		lib, err := singularity.NewLibrary(libraryConfig, imgCache, keyServerURL)
		if err != nil {
			sylog.Errorf("Could not initialize library: %v", err)
			return
		}

		err = lib.Pull(ctx, pullFrom, tmpDst, pullArch)
		if err != nil {
			sylog.Errorf("While pulling library image: %v", err)
			return
		}

		if sigType, err := lib.CheckSignature(ctx, tmpDst); err != nil {
			sylog.Warningf("Cannot verify container %s: %v", pullTo, err)
			return
		} else {
			switch sigType {
			case signing.NoSignature:
				sylog.Warningf("Container is not signed, skipping container verification")
			case signing.RemoteSignature:
				sylog.Warningf("Signing key is not available locally; run 'singularity verify %s' to show who signed it", pullTo)
			case signing.LocalSignature:
				sylog.Infof("Container is trusted - run 'singularity key list' to list your trusted keys")
			}
		}

	case ShubProtocol:
		err := singularity.PullShub(ctx, imgCache, tmpDst, pullFrom, noHTTPS)
		if err != nil {
			sylog.Errorf("While pulling shub image: %v\n", err)
			return
		}

	case OrasProtocol:
		ociAuth, err := makeDockerCredentials(cmd)
		if err != nil {
			sylog.Errorf("Unable to make docker oci credentials: %s", err)
			return
		}

		err = singularity.OrasPull(ctx, imgCache, tmpDst, ref, forceOverwrite, &ociAuth)
		if err != nil {
			sylog.Errorf("While pulling image from oci registry: %v", err)
			return
		}

	case HTTPProtocol, HTTPSProtocol:
		err := net.DownloadImage(ctx, tmpDst, pullFrom)
		if err != nil {
			sylog.Errorf("While pulling from image from http(s): %v\n", err)
			return
		}

	case ociclient.IsSupported(transport):
		ociAuth, err := makeDockerCredentials(cmd)
		if err != nil {
			sylog.Errorf("While creating Docker credentials: %v", err)
			return
		}

		err = singularity.OciPull(ctx, imgCache, tmpDst, pullFrom, tmpDir, &ociAuth, noHTTPS, buildArgs.noCleanUp)
		if err != nil {
			sylog.Errorf("While making image from oci registry: %v", err)
			return
		}

	default:
		sylog.Errorf("Unsupported transport type: %s", transport)
		return
	}

	if ok := checkOverwrite(pullTo); !ok {
		return
	}

	sylog.Debugf("Renaming temporary filename %s to %s", tmpDst, pullTo)
	if err := os.Rename(tmpDst, pullTo); err != nil {
		sylog.Debugf("Error while renaming temporary filename %s to %s: %s", tmpDst, pullTo, err)
	}

	sylog.Infof("Download complete: %s\n", pullTo)
}

func handlePullFlags(cmd *cobra.Command) {
	// if we can load config and if default endpoint is set, use that
	// otherwise fall back on regular authtoken and URI behavior
	endpoint, err := sylabsRemote(remoteConfig)
	if err == scs.ErrNoDefault {
		sylog.Warningf("No default remote in use, falling back to: %v", pullLibraryURI)
		sylog.Debugf("Using default key server url: %v", keyServerURL)
		return
	}
	if err != nil {
		sylog.Fatalf("Unable to load remote configuration: %v", err)
	}

	authToken = endpoint.Token
	if !cmd.Flags().Lookup("library").Changed {
		libraryURI, err := endpoint.GetServiceURI("library")
		if err != nil {
			sylog.Fatalf("Unable to get library service URI: %v", err)
		}
		pullLibraryURI = libraryURI
	}

	keystoreURI, err := endpoint.GetServiceURI("keystore")
	if err != nil {
		sylog.Warningf("Unable to get library service URI: %v, defaulting to %s", err, keyServerURL)
		return
	}
	keyServerURL = keystoreURI
}
