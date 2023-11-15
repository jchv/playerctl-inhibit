package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coreos/go-systemd/v22/login1"
	"github.com/godbus/dbus/v5"
	"github.com/leberKleber/go-mpris"
)

const playerctldServiceName = "org.mpris.MediaPlayer2.playerctld"

func main() {
	inhibitWhat := flag.String("inhibit-what", "sleep:handle-lid-switch", "what to inhibit; colon-separated")
	inhibitPaused := flag.Bool("inhibit-while-paused", false, "whether to inhibit if media is paused but not stopped")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
		signal := <-signalCh
		log.Printf("received %v signal, cleaning up...", signal)
		cancel()
	}()

	logind, err := login1.New()
	if err != nil {
		log.Fatalf("could not connect to logind: %v", err)
	}
	defer logind.Close()

	var inhibitFile *os.File

	acquireInhibit := func() {
		if inhibitFile == nil {
			log.Printf("acquiring inhibit lock")
			inhibitFile, err = logind.Inhibit(*inhibitWhat, "MPRIS Inhibit", "Media is currently playing", "block")
			if err != nil {
				log.Printf("could not acquire inhibit lock: %v", err)
			}
		}
	}
	releaseInhibit := func() {
		if inhibitFile != nil {
			log.Printf("releasing inhibit lock")
			if err := inhibitFile.Close(); err != nil {
				log.Printf("error releasing inhibit lock: %v", err)
			}
			inhibitFile = nil
		}
	}

	func() {
		defer releaseInhibit()
		for shouldInhibit := range runMprisChannel(ctx, mprisOptions{InhibitPaused: *inhibitPaused}) {
			if shouldInhibit {
				acquireInhibit()
			} else {
				releaseInhibit()
			}
		}
	}()

	log.Printf("exiting...")
}

type mprisOptions struct {
	InhibitPaused bool
}

// shouldInhibit returns whether the status should inhibit for these options.
func (opts mprisOptions) shouldInhibit(status mpris.PlaybackStatus) bool {
	if opts.InhibitPaused && status == mpris.PlaybackStatusPaused {
		return true
	}
	return status == mpris.PlaybackStatusPlaying
}

// runMprisChannel spawns a goroutine that connects to playerctld via dbus/MPRIS and polls for player status changes.
// It returns a channel that emits a boolean value specifying whether or not to inhibit.
// It returns when the provided context is closed.
// When it exits, the channel returned will be closed.
func runMprisChannel(ctx context.Context, opts mprisOptions) chan bool {
	ch := make(chan bool)
	last := false
	updateStatus := func(status mpris.PlaybackStatus) {
		shouldInhibit := opts.shouldInhibit(status)
		if shouldInhibit != last {
			ch <- shouldInhibit
			last = shouldInhibit
		}
	}
	go func() {
		defer close(ch)
		for ctx.Err() == nil {
			log.Printf("connecting to playerctld")
			player, err := mpris.NewPlayer(playerctldServiceName)
			if err != nil {
				log.Printf("could not connect to playerctld: %v; waiting 5s for playerctld", err)
				select {
				case <-time.After(5 * time.Second):
					continue
				case <-ctx.Done():
					return
				}
			}

		playerctldLoop:
			for ctx.Err() == nil {
				status, err := player.PlaybackStatus()
				if err != nil {
					derr := dbus.Error{}
					if errors.As(err, &derr) {
						if derr.Name == "com.github.altdesktop.playerctld.NoActivePlayer" {
							status = mpris.PlaybackStatusStopped
							goto noPlayer
						}
					}
					log.Printf("error getting player status: %v", err)
					select {
					case <-time.After(5 * time.Second):
						break playerctldLoop
					case <-ctx.Done():
						return
					}
				}
			noPlayer:
				updateStatus(status)
				select {
				case <-time.After(time.Second):
					continue
				case <-ctx.Done():
					break playerctldLoop
				}
			}

			if err := player.Close(); err != nil {
				log.Printf("warning: error while closing playerctld connection: %v", err)
			} else {
				log.Printf("closed playerctld connection")
			}
		}
	}()
	return ch
}
