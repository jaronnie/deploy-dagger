package v1

import (
	"context"
	"fmt"
	"io"
	"os"

	"dagger.io/dagger"
	"github.com/gin-gonic/gin"
	"github.com/jaronnie/deploy-dagger/server/middlewares"
	"github.com/jaronnie/deploy-dagger/server/pkg/giturl"
	"github.com/spf13/viper"
)

type SSEWriter struct {
	data chan []byte
}

func (w *SSEWriter) Write(p []byte) (n int, err error) {
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
		home, _ := os.UserHomeDir()
		git := giturl.GenCloneGitRepoUrl(&giturl.GitConfig{
			Private:     viper.GetBool("git.private"),
			Type:        viper.GetString("git.type"),
			Protocol:    viper.GetString("git.protocol"),
			Url:         viper.GetString("git.url"),
			Group:       viper.GetString("git.group"),
			ProjectName: projectName,
			AccessToken: viper.GetString("git.accessToken"),
		})
		ctx := context.Background()

		writer := &SSEWriter{
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

			settings := client.Host().File(fmt.Sprintf("%s/.m2/settings.xml", home))
			daggerCache := client.CacheVolume("maven")
			// use a mvn:3.6.3 container
			// get version
			// execute
			_, _ = client.Container().From("maven:3.6.3-openjdk-8").
				WithExec([]string{"mvn", "--version"}).
				WithFile("/root/.m2/settings.xml", settings).
				WithDirectory("/src", project).
				WithMountedCache(fmt.Sprintf("%s/.m2/repository", home), daggerCache).
				WithWorkdir("/src").
				WithExec([]string{"sh", "-c", "mvn package -Dmaven.test.skip=true"}).
				File("./yx-admin/target/yx-admin.jar").
				Export(ctx, "./yx-admin/target/yx-admin.jar")
		}()

		c.Stream(func(w io.Writer) bool {
			msg := <-writer.data
			// Stream message to client from message channel
			c.SSEvent("message", string(msg)+"\n")
			return true
		})
	})

}
