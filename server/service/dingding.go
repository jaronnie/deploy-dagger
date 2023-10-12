package service

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Robot struct {
	AccessToken string
}

func Send(c *gin.Context) {
	robot := Robot{AccessToken: c.Query("access_token")}
	message := &Message{}
	err := c.BindJSON(message)
	if err != nil {
		c.String(500, "bind Message struct error: "+err.Error())
		return
	}
	res, _ := robot.send(message)
	c.String(200, string(res))
}

type Message struct {
	Msgtype  string   `json:"msgtype"`
	Text     Text     `json:"text"`
	Markdown Markdown `json:"markdown"`
}

type Text struct {
	Content string `json:"content"`
}

type Markdown struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

func (r *Robot) send(message *Message) ([]byte, error) {
	b, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Post("https://oapi.dingtalk.com/robot/send?access_token="+r.AccessToken, "application/json", bytes.NewReader(b))

	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	return body, nil
}
