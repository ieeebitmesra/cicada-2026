// Command server runs the IEEE CTF Submission Portal over SSH (Wish + Bubble Tea).
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ieee-ctf/internal/scoring"
	"ieee-ctf/internal/server"
	"ieee-ctf/internal/store"
)

func main() {
	cfgPath := "configs/server.yaml"
	if v := os.Getenv("CTF_CONFIG"); v != "" {
		cfgPath = v
	}

	cfg, err := server.Load(cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db := store.MustOpen(cfg.Database.Path)
	defer db.Close()

	if err := db.Migrate(cfg.Database.MigrationsDir); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	roundsPath := "configs/rounds.yaml"
	if v := os.Getenv("CTF_ROUNDS"); v != "" {
		roundsPath = v
	}
	roundsCfg, err := server.LoadRounds(roundsPath)
	if err != nil {
		log.Fatalf("load rounds: %v", err)
	}
	if err := db.UpsertRounds(roundsCfg.Rounds); err != nil {
		log.Fatalf("sync rounds: %v", err)
	}

	flags := scoring.NewFlagValidator(
		cfg.Security.Input.MaxFlagLength,
		cfg.Security.Submissions.PerMinutePerTeam,
		time.Duration(cfg.Security.Submissions.CooldownSeconds)*time.Second,
	)
	svc := scoring.NewService(db, roundsCfg, flags,
		time.Duration(cfg.Hints.ChallengeValidityMinutes)*time.Minute)

	srv, err := server.New(cfg, db, svc)
	if err != nil {
		log.Fatalf("create server: %v", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Submission Portal listening on %s:%s", cfg.Server.Host, cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-done
	log.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
