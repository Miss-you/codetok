package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	cursorapi "github.com/miss-you/codetok/cursor"
)

type cursorCommandService interface {
	Login(context.Context, string) (cursorapi.ValidationResult, error)
	Status(context.Context) (cursorapi.StatusResult, error)
	Activity(context.Context, string) (cursorapi.ActivityResult, error)
	Sync(context.Context) (cursorapi.SyncResult, error)
	Logout() error
}

func init() {
	rootCmd.AddCommand(newCursorCommand(newDefaultCursorCommandService()))
}

func newDefaultCursorCommandService() cursorCommandService {
	return cursorapi.NewService(
		cursorapi.NewStore(""),
		cursorapi.NewClient("", nil),
	)
}

func newCursorCommand(service cursorCommandService) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cursor",
		Short: "Cursor auth, sync, and local activity tools",
		Long: `Manage Cursor authentication, local dashboard sync, and local activity attribution.

Only 'login', 'status', and 'sync' may contact the remote Cursor API.
'activity' plus daily and session reporting remain local-file based.`,
	}

	cmd.AddCommand(
		newCursorLoginCommand(service),
		newCursorStatusCommand(service),
		newCursorActivityCommand(service),
		newCursorSyncCommand(service),
		newCursorLogoutCommand(service),
	)

	return cmd
}

func newCursorLoginCommand(service cursorCommandService) *cobra.Command {
	var token string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Validate and save a Cursor session token",
		Long: `Validate a WorkosCursorSessionToken with Cursor before saving it locally.

Provide the token with --token or pipe it on stdin.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			tokenValue, err := resolveCursorToken(cmd, token)
			if err != nil {
				return err
			}

			result, err := service.Login(cmd.Context(), tokenValue)
			if err != nil {
				return err
			}

			if result.MembershipType != "" {
				fmt.Printf("Cursor login successful (membership: %s)\n", result.MembershipType)
				return nil
			}
			fmt.Println("Cursor login successful")
			return nil
		},
	}

	cmd.Flags().StringVar(&token, "token", "", "Cursor WorkosCursorSessionToken")
	return cmd
}

func newCursorStatusCommand(service cursorCommandService) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check saved Cursor credentials and remote validity",
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := service.Status(cmd.Context())
			if err != nil {
				return err
			}

			if !status.HasCredentials {
				fmt.Println("Cursor is not logged in")
				return nil
			}

			if status.RemoteValid {
				if status.MembershipType != "" {
					fmt.Printf("Cursor credentials saved and valid (membership: %s)\n", status.MembershipType)
					return nil
				}
				fmt.Println("Cursor credentials saved and valid")
				return nil
			}

			if status.Message != "" {
				fmt.Printf("Cursor credentials are saved locally but remote validation failed: %s\n", status.Message)
				return nil
			}
			fmt.Println("Cursor credentials are saved locally but remote validation failed")
			return nil
		},
	}
}

func newCursorActivityCommand(service cursorCommandService) *cobra.Command {
	var (
		jsonOutput bool
		dbPath     string
	)

	cmd := &cobra.Command{
		Use:   "activity",
		Short: "Show Cursor activity attribution from the local tracking database",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := service.Activity(cmd.Context(), dbPath)
			if err != nil {
				return err
			}

			if jsonOutput {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			printCursorActivity(result)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output Cursor activity attribution as JSON")
	cmd.Flags().StringVar(&dbPath, "db-path", "", "Override Cursor tracking database path")
	return cmd
}

func newCursorSyncCommand(service cursorCommandService) *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Fetch Cursor dashboard CSV into the local cache",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := service.Sync(cmd.Context())
			if err != nil {
				return err
			}

			if result.Bytes > 0 {
				fmt.Printf("Cursor sync complete: %s (%d bytes)\n", result.Path, result.Bytes)
				return nil
			}
			fmt.Printf("Cursor sync complete: %s\n", result.Path)
			return nil
		},
	}
}

func newCursorLogoutCommand(service cursorCommandService) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove saved Cursor credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := service.Logout(); err != nil {
				return err
			}
			fmt.Println("Cursor logged out")
			return nil
		},
	}
}

func resolveCursorToken(cmd *cobra.Command, flagValue string) (string, error) {
	if token := strings.TrimSpace(flagValue); token != "" {
		return token, nil
	}

	input := cmd.InOrStdin()
	if file, ok := input.(*os.File); ok {
		info, err := file.Stat()
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeCharDevice != 0 {
			return "", fmt.Errorf("provide a Cursor session token with --token or via stdin")
		}
	}

	data, err := io.ReadAll(input)
	if err != nil {
		return "", err
	}
	if token := strings.TrimSpace(string(data)); token != "" {
		return token, nil
	}

	return "", fmt.Errorf("provide a Cursor session token with --token or via stdin")
}

func printCursorActivity(result cursorapi.ActivityResult) {
	fmt.Println("Cursor Activity Attribution")
	if result.DBPath != "" {
		fmt.Printf("Database: %s\n", result.DBPath)
	}

	if !result.HasData {
		fmt.Println("No Cursor activity attribution data found.")
		return
	}

	fmt.Printf("Scored commits: %d\n\n", result.ScoredCommits)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Source\tLines Added\tLines Deleted")
	fmt.Fprintf(w, "composer\t%d\t%d\n", result.Composer.LinesAdded, result.Composer.LinesDeleted)
	fmt.Fprintf(w, "tab\t%d\t%d\n", result.Tab.LinesAdded, result.Tab.LinesDeleted)
	_ = w.Flush()
}
