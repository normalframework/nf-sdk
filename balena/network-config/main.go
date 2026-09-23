// Command netcfgd configures a balenaOS device's network interfaces and its
// outbound proxy, and serves a local web console for diagnosing both.
//
// Interfaces are configured through NetworkManager's D-Bus API, because the
// balena Supervisor API has no per-interface endpoint and containers cannot
// write NetworkManager keyfiles to the boot partition. The proxy is configured
// through the Supervisor, because that is the only thing which can write
// /mnt/boot/system-proxy/redsocks.conf.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/normalframework/netcfgd/internal/activity"
	"github.com/normalframework/netcfgd/internal/netmgr"
	"github.com/normalframework/netcfgd/internal/proxyprobe"
	"github.com/normalframework/netcfgd/internal/store"
	"github.com/normalframework/netcfgd/internal/supervisor"
	"github.com/normalframework/netcfgd/internal/watchdog"
	"github.com/normalframework/netcfgd/internal/web"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	var (
		addr     = flag.String("listen", env("NETCFG_LISTEN", ":8081"), "address to serve the console on")
		dataDir  = flag.String("data", env("NETCFG_DATA", "/data"), "directory for configuration and state")
		readOnly = flag.Bool("read-only", envBool("NETCFG_READONLY"), "serve the console without allowing changes")
		dryRun   = flag.Bool("fake-supervisor", envBool("NETCFG_FAKE_SUPERVISOR"),
			"log Supervisor changes instead of making them, for development")
		caBundle = flag.String("ca-bundle", env("NETCFG_CA_BUNDLE", ""),
			"PEM bundle of extra CAs to trust, for networks that re-encrypt TLS (default <data>/ca.pem)")
		showVersion = flag.Bool("version", false, "print the version and exit")
	)
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("netcfgd: ")
	log.Printf("starting version %s", version)

	if err := run(*addr, *dataDir, *caBundle, *readOnly, *dryRun); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

func run(addr, dataDir, caBundle string, readOnly, dryRun bool) error {
	cfg, err := store.Open(dataDir)
	if err != nil {
		return err
	}
	if err := cfg.EnsureAuth(); err != nil {
		return err
	}
	state, err := store.OpenState(dataDir)
	if err != nil {
		return err
	}

	acts := activity.New(1000)
	acts.Infof("netcfgd", "", "started", "version "+version)

	// Extra CAs, for networks whose proxy re-encrypts TLS. balenaOS installs
	// its own balenaRootCA into the *host* trust store, which containers do
	// not share, so the same certificate has to be given to us separately or
	// every check reports a certificate failure.
	if caBundle == "" {
		caBundle = filepath.Join(dataDir, "ca.pem")
	}
	if err := proxyprobe.LoadTrustedCAsFile(caBundle); err != nil {
		acts.Errorf("netcfgd", "", "could not load the CA bundle", err.Error())
	} else if proxyprobe.TrustNote() != "" {
		acts.Infof("netcfgd", "", "loaded extra trusted CAs", proxyprobe.TrustNote())
	}

	if cfg.Auth().MustChange {
		log.Printf("WARNING: the console is using the default credentials (%s/%s). "+
			"Sign in and change them.", store.DefaultUsername, store.DefaultPassword)
	}

	// NetworkManager is not fatal: without it the console still serves proxy
	// configuration and diagnostics, and says clearly why interfaces are
	// unavailable. That is far more useful than refusing to start.
	nm, nmErr := netmgr.New()
	if nmErr != nil {
		acts.Errorf("netcfgd", "", "NetworkManager is not reachable", nmErr.Error())
	} else {
		if v, err := nm.Version(); err == nil {
			acts.Infof("netcfgd", "", "connected to NetworkManager", "version "+v)
		}
		defer nm.Close()
	}

	sup := supervisor.New()
	sup.DryRun = dryRun
	switch {
	case dryRun:
		acts.Warnf("netcfgd", "", "running with a fake Supervisor",
			"proxy changes will be logged, not applied")
	case !sup.Available():
		acts.Warnf("netcfgd", "", "the balena Supervisor API is not available",
			"proxy configuration is disabled; the io.balena.features.supervisor-api label is required")
	}

	dog := watchdog.New(cfg, state, sup, acts)

	app, err := web.New(cfg, state, nm, nmErr, sup, dog, acts, web.Options{
		ReadOnly: readOnly,
		Version:  version,
	})
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go dog.Run(ctx)

	srv := &http.Server{
		Addr:    addr,
		Handler: app.Handler(),
		// Long write timeouts would kill the streaming diagnostics, so the
		// read side is bounded and the write side is left open.
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errs := make(chan error, 1)
	go func() {
		log.Printf("console listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errs <- err
		}
	}()

	select {
	case err := <-errs:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
		log.Print("shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envBool(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}
