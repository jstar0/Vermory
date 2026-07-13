package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"vermory/internal/authn"
	"vermory/internal/runtime"
	"vermory/internal/webchat"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
)

type serveOptions struct {
	DatabaseURL string
	Listen      string
	TLSCert     string
	TLSKey      string
	Provider    webChatProviderOptions
}

func (options serveOptions) Validate() error {
	if strings.TrimSpace(options.DatabaseURL) == "" {
		return fmt.Errorf("--database-url is required")
	}
	host, port, err := net.SplitHostPort(strings.TrimSpace(options.Listen))
	if err != nil || strings.TrimSpace(port) == "" {
		return fmt.Errorf("--listen must be a host:port address")
	}
	certSet := strings.TrimSpace(options.TLSCert) != ""
	keySet := strings.TrimSpace(options.TLSKey) != ""
	if certSet != keySet {
		return fmt.Errorf("--tls-cert and --tls-key must be provided together")
	}
	if !isLoopbackHost(host) && !certSet {
		return fmt.Errorf("non-loopback --listen requires --tls-cert and --tls-key")
	}
	return nil
}

func newServeCommand() *cobra.Command {
	options := serveOptions{}
	command := &cobra.Command{
		Use:   "serve",
		Short: "Run the authenticated multi-tenant Vermory API",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			if err := options.Validate(); err != nil {
				return err
			}
			llm, model, err := buildWebChatProvider(options.Provider)
			if err != nil {
				return err
			}
			store, err := runtime.OpenStoreWithOptions(command.Context(), options.DatabaseURL, runtime.StoreOptions{EnforceTenantContext: true})
			if err != nil {
				return err
			}
			defer store.Close()
			if err := store.ValidateRuntimeRole(command.Context()); err != nil {
				return err
			}
			authPool, err := pgxpool.New(command.Context(), options.DatabaseURL)
			if err != nil {
				return fmt.Errorf("open authentication database pool: %w", err)
			}
			defer authPool.Close()
			if err := authPool.Ping(command.Context()); err != nil {
				return fmt.Errorf("connect authentication database pool: %w", err)
			}
			server := &http.Server{
				Addr:              options.Listen,
				Handler:           webchat.NewAuthenticatedHandler(store, llm, model, authn.NewPostgresAuthenticator(authPool)),
				ReadHeaderTimeout: 5 * time.Second,
				ReadTimeout:       30 * time.Second,
				WriteTimeout:      5 * time.Minute,
				IdleTimeout:       60 * time.Second,
			}
			go shutdownWebChatServer(command.Context(), server)
			if strings.TrimSpace(options.TLSCert) != "" {
				err = server.ListenAndServeTLS(options.TLSCert, options.TLSKey)
			} else {
				err = server.ListenAndServe()
			}
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			return err
		},
	}
	command.Flags().StringVar(&options.DatabaseURL, "database-url", "", "restricted runtime PostgreSQL connection URL")
	command.Flags().StringVar(&options.Listen, "listen", "127.0.0.1:8788", "authenticated API listen address")
	command.Flags().StringVar(&options.TLSCert, "tls-cert", "", "TLS certificate path")
	command.Flags().StringVar(&options.TLSKey, "tls-key", "", "TLS private key path")
	command.Flags().StringVar(&options.Provider.Name, "provider", "mock", "provider: external, mock, grok-cli, openai-compatible, siliconflow, or duojie")
	command.Flags().StringVar(&options.Provider.Model, "model", "", "server-owned provider model")
	command.Flags().StringVar(&options.Provider.BaseURL, "base-url", "", "direct provider base URL")
	command.Flags().StringVar(&options.Provider.APIKeyEnv, "api-key-env", "", "environment variable containing provider API key")
	command.Flags().StringVar(&options.Provider.GrokCommand, "grok-command", "", "authenticated Grok CLI command")
	return command
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
