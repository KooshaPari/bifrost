package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	bifrost "github.com/maximhq/bifrost/core"
	"github.com/maximhq/bifrost/core/schemas"
	"github.com/spf13/cobra"

	"github.com/kooshapari/bifrost-extensions/account"
	"github.com/kooshapari/bifrost-extensions/plugins/intelligentrouter"
	"github.com/kooshapari/bifrost-extensions/plugins/learning"
	"github.com/kooshapari/bifrost-extensions/plugins/smartfallback"
)

var (
	port     int
	host     string
	plugins  []string
	logLevel string
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the Bifrost server",
	Long:  `Start the Bifrost LLM gateway server with configured plugins and providers.`,
	RunE:  runServer,
}

func init() {
	serverCmd.Flags().IntVarP(&port, "port", "p", 8080, "Server port")
	serverCmd.Flags().StringVarP(&host, "host", "h", "0.0.0.0", "Server host")
	serverCmd.Flags().StringSliceVarP(&plugins, "plugins", "P", []string{"router", "learning", "fallback"}, "Plugins to load")
	serverCmd.Flags().StringVarP(&logLevel, "log-level", "l", "info", "Log level (debug, info, warn, error)")
}

func runServer(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutting down...")
		cancel()
	}()

	// Create enhanced account
	acct := account.NewEnhancedAccount(nil)
	setupProviders(acct)

	// Create plugins
	var pluginList []schemas.Plugin
	for _, p := range plugins {
		switch p {
		case "router":
			pluginList = append(pluginList, intelligentrouter.New(intelligentrouter.DefaultConfig()))
		case "learning":
			pluginList = append(pluginList, learning.New(learning.DefaultConfig()))
		case "fallback":
			pluginList = append(pluginList, smartfallback.New(smartfallback.DefaultConfig()))
		}
	}

	// Initialize Bifrost
	bf, err := bifrost.Init(ctx, schemas.BifrostConfig{
		Account:         acct,
		Plugins:         pluginList,
		Logger:          bifrost.NewDefaultLogger(schemas.LogLevelInfo),
		InitialPoolSize: 100,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize Bifrost: %w", err)
	}

	fmt.Printf("✓ Bifrost server started on %s:%d\n", host, port)
	fmt.Printf("✓ Plugins loaded: %v\n", plugins)

	// Wait for shutdown
	<-ctx.Done()
	bf.Shutdown()
	fmt.Println("✓ Shutdown complete")

	return nil
}

func setupProviders(acct *account.EnhancedAccount) {
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		acct.SetKeys(schemas.OpenAI, []schemas.Key{{
			ID:     "openai-default",
			Value:  key,
			Weight: 1.0,
		}})
		acct.SetConfig(schemas.OpenAI, &schemas.ProviderConfig{
			NetworkConfig: schemas.NetworkConfig{
				DefaultRequestTimeoutInSeconds: 60,
				MaxRetries:                     3,
				RetryBackoffInitial:            500 * time.Millisecond,
				RetryBackoffMax:                5 * time.Second,
			},
		})
	}
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		acct.SetKeys(schemas.Anthropic, []schemas.Key{{
			ID:     "anthropic-default",
			Value:  key,
			Weight: 1.0,
		}})
		acct.SetConfig(schemas.Anthropic, &schemas.ProviderConfig{
			NetworkConfig: schemas.NetworkConfig{
				DefaultRequestTimeoutInSeconds: 60,
				MaxRetries:                     3,
				RetryBackoffInitial:            500 * time.Millisecond,
				RetryBackoffMax:                5 * time.Second,
			},
		})
	}
}
