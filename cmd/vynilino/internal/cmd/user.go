package cmd

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"zntr.io/vynilino/internal/adapter/storage/sqlite"
	"zntr.io/vynilino/internal/app"
	"zntr.io/vynilino/internal/domain"
)

// UserCmd returns the `user` parent command group.
func UserCmd() *cobra.Command {
	defaultDB := os.Getenv("VYNILINO_DB_PATH")
	if defaultDB == "" {
		defaultDB = "./vynilino.db"
	}

	var dbPath string

	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage users",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
		},
	}
	cmd.PersistentFlags().StringVar(&dbPath, "db", defaultDB, "path to SQLite database (default: $VYNILINO_DB_PATH or ./vynilino.db)")

	cmd.AddCommand(userListCmd(&dbPath))
	cmd.AddCommand(userAddCmd(&dbPath))
	cmd.AddCommand(userDeactivateCmd(&dbPath))
	cmd.AddCommand(userActivateCmd(&dbPath))
	cmd.AddCommand(userChangePasswordCmd(&dbPath))

	return cmd
}

func openUserRepo(dbPath string) (domain.UserRepository, *sql.DB, error) {
	db, err := sqlite.Open(context.Background(), dbPath, time.Minute)
	if err != nil {
		return nil, nil, fmt.Errorf("open db %s: %w", dbPath, err)
	}
	return sqlite.NewUserRepository(db), db, nil
}

func userListCmd(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all users",
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, db, err := openUserRepo(*dbPath)
			if err != nil {
				return err
			}
			defer db.Close()

			users, err := repo.ListAll(context.Background())
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tEMAIL\tROLE\tACTIVE\tCREATED")
			for _, u := range users {
				active := "yes"
				if !u.Active {
					active = "no"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					u.ID, u.Email, string(u.Role), active, u.CreatedAt.Format("2006-01-02"))
			}
			return w.Flush()
		},
	}
}

func userAddCmd(dbPath *string) *cobra.Command {
	var email string
	var passwordStdin bool

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new user",
		RunE: func(cmd *cobra.Command, args []string) error {
			password, err := readPassword(passwordStdin, "Password: ")
			if err != nil {
				return err
			}

			if err := app.ValidatePasswordStrength(password); err != nil {
				return err
			}

			hash, err := app.HashPassword(password)
			if err != nil {
				return fmt.Errorf("hash password: %w", err)
			}

			repo, db, err := openUserRepo(*dbPath)
			if err != nil {
				return err
			}
			defer db.Close()

			ctx := context.Background()
			count, err := repo.Count(ctx)
			if err != nil {
				return err
			}
			role := domain.RoleUser
			if count == 0 {
				role = domain.RoleAdmin
			}

			user, err := repo.Create(ctx, &domain.User{
				Email:        email,
				PasswordHash: hash,
				Role:         role,
				Active:       true,
			})
			if err != nil {
				return err
			}

			fmt.Printf("Created user %s (%s) with role %s\n", user.ID, user.Email, string(user.Role))
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "user email address (required)")
	_ = cmd.MarkFlagRequired("email")
	cmd.Flags().BoolVar(&passwordStdin, "password-stdin", false, "read password from stdin")
	return cmd
}

func userDeactivateCmd(dbPath *string) *cobra.Command {
	var email string

	cmd := &cobra.Command{
		Use:   "deactivate",
		Short: "Deactivate a user account",
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, db, err := openUserRepo(*dbPath)
			if err != nil {
				return err
			}
			defer db.Close()

			ctx := context.Background()
			if err := repo.DeactivateUser(ctx, email); err != nil {
				return err
			}
			// Revoke all outstanding refresh tokens so the deactivated account
			// cannot continue to use its existing session.
			if u, err := repo.GetByEmail(ctx, email); err == nil {
				tokenRepo := sqlite.NewTokenRepository(db)
				_ = tokenRepo.RevokeAllForUser(ctx, u.ID)
			}
			fmt.Printf("User %s deactivated\n", email)
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "user email address (required)")
	_ = cmd.MarkFlagRequired("email")
	return cmd
}

func userActivateCmd(dbPath *string) *cobra.Command {
	var email string

	cmd := &cobra.Command{
		Use:   "activate",
		Short: "Re-enable a user account",
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, db, err := openUserRepo(*dbPath)
			if err != nil {
				return err
			}
			defer db.Close()

			if err := repo.ActivateUser(context.Background(), email); err != nil {
				return err
			}
			fmt.Printf("User %s activated\n", email)
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "user email address (required)")
	_ = cmd.MarkFlagRequired("email")
	return cmd
}

func userChangePasswordCmd(dbPath *string) *cobra.Command {
	var email string
	var passwordStdin bool

	cmd := &cobra.Command{
		Use:   "change-password",
		Short: "Change a user's password",
		RunE: func(cmd *cobra.Command, args []string) error {
			password, err := readPassword(passwordStdin, "New password: ")
			if err != nil {
				return err
			}

			if !passwordStdin {
				confirm, err := readPassword(false, "Confirm password: ")
				if err != nil {
					return err
				}
				if password != confirm {
					return fmt.Errorf("passwords do not match")
				}
			}

			if err := app.ValidatePasswordStrength(password); err != nil {
				return err
			}

			hash, err := app.HashPassword(password)
			if err != nil {
				return fmt.Errorf("hash password: %w", err)
			}

			repo, db, err := openUserRepo(*dbPath)
			if err != nil {
				return err
			}
			defer db.Close()

			if err := repo.UpdatePassword(context.Background(), email, hash); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "audit: password updated for %s\n", email)
			fmt.Printf("Password updated for %s\n", email)
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "user email address (required)")
	_ = cmd.MarkFlagRequired("email")
	cmd.Flags().BoolVar(&passwordStdin, "password-stdin", false, "read password from stdin")
	return cmd
}

// readPassword reads a password interactively (masked) or from stdin.
func readPassword(fromStdin bool, prompt string) (string, error) {
	if fromStdin {
		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			return "", fmt.Errorf("no password provided on stdin")
		}
		return strings.TrimRight(scanner.Text(), "\r\n"), nil
	}

	fmt.Fprint(os.Stderr, prompt)
	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	return string(pw), nil
}
