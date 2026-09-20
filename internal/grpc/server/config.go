package server

import "time"

type Config struct {
	Address               string        `env:"ADDRESS" envDefault:":3200" json:"address"`
	MaxConnectionAge      time.Duration `env:"MAX_CONNECTION_AGE" default:"10m" json:"maxConnectionAge"`
	MaxConnectionAgeGrace time.Duration `env:"MAX_CONNECTION_AGE_GRACE" default:"10s" json:"maxConnectionAgeGrace"`
}
