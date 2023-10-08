package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"dagger.io/dagger"
	"github.com/gin-gonic/gin"
	"github.com/jaronnie/deploy-dagger/server/pkg/dcompose"
	"github.com/jaronnie/deploy-dagger/server/pkg/giturl"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type ResponseWriter struct {
	data chan []byte
}

func (w *ResponseWriter) Write(p []byte) (n int, err error) {
	w.data <- p
	return len(p), nil
}

type ComposeService struct {
	Name     string `json:"name"`
	Mapping  string `json:"mapping"`
	CheckUrl string `json:"checkUrl"`
}

func mappingComposeServiceWithProjectName(projectName string) (*ComposeService, error) {
	services := viper.Get("compose.services")

	marshalServices, err := json.Marshal(services)
	if err != nil {
		return nil, err
	}

	var css []*ComposeService
	err = json.Unmarshal(marshalServices, &css)
	if err != nil {
		return nil, err
	}

	for _, v := range css {
		if v.Name == projectName {
			return v, nil
		}
	}

	return nil, errors.Errorf("not found mapping")
}

func Deploy(c *gin.Context) {
	projectName := c.Query("project")
	branch := c.DefaultQuery("branch", "dev")
	target := c.Query("target")

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

	writer := &ResponseWriter{
		data: make(chan []byte),
	}

	done := make(chan int, 1)

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
		exportFile := filepath.Join(target[:len(target)-len(filepath.Ext(target))], "target", target)
		_, _ = client.Container().From("maven:3.6.3-openjdk-8").
			WithExec([]string{"mvn", "--version"}).
			WithFile("/root/.m2/settings.xml", settings).
			WithDirectory("/src", project).
			WithMountedCache(fmt.Sprintf("%s/.m2/repository", home), daggerCache).
			WithWorkdir("/src").
			WithExec([]string{"sh", "-c", "mvn package -Dmaven.test.skip=true"}).
			File(exportFile).
			Export(ctx, target)

		// 执行 docker-compose
		engine := dcompose.DockerComposeEngine{
			YmlPath: viper.GetString("compose.yaml"),
		}
		cs, err := mappingComposeServiceWithProjectName(projectName)
		if err != nil {
			writer.Write([]byte(err.Error()))
			done <- 1
			return
		}

		robot := Robot{AccessToken: "b226184a8c69d20d0b2c5b232884c15e59e43c3fb1a59d568468a16b206b4c0f"}
		robot.send(&Message{
			Msgtype: "text",
			Text: Text{
				Content: fmt.Sprintf("自动化通知: 开始更新后端 [%s] 服务, 两分钟后重试...【调试信息请忽略】", projectName),
			},
		})

		s, err := engine.RunDockerComposeCommand("stop", []string{cs.Mapping})
		if err != nil {
			writer.Write([]byte(err.Error()))
			done <- 1
			return
		}
		writer.Write([]byte(s))

		err = os.Rename(target, filepath.Join(filepath.Dir(viper.GetString("compose.yaml")), cs.Mapping, "app", target))
		if err != nil {
			writer.Write([]byte(err.Error()))
			done <- 1
			return
		}

		s, err = engine.RunDockerComposeCommand("up", []string{"-d", "--build", cs.Mapping})
		if err != nil {
			writer.Write([]byte(err.Error()))
			done <- 1
			return
		}
		writer.Write([]byte(s))

		// 开始检测服务是否正常
		timeout, cancel := context.WithTimeout(context.Background(), time.Duration(100)*time.Second)
		defer cancel()
		err = checkOK(timeout, cs.CheckUrl)
		if err != nil {
			writer.Write([]byte(err.Error()))
			robot.send(&Message{
				Msgtype: "text",
				Text: Text{
					Content: fmt.Sprintf("自动化通知: 后端 [%s] 服务异常, 请检查【调试信息请忽略】", projectName),
				},
			})
			done <- 1
			return
		}

		// 部署完毕
		done <- 1
		robot.send(&Message{
			Msgtype: "text",
			Text: Text{
				Content: fmt.Sprintf("自动化通知: 后端 [%s] 部署完成, 可正常使用【调试信息请忽略】", projectName),
			},
		})
	}()

	c.Stream(func(w io.Writer) bool {
		select {
		case done := <-done:
			if done == 1 {
				return false
			}
		case msg := <-writer.data:
			c.Writer.Write([]byte(msg))
			return true
		}

		return true
	})
}

func checkOK(ctx context.Context, checkurl string) error {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			_, err := HTTPDoGetWithCtx(ctx, checkurl)
			if err != nil {
				time.Sleep(time.Second * 10)
				continue
			}

			return nil
		case <-ctx.Done():
			return errors.New("fail to check ok, because timeout")
		}
	}

}

func HTTPDoGetWithCtx(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "new request error")
	}
	req = req.WithContext(ctx)
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "do http get")
	} else if response == nil {
		return nil, errors.New("http response is nil")
	}
	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Wrap(err, "read response body")
	}
	if response.StatusCode != http.StatusOK {
		return nil, errors.Errorf("fail to get, because http response code [%d], data [%s]", response.StatusCode, string(data))
	}
	return data, nil
}
