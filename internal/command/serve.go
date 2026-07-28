package command

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KaribuLab/linux-mcp/internal/handler"
	"github.com/KaribuLab/linux-mcp/internal/issuer"
	"github.com/KaribuLab/linux-mcp/internal/token"
	"github.com/spf13/cobra"
)

const (
	defaultAddr        = "127.0.0.1:5000"
	defaultSocketGroup = "mcp-admin"
	defaultMaxTTL      = 24 * time.Hour
	shutdownTimeout    = 5 * time.Second
)

type serveOptions struct {
	addr        string
	socketPath  string
	socketGroup string
	maxTTL      time.Duration
	origins     []string
}

func newServeCmd() *cobra.Command {
	var opts serveOptions
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the MCP server",
		Long: "serve exposes the MCP endpoint over Streamable HTTP and, on a unix socket,\n" +
			"the channel used by 'auth' to obtain tokens.\n\n" +
			"The token signing key is generated in memory at startup, so restarting the\n" +
			"server invalidates every token issued before the restart.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			socket, err := cmd.Flags().GetString("socket")
			if err != nil {
				return err
			}
			opts.socketPath = socket
			return runServe(opts)
		},
	}
	cmd.Flags().StringVar(&opts.addr, "addr", defaultAddr, "address the MCP endpoint listens on")
	cmd.Flags().StringVar(&opts.socketGroup, "socket-group", defaultSocketGroup,
		"group allowed to request tokens; empty leaves the socket owner-only")
	cmd.Flags().DurationVar(&opts.maxTTL, "max-ttl", defaultMaxTTL, "maximum lifetime granted to an issued token")
	cmd.Flags().StringSliceVar(&opts.origins, "cors", nil,
		"comma-separated browser origins allowed to call the MCP endpoint")
	return cmd
}

func runServe(opts serveOptions) error {
	// Origins are validated before anything is opened, so a malformed value is
	// a startup error and never a rule that silently never matches.
	origins, err := handler.ParseOrigins(opts.origins)
	if err != nil {
		return err
	}
	signer, err := token.NewSigner(resourceURL(opts.addr))
	if err != nil {
		return err
	}
	mcpHandler, err := handler.NewHandler(handler.Config{
		Verifier: signer,
		Addr:     opts.addr,
		Origins:  origins,
	})
	if err != nil {
		return err
	}

	listener, err := issuer.Listen(opts.socketPath, opts.socketGroup)
	if err != nil {
		return err
	}
	// Closing the listener also unlinks the socket file.
	defer listener.Close()

	issuance := &issuer.Server{Signer: signer, MaxTTL: opts.maxTTL}
	go func() {
		if err := issuance.Serve(listener); err != nil {
			slog.Error("issuance socket stopped", "error", err)
		}
	}()
	slog.Info("token issuance listening", "socket", opts.socketPath, "group", opts.socketGroup)

	server := &http.Server{Addr: opts.addr, Handler: mcpHandler}
	slog.Info("MCP endpoint listening", "addr", opts.addr)
	return listenAndShutdown(server)
}

func listenAndShutdown(server *http.Server) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	failed := make(chan error, 1)
	go func() { failed <- server.ListenAndServe() }()

	select {
	case err := <-failed:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		slog.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}

// resourceURL is the audience tokens are bound to, and the value clients reach.
func resourceURL(addr string) string {
	return "http://" + addr
}
