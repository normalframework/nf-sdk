package proxyprobe

import (
	"encoding/base64"
	"fmt"
)

func basicAuthHeader(user, pass string) string {
	cred := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
	return fmt.Sprintf("Proxy-Authorization: Basic %s\r\n", cred)
}
