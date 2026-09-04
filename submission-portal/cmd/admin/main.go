// Command admin is the operator CLI for the CTF submission portal.
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"ieee-ctf/internal/auth"
	"ieee-ctf/internal/scoring"
	"ieee-ctf/internal/server"
	"ieee-ctf/internal/store"
)

const usage = `IEEE CTF admin CLI

Usage:
  admin teams list
  admin teams create   --name NAME --ssh-user USER --password PASS
  admin teams import   --csv teams.csv            (columns: name,ssh_user,password)
  admin teams reset-password --user USER
  admin teams delete   --user USER

  admin rounds list
  admin rounds set-active --round N --active true|false
  admin rounds load    --file configs/rounds.yaml
  admin rounds hash-flag --flag 'PANTHEON{...}'

  admin scoreboard
  admin scoreboard export --format csv --output scores.csv
  admin hints list [--team SSH_USER]

  admin db migrate [--migrations migrations]
  admin db backup  --output backup.sql
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}
	group, rest := os.Args[1], os.Args[2:]

	var err error
	switch group {
	case "teams":
		err = cmdTeams(rest)
	case "rounds":
		err = cmdRounds(rest)
	case "scoreboard":
		err = cmdScoreboard(rest)
	case "hints":
		err = cmdHints(rest)
	case "db":
		err = cmdDB(rest)
	case "-h", "--help", "help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", group, usage)
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// openDB opens + migrates the database for commands that need it.
func openDB() (*store.DB, func(), error) {
	cfgPath := envOr("CTF_CONFIG", "configs/server.yaml")
	var target, token, migDir string

	if cfg, err := server.Load(cfgPath); err == nil {
		target, token = cfg.DatabaseTarget()
		migDir = cfg.Database.MigrationsDir
	}

	if v := os.Getenv("CTF_DB_URL"); v != "" {
		target = v
	} else if v := os.Getenv("TURSO_DATABASE_URL"); v != "" {
		target = v
	}
	if v := os.Getenv("CTF_DB_AUTH_TOKEN"); v != "" {
		token = v
	} else if v := os.Getenv("TURSO_AUTH_TOKEN"); v != "" {
		token = v
	} else if v := os.Getenv("LIBSQL_AUTH_TOKEN"); v != "" {
		token = v
	}
	if v := os.Getenv("CTF_DB_PATH"); v != "" && target == "" {
		target = v
	}
	if target == "" {
		target = "ctf.db"
	}
	if migDir == "" {
		migDir = envOr("CTF_MIGRATIONS", "migrations")
	}

	db, err := store.OpenWithAuth(target, token)
	if err != nil {
		return nil, nil, err
	}
	if err := db.Migrate(migDir); err != nil {
		db.Close()
		return nil, nil, err
	}
	return db, func() { db.Close() }, nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	return fs
}

/* ----------------------------- teams ----------------------------- */

func cmdTeams(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("teams: missing subcommand (list|create|import|reset-password|delete)")
	}
	sub, rest := args[0], args[1:]
	db, done, err := openDB()
	if err != nil {
		return err
	}
	defer done()

	switch sub {
	case "list":
		teams, err := db.ListTeams()
		if err != nil {
			return err
		}
		w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tSSH USER\tREGISTERED\tPGP")
		for _, t := range teams {
			pgp := "—"
			if t.PGPPubkey != "" {
				pgp = auth.PublicKeyFingerprint(t.PGPPubkey)
			}
			fmt.Fprintf(w, "%d\t%s\t%s\t%v\t%s\n", t.ID, t.Name, t.SSHUser, t.Registered, pgp)
		}
		return w.Flush()

	case "create":
		fs := newFlagSet("teams create")
		name := fs.String("name", "", "team display name")
		user := fs.String("ssh-user", "", "ssh username")
		pass := fs.String("password", "", "initial password")
		fs.Parse(rest)
		if *name == "" || *user == "" || *pass == "" {
			return fmt.Errorf("--name, --ssh-user and --password are required")
		}
		hash, err := auth.HashPassword(*pass)
		if err != nil {
			return err
		}
		id, err := db.CreateTeam(*name, strings.ToLower(*user), hash)
		if err != nil {
			return err
		}
		fmt.Printf("created team #%d (%s / ssh:%s)\n", id, *name, *user)

	case "import":
		fs := newFlagSet("teams import")
		path := fs.String("csv", "", "CSV file with header: name,ssh_user,password")
		fs.Parse(rest)
		if *path == "" {
			return fmt.Errorf("--csv is required")
		}
		f, err := os.Open(*path)
		if err != nil {
			return err
		}
		defer f.Close()
		r := csv.NewReader(f)
		header, err := r.Read()
		if err != nil {
			return err
		}
		want := []string{"name", "ssh_user", "password"}
		for i, h := range want {
			if i >= len(header) || strings.TrimSpace(header[i]) != h {
				return fmt.Errorf("csv header must be name,ssh_user,password")
			}
		}
		n := 0
		for {
			rec, err := r.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			if len(rec) < 3 {
				continue
			}
			hash, err := auth.HashPassword(rec[2])
			if err != nil {
				return fmt.Errorf("%s: %w", rec[0], err)
			}
			if _, err := db.CreateTeam(rec[0], strings.ToLower(rec[1]), hash); err != nil {
				return fmt.Errorf("%s: %w", rec[0], err)
			}
			n++
		}
		fmt.Printf("imported %d teams\n", n)

	case "reset-password":
		fs := newFlagSet("teams reset-password")
		user := fs.String("user", "", "ssh username")
		fs.Parse(rest)
		if *user == "" {
			return fmt.Errorf("--user is required")
		}
		t, err := db.GetTeamBySSHUser(strings.ToLower(*user))
		if err != nil {
			return err
		}
		pass, err := auth.ResetPassword(db, t.ID)
		if err != nil {
			return err
		}
		fmt.Printf("new password for %s: %s\n", t.SSHUser, pass)

	case "delete":
		fs := newFlagSet("teams delete")
		user := fs.String("user", "", "ssh username")
		fs.Parse(rest)
		if *user == "" {
			return fmt.Errorf("--user is required")
		}
		t, err := db.GetTeamBySSHUser(strings.ToLower(*user))
		if err != nil {
			return err
		}
		if err := db.DeleteTeam(t.ID); err != nil {
			return err
		}
		fmt.Printf("deleted team %s\n", t.Name)

	default:
		return fmt.Errorf("teams: unknown subcommand %q", sub)
	}
	return nil
}

/* ----------------------------- rounds ----------------------------- */

func cmdRounds(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("rounds: missing subcommand (list|set-active|load|hash-flag)")
	}
	sub, rest := args[0], args[1:]

	switch sub {
	case "hash-flag":
		fs := newFlagSet("rounds hash-flag")
		fl := fs.String("flag", "", "the flag to hash")
		fs.Parse(rest)
		if *fl == "" {
			return fmt.Errorf("--flag is required")
		}
		fmt.Println(scoring.HashFlag(*fl))
		return nil
	}

	db, done, err := openDB()
	if err != nil {
		return err
	}
	defer done()

	switch sub {
	case "list":
		rounds, err := db.ListRounds(false)
		if err != nil {
			return err
		}
		w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tPOINTS\tACTIVE\tFLAG HASH (first 12)")
		for _, r := range rounds {
			active := "no"
			if r.IsActive {
				active = "yes"
			}
			short := r.FlagHash
			if len(short) > 12 {
				short = short[:12]
			}
			fmt.Fprintf(w, "%d\t%s\t%d\t%s\t%s…\n", r.ID, r.Name, r.Points, active, short)
		}
		return w.Flush()

	case "set-active":
		fs := newFlagSet("rounds set-active")
		id := fs.Int("round", 0, "round id")
		active := fs.Bool("active", true, "activate or deactivate")
		fs.Parse(rest)
		if *id == 0 {
			return fmt.Errorf("--round is required")
		}
		if err := db.SetRoundActive(*id, *active); err != nil {
			return err
		}
		fmt.Printf("round %d active=%v\n", *id, *active)

	case "load":
		fs := newFlagSet("rounds load")
		file := fs.String("file", "configs/rounds.yaml", "rounds yaml file")
		fs.Parse(rest)
		rc, err := server.LoadRounds(*file)
		if err != nil {
			return err
		}
		if err := db.UpsertRounds(rc.Rounds); err != nil {
			return err
		}
		fmt.Printf("loaded %d rounds from %s\n", len(rc.Rounds), *file)

	default:
		return fmt.Errorf("rounds: unknown subcommand %q", sub)
	}
	return nil
}

/* ---------------------------- scoreboard ---------------------------- */

func cmdScoreboard(args []string) error {
	sub, rest := "show", args
	if len(args) > 0 && args[0] != "export" && args[0] != "show" {
		return fmt.Errorf("scoreboard: unknown subcommand %q", args[0])
	}
	if len(args) > 0 {
		sub, rest = args[0], args[1:]
	}

	db, done, err := openDB()
	if err != nil {
		return err
	}
	defer done()

	entries, err := db.Scoreboard()
	if err != nil {
		return err
	}

	switch sub {
	case "show":
		w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
		fmt.Fprintln(w, "RANK\tTEAM\tSCORE\tSOLVED")
		for i, e := range entries {
			fmt.Fprintf(w, "%d\t%s\t%.1f\t%d\n", i+1, e.TeamName, e.TotalScore, e.RoundsSolved)
		}
		return w.Flush()

	case "export":
		fs := newFlagSet("scoreboard export")
		out := fs.String("output", "scores.csv", "output csv path")
		fs.Parse(rest)
		f, err := os.Create(*out)
		if err != nil {
			return err
		}
		defer f.Close()
		w := csv.NewWriter(f)
		if err := w.Write([]string{"rank", "team", "score", "rounds_solved"}); err != nil {
			return err
		}
		for i, e := range entries {
			if err := w.Write([]string{
				fmt.Sprint(i + 1), e.TeamName,
				fmt.Sprintf("%.1f", e.TotalScore), fmt.Sprint(e.RoundsSolved),
			}); err != nil {
				return err
			}
		}
		w.Flush()
		fmt.Printf("exported %d rows to %s\n", len(entries), *out)
		return w.Error()
	}
	return nil
}

/* ------------------------------- hints ------------------------------- */

func cmdHints(args []string) error {
	sub, rest := "list", args
	if len(args) > 0 {
		sub, rest = args[0], args[1:]
	}
	if sub != "list" {
		return fmt.Errorf("hints: unknown subcommand %q", sub)
	}

	fs := newFlagSet("hints list")
	user := fs.String("team", "", "filter by ssh user")
	fs.Parse(rest)

	db, done, err := openDB()
	if err != nil {
		return err
	}
	defer done()

	var teamID int64
	teamName := "(all)"
	if *user != "" {
		t, err := db.GetTeamBySSHUser(strings.ToLower(*user))
		if err != nil {
			return err
		}
		teamID = t.ID
		teamName = t.Name
	}

	rows, err := hintRows(db, teamID)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	fmt.Fprintf(w, "TEAM\tROUND\tTYPE\t#\tCOST\tWHEN\n")
	for _, r := range rows {
		if *user != "" && r.team != teamName {
			continue
		}
		fmt.Fprintf(w, "%s\t%d\t%s\t%d\t-%.1f\t%s\n", r.team, r.round, r.typ, r.idx, r.cost, r.at)
	}
	return w.Flush()
}

type hintRow struct {
	team  string
	round int
	typ   string
	idx   int
	cost  float64
	at    string
}

func hintRows(db *store.DB, teamID int64) ([]hintRow, error) {
	allTeams, err := db.ListTeams()
	if err != nil {
		return nil, err
	}
	names := map[int64]string{}
	for _, t := range allTeams {
		names[t.ID] = t.Name
	}
	var out []hintRow
	for id, name := range names {
		if teamID != 0 && id != teamID {
			continue
		}
		hints, err := db.TeamHints(id)
		if err != nil {
			return nil, err
		}
		for _, h := range hints {
			out = append(out, hintRow{
				team:  name,
				round: h.RoundID,
				typ:   h.HintType,
				idx:   h.HintIndex + 1,
				cost:  h.CostPoints,
				at:    h.UsedAt.Format("2006-01-02 15:04"),
			})
		}
	}
	return out, nil
}

/* --------------------------------- db --------------------------------- */

func cmdDB(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("db: missing subcommand (migrate|backup)")
	}
	sub, rest := args[0], args[1:]

	db, done, err := openDB()
	if err != nil {
		return err
	}
	defer done()

	switch sub {
	case "migrate":
		dir := "migrations"
		fs := newFlagSet("db migrate")
		fs.StringVar(&dir, "migrations", dir, "migrations directory")
		fs.Parse(rest)
		if err := db.Migrate(dir); err != nil {
			return err
		}
		fmt.Printf("migrations applied (dir=%s)\n", dir)

	case "backup":
		fs := newFlagSet("db backup")
		out := fs.String("output", "backup.sql", "output file")
		fs.Parse(rest)
		if *out == "" {
			return fmt.Errorf("--output is required")
		}
		if err := db.Backup(*out); err != nil {
			return err
		}
		fmt.Printf("backup written to %s\n", *out)

	default:
		return fmt.Errorf("db: unknown subcommand %q", sub)
	}
	return nil
}
