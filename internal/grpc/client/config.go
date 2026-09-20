package client

import "time"

type Config struct {
	Address      string        `env:"ADDRESS" envDefault:":3200" json:"address"`
	Timeout      time.Duration `env:"TIMEOUT" envDefault:"5s" json:"timeout"`
	MsgSizeMaxMB int           `env:"MSG_SIZE_MAX_MB" json:"msgSizeMaxMB"`
}
