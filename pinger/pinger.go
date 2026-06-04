package pinger

import (
	"fmt"
	"net"
	"time"
)

type Result struct {
	Server  string `json:"server"`
	Success bool   `json:"success"`
	Latency string `json:"latency"`
	Error   string `json:"error,omitempty"`
}

func Ping(server string) Result {
	start := time.Now()

	conn, err := net.DialTimeout("tcp", net.JoinHostPort(server, "53"), 5*time.Second)
	if err != nil {
		conn2, err2 := net.DialTimeout("udp", net.JoinHostPort(server, "53"), 5*time.Second)
		if err2 != nil {
			return Result{
				Server:  server,
				Success: false,
				Latency: "N/A",
				Error:   fmt.Sprintf("%v", err2),
			}
		}
		defer conn2.Close()
		latency := time.Since(start)
		return Result{
			Server:  server,
			Success: true,
			Latency: fmt.Sprintf("%dms", latency.Milliseconds()),
		}
	}
	defer conn.Close()
	latency := time.Since(start)
	return Result{
		Server:  server,
		Success: true,
		Latency: fmt.Sprintf("%dms", latency.Milliseconds()),
	}
}
