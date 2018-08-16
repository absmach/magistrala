package cli

import (
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

const contentTypeSenml = "application/senml+json"

var cmdMessages = []cobra.Command{
	cobra.Command{
		Use:   "send",
		Short: "send <channel_id> <JSON_string> <client_token>",
		Long:  `Sends message on the channel`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				LogUsage(cmd.Short)
				return
			}
			SendMsg(args[0], args[1], args[2])
		},
	},
}

func NewMessagesCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "msg",
		Short: "Send or retrieve messages",
		Long:  `Send or retrieve messages: control message flow on the channel`,
	}

	for i, _ := range cmdMessages {
		cmd.AddCommand(&cmdMessages[i])
	}

	return &cmd
}

// SendMsg - publishes SenML message on the channel
func SendMsg(id, msg, token string) {
	url := serverAddr + "/http/channels/" + id + "/messages"
	req, err := http.NewRequest("POST", url, strings.NewReader(msg))
	LogError(err)

	req.Header.Set("Authorization", token)
	req.Header.Add("Content-Type", contentTypeSenml)

	resp, err := httpClient.Do(req)
	FormatResLog(resp, err)
}
