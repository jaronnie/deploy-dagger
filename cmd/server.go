/*
Copyright © 2023 jaronnie <jaron@jaronnie.com>

*/

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jaronnie/deploy-dagger/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "deploy-dagger server",
	Long:  `deploy-dagger server`,
	RunE: func(cmd *cobra.Command, args []string) error {
		r := gin.Default()
		server.Cors(r)
		server.Router(r)

		port := viper.GetString("port")
		if port == "" {
			port = "8080"
		}
		base := fmt.Sprintf("%s:%s", "0.0.0.0", port)

		go func() {
			if err := r.Run(base); err != nil {
				panic(err)
			}

		}()

		// Wait for interrupt signal to gracefully shutdown the server with
		quit := make(chan os.Signal, 1)

		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		select {
		case <-ctx.Done():
			return nil
		default:
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
}
