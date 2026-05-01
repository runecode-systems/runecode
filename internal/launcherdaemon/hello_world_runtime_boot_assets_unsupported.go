//go:build !linux

package launcherdaemon

import "fmt"

func prepareHelloWorldBootAssets(string, string) (string, string, error) {
	return "", "", fmt.Errorf("prepare hello-world boot assets: linux host boot assets are required")
}
