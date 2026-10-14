package web

import "os"

func defaultClaimant() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		return "web"
	}
	return "web@" + host
}
