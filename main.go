// Braise — le feu de camp de votre réseau local.
//
// Un binaire, un dossier partagé, un chat et un point de montage WebDAV.
package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/grandcat/zeroconf"

	"braise/internal/files"
	"braise/internal/netutil"
	"braise/internal/server"
	"braise/internal/sse"
	"braise/internal/store"
)

var version = "0.1.0"

//go:embed all:web
var webAssets embed.FS

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "braise:", err)
		os.Exit(1)
	}
}

func defaultDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "Braise"
	}
	return filepath.Join(home, "Braise")
}

func run() error {
	var (
		dir      = flag.String("dir", defaultDir(), "dossier partagé (créé si absent)")
		bind     = flag.String("bind", "0.0.0.0", "adresse d'écoute (0.0.0.0 = tout le réseau local)")
		port     = flag.Int("port", 8080, "port HTTP")
		name     = flag.String("name", "Braise", "nom affiché de l'espace")
		readOnly = flag.Bool("read-only", false, "interdire dépôt, création et suppression")
		noOpen   = flag.Bool("no-open", false, "ne pas ouvrir le navigateur au démarrage")
		noMDNS   = flag.Bool("no-mdns", false, "désactiver l'annonce mDNS (braise.local)")
		v        = flag.Bool("version", false, "afficher la version et quitter")
	)
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Braise %s — partage de fichiers et chat sur le réseau local\n\n", version)
		fmt.Fprintf(flag.CommandLine.Output(), "Usage : braise [options]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *v {
		fmt.Println("Braise", version)
		return nil
	}
	if *port < 1 || *port > 65535 {
		return fmt.Errorf("port invalide : %d", *port)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	root, err := filepath.Abs(*dir)
	if err != nil {
		return err
	}
	fm, err := files.New(root, *readOnly)
	if err != nil {
		return fmt.Errorf("dossier partagé : %w", err)
	}
	st, err := store.Open(fm.Root())
	if err != nil {
		return fmt.Errorf("stockage : %w", err)
	}
	defer st.Close()

	webSub, err := fs.Sub(webAssets, "web")
	if err != nil {
		return err
	}

	urls := urlsForBind(*bind, *port)
	hub := sse.New()
	started := time.Now().UTC()

	handler := server.New(server.Config{
		Name:      *name,
		Version:   version,
		Root:      fm.Root(),
		Port:      *port,
		ReadOnly:  *readOnly,
		URLs:      urls,
		Web:       http.FS(webSub),
		Store:     st,
		Files:     fm,
		Hub:       hub,
		Logger:    logger,
		StartedAt: started,
	})

	httpSrv := &http.Server{
		Addr:              net.JoinHostPort(*bind, fmt.Sprint(*port)),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	ln, err := net.Listen("tcp", httpSrv.Addr)
	if err != nil {
		return fmt.Errorf("écoute sur %s : %w (le port est peut-être déjà utilisé, essayez --port 8081)", httpSrv.Addr, err)
	}

	printBanner(*name, fm.Root(), *readOnly, urls)

	// Presence pruning + periodic roster broadcast.
	go func() {
		t := time.NewTicker(10 * time.Second)
		defer t.Stop()
		for range t.C {
			peers, changed := hub.Prune()
			if changed {
				hub.BroadcastJSON("presence", peers)
			}
		}
	}()

	// mDNS advertisement, best effort.
	if !*noMDNS {
		if ips := netutil.LANIPs(); len(ips) > 0 {
			mdns, err := zeroconf.RegisterProxy(
				"Braise", "_http._tcp", "local.", *port,
				"braise.local.", ips,
				[]string{"path=/", "app=braise", "version=" + version}, nil,
			)
			if err != nil {
				logger.Warn("mDNS indisponible", "err", err)
			} else {
				defer mdns.Shutdown()
				logger.Info("mDNS actif", "url", fmt.Sprintf("http://braise.local:%d", *port))
			}
		}
	}

	// Graceful shutdown.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stop
		fmt.Println("\nExtinction de la braise…")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(ctx)
	}()

	if !*noOpen {
		go func() {
			time.Sleep(350 * time.Millisecond)
			local := fmt.Sprintf("http://localhost:%d", *port)
			if err := netutil.OpenBrowser(local); err != nil {
				logger.Debug("ouverture du navigateur impossible", "err", err)
			}
		}()
	}

	logger.Info("Braise démarre", "version", version, "root", fm.Root(), "addr", httpSrv.Addr)
	if err := httpSrv.Serve(ln); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func urlsForBind(bind string, port int) []string {
	switch bind {
	case "", "0.0.0.0", "::", "[::]":
		return netutil.URLs(port)
	case "127.0.0.1", "localhost", "::1":
		return []string{fmt.Sprintf("http://localhost:%d", port)}
	default:
		host := strings.Trim(bind, "[]")
		return []string{fmt.Sprintf("http://%s:%d", host, port)}
	}
}

func printBanner(name, root string, readOnly bool, urls []string) {
	fmt.Printf("\n  Braise %s — le feu de camp de votre réseau local.\n\n", version)
	fmt.Printf("  Espace    %s\n", name)
	fmt.Printf("  Dossier   %s\n", root)
	if len(urls) > 0 {
		fmt.Printf("  Web       %s\n", urls[0])
		for _, u := range urls[1:] {
			fmt.Printf("            %s\n", u)
		}
		if len(urls) > 1 {
			fmt.Printf("  WebDAV    %s/dav/\n", urls[0])
		}
	}
	if len(urls) == 1 && strings.Contains(urls[0], "localhost") {
		fmt.Printf("  Astuce    aucune adresse LAN privée détectée ; ouvrez Braise sur la machine hôte.\n")
	}
	mode := "lecture/écriture"
	if readOnly {
		mode = "lecture seule"
	}
	fmt.Printf("  Mode      %s\n", mode)
	fmt.Printf("  Arrêt     Ctrl+C\n\n")
}
