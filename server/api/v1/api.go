package v1

import (
	"context"
	"fmt"
	"io"

	"dagger.io/dagger"
	"github.com/gin-gonic/gin"
	"github.com/jaronnie/deploy-dagger/server/middlewares"
	"github.com/spf13/viper"
)

type MyWriter struct {
	data chan []byte
}

func (w *MyWriter) Write(p []byte) (n int, err error) {
	w.data <- p
	return len(p), nil
}

func ApiRouter(rg *gin.RouterGroup) {
	rg.GET("/health", func(ctx *gin.Context) {
		ctx.String(200, "success")
	})

	rg.GET("/deploy", middlewares.HeadersMiddleware(), func(c *gin.Context) {

		projectName := c.Query("project")
		branch := c.DefaultQuery("branch", "dev")
		git := fmt.Sprintf("%s://oauth2:%s@%s/%s/%s", viper.GetString("git.protocol"), viper.GetString("git.accessToken"), viper.GetString("git.url"), viper.GetString("git.group"), projectName)
		ctx := context.Background()

		writer := &MyWriter{
			data: make(chan []byte),
		}

		go func() {
			client, err := dagger.Connect(ctx, dagger.WithLogOutput(writer))
			if err != nil {
				panic(err)
			}
			defer client.Close()

			project := client.
				Git(git).
				Branch(branch).
				Tree()

			// TODO 发送钉钉消息

			settings := client.Host().File("/Users/jaronnie/.m2/settings.xml")
			daggerCache := client.CacheVolume("maven")
			// use a mvn:3.6.3 container
			// get version
			// execute
			_, _ = client.Container().From("maven:3.6.3-openjdk-8").
				WithExec([]string{"mvn", "--version"}).
				WithFile("/root/.m2/settings.xml", settings).
				WithDirectory("/src", project).
				WithMountedCache("/Users/jaronnie/.m2/repository", daggerCache).
				WithWorkdir("/src").
				WithExec([]string{"sh", "-c", "mvn package -Dmaven.test.skip=true"}).
				File("./yx-admin/target/yx-admin.jar").
				Export(ctx, "./yx-admin/target/yx-admin.jar")
		}()

		c.Stream(func(w io.Writer) bool {
			msg := <-writer.data

			if string(msg) == "deploy DONE" {
				return false
			}
			// Stream message to client from message channel
			c.SSEvent("message", string(msg))
			return true
		})
	})

}
