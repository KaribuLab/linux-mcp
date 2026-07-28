package command

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"time"

	"github.com/KaribuLab/linux-mcp/internal/issuer"
	"github.com/spf13/cobra"
)

const (
	defaultTTL          = 8 * time.Hour
	maxResponseBytes    = 64 << 10
	authExchangeTimeout = 5 * time.Second
)

func newAuthCmd() *cobra.Command {
	var ttl time.Duration
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Obtain an access token from a running server",
		Long: "auth asks a running linux-mcp server for a bearer token.\n" +
			"The token is written to stdout; everything else goes to stderr, so it can be\n" +
			"captured with TOKEN=$(linux-mcp auth).\n\n" +
			"The token is issued for the user running this command, as reported by the\n" +
			"kernel. There is no flag to request a token for someone else.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			socket, err := cmd.Flags().GetString("socket")
			if err != nil {
				return err
			}
			return runAuth(cmd, socket, ttl)
		},
	}
	cmd.Flags().DurationVar(&ttl, "ttl", defaultTTL, "requested token lifetime; the server caps it")
	return cmd
}

func runAuth(cmd *cobra.Command, socket string, ttl time.Duration) error {
	conn, err := net.DialTimeout("unix", socket, authExchangeTimeout)
	if err != nil {
		return dialError(socket, err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(authExchangeTimeout))

	if err := json.NewEncoder(conn).Encode(issuer.Request{TTL: ttl.String()}); err != nil {
		return fmt.Errorf("send token request: %w", err)
	}

	var resp issuer.Response
	if err := json.NewDecoder(io.LimitReader(conn, maxResponseBytes)).Decode(&resp); err != nil {
		return fmt.Errorf("read token response: %w", err)
	}
	if resp.Error != "" {
		return errors.New(resp.Error)
	}

	if resp.Capped {
		fmt.Fprintf(cmd.ErrOrStderr(),
			"requested lifetime exceeds the server maximum; token capped\n")
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "subject: %s\nexpires: %s\n",
		resp.Subject, resp.ExpiresAt.Local().Format(time.RFC3339))
	fmt.Fprintln(cmd.OutOrStdout(), resp.Token)
	return nil
}

// dialError turns the two failures operators actually hit into actionable text.
func dialError(socket string, err error) error {
	switch {
	case errors.Is(err, fs.ErrPermission):
		return fmt.Errorf("permission denied on %s: you must belong to the group that owns the socket, "+
			"and group changes only apply to new login sessions", socket)
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("no socket at %s: the server may not be running, or it uses a different --socket", socket)
	default:
		return fmt.Errorf("connect to %s: %w", socket, err)
	}
}
