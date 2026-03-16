package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	cursorapi "github.com/miss-you/codetok/cursor"
)

type cursorCommandService interface {
	Login(context.Context, string) (cursorapi.ValidationResult, error)
	Status(context.Context) (cursorapi.StatusResult, error)
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
		Short: "Explicit Cursor authentication and dashboard sync",
		Long: `Manage explicit Cursor authentication and local dashboard sync.

These commands are the only codetok flows that may contact the remote Cursor API.
Daily and session reporting remain local-file based.`,
	}

	cmd.AddCommand(
		newCursorLoginCommand(service),
		newCursorStatusCommand(service),
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
