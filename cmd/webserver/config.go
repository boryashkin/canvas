package main

type Environment struct {
	WebsocketPort int `env:"WEBSOCKET_PORT"` // the same as http
}
