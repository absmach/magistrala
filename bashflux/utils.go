package cmd

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/fatih/color"
	"github.com/hokaccha/go-prettyjson"
)

const contentType = "application/json"

var Limit = 10
var Offset = 0

func SendRequest(req *http.Request, token string, e error) {
	req.Header.Set("Authorization", token)
	req.Header.Add("Content-Type", contentType)
	if e != nil {
		LogError(e)
		return
	}

	resp, err := httpClient.Do(req)
	FormatResLog(resp, err)
}

// FormatResLog - format http response
func FormatResLog(resp *http.Response, err error) {
	if err != nil {
		LogError(err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf(color.CyanString("%s %s\nContent-Length: %v\n\n"),
		resp.Proto, resp.Status, resp.ContentLength)

	if len(resp.Header.Get("Location")) != 0 {
		fmt.Printf(color.BlueString("Resource location: %s\n\n"),
			resp.Header.Get("Location"))
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		LogError(err)
		return
	}

	if len(body) != 0 {
		pj, err := prettyjson.Format([]byte(body))
		if err != nil {
			fmt.Printf("%s\n\n", color.BlueString(string(body)))
			return
		}
		fmt.Printf("%s\n\n", string(pj))
	}
}

func LogUsage(u string) {
	fmt.Printf(color.YellowString("Usage:  %s\n\n"), u)
}

func LogError(err error) {
	fmt.Printf("%s\n\n", color.RedString(err.Error()))
}
