package app

import "time"

func timestampToken() string {
	return time.Now().UTC().Format("20060102T150405Z")
}
