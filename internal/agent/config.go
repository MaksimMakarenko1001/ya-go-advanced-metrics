package agent

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/MaksimMakarenko1001/ya-go-advanced/internal/grpc/client"
	"github.com/caarlos0/env/v6"
)

type Config struct {
	Address        string        `env:"ADDRESS" json:"address"`
	Timeout        time.Duration `env:"TIMEOUT" envDefault:"10s" json:"timeout"`
	BatchSize      int           `env:"BATCH_SIZE" envDefault:"3" json:"batchSize"`
	MaxRetries     uint16        `env:"MAX_RETRIES" envDefault:"3" json:"maxRetries"`
	Key            string        `env:"KEY" json:"key"`
	PollInterval   time.Duration `env:"POOL_INTERVAL" json:"pollInterval"`
	ReportInterval time.Duration `env:"REPORT_INTERVAL" json:"reportInterval"`
	RateLimit      int           `env:"RATE_LIMIT" envDefault:"3" json:"rateLimit"`
	CryptoKey      string        `env:"CRYPTO_KEY" json:"cryptoKey"`
	ConfigJSON     struct {
		Config string `env:"CONFIG" json:"config"`
	} `json:"configJSON"`
	RealIP string        `env:"REAL_IP" envDefault:"127.0.0.1" json:"realIP"`
	GRPC   client.Config `envPrefix:"GRPC_" json:"grpc"`
}

func (cfg *Config) LoadConfig(envPrefix string) {
	cfg.loadDefaults(envPrefix)
	cfg.loadFromJSON(envPrefix)
	cfg.loadFromArg()
	cfg.loadFromEnv(envPrefix)
	cfg.loadFromEnvPassTests() // meets tests required
}

func (cfg *Config) loadDefaults(envPrefix string) {
	if err := env.Parse(cfg, env.Options{Prefix: envPrefix}); err != nil {
		log.Printf("env defaults not ok, %s\n", err.Error())
	}
}

func (cfg *Config) loadFromJSON(envPrefix string) {
	flag.StringVar(&cfg.ConfigJSON.Config, "c", "", "config path")
	flag.StringVar(&cfg.ConfigJSON.Config, "config", "", "config path")

	flag.Parse()

	if err := env.Parse(&cfg.ConfigJSON, env.Options{Prefix: envPrefix}); err != nil {
		log.Printf("env config json path not ok, %s\n", err.Error())
	}

	if config := os.Getenv("CONFIG"); config != "" {
		cfg.ConfigJSON.Config = config
	}

	if cfg.ConfigJSON.Config == "" {
		return
	}

	var config struct {
		Address        string `json:"address"`
		RemoteAddress  string `json:"remote_address"`
		ReportInterval string `json:"report_interval"`
		PollInterval   string `json:"poll_interval"`
		CryptoKey      string `json:"crypto_key"`
	}

	data, err := os.ReadFile(cfg.ConfigJSON.Config)
	if err != nil {
		log.Printf("json config not ok, %s\n", err.Error())
		return
	}

	if err := json.Unmarshal(data, &config); err != nil {
		log.Printf("json config not ok, %s\n", err.Error())
		return
	}

	if address := config.Address; address != "" {
		cfg.Address = address
	}
	if address := config.RemoteAddress; address != "" {
		cfg.GRPC.Address = address
	}
	if reportInterval := config.ReportInterval; reportInterval != "" {
		if report, err := time.ParseDuration(reportInterval); err == nil {
			cfg.ReportInterval = report
		}
	}
	if pollInterval := config.PollInterval; pollInterval != "" {
		if pool, err := time.ParseDuration(pollInterval); err == nil {
			cfg.PollInterval = pool
		}
	}
	if cryptoKey := config.CryptoKey; cryptoKey != "" {
		cfg.CryptoKey = cryptoKey
	}
}

func (cfg *Config) loadFromArg() {
	var config struct {
		Address       string
		RemoteAddress string
		Pool          int
		Report        int
		Key           string
		RateLimit     int
		CryptoKey     string
	}

	flag.StringVar(&config.Address, "a", "", "agent net address")
	flag.StringVar(&config.RemoteAddress, "remote-address", "", "remote server net address")
	flag.IntVar(&config.Pool, "p", 0, "pool interval in seconds")
	flag.IntVar(&config.Report, "r", 0, "report interval in seconds")
	flag.StringVar(&config.Key, "k", "", "hash key")
	flag.IntVar(&config.RateLimit, "l", 0, "num threads work concurrently")
	flag.StringVar(&config.CryptoKey, "crypto-key", "", "crypto key path")

	flag.Parse()

	if address := config.Address; address != "" {
		cfg.Address = address
	}
	if address := config.RemoteAddress; address != "" {
		cfg.GRPC.Address = address
	}
	if pool := config.Pool; pool > 0 {
		cfg.PollInterval = time.Second * time.Duration(pool)
	}
	if report := config.Report; report > 0 {
		cfg.ReportInterval = time.Second * time.Duration(report)
	}
	if key := config.Key; key != "" {
		cfg.Key = key
	}
	if rateLimit := config.RateLimit; rateLimit > 1 {
		cfg.RateLimit = rateLimit
	}
	if cryptoKey := config.CryptoKey; cryptoKey != "" {
		cfg.CryptoKey = cryptoKey
	}
}

func (cfg *Config) loadFromEnv(envPrefix string) {
	var config Config

	if err := env.Parse(&config, env.Options{Prefix: envPrefix}); err != nil {
		log.Printf("env not ok, %s\n", err.Error())
		return
	}

	if address := config.Address; address != "" {
		cfg.Address = address
	}
	if address := config.GRPC.Address; address != "" {
		cfg.GRPC.Address = address
	}
	if pool := config.PollInterval; pool.String() != "0s" {
		cfg.PollInterval = pool
	}
	if report := config.ReportInterval; report.String() != "0s" {
		cfg.ReportInterval = report
	}
	if rateLimit := config.RateLimit; rateLimit > 1 {
		cfg.RateLimit = rateLimit
	}
	if cryptoKey := config.CryptoKey; cryptoKey != "" {
		cfg.CryptoKey = cryptoKey
	}
}

func (cfg *Config) loadFromEnvPassTests() {
	if address := os.Getenv("ADDRESS"); address != "" {
		cfg.Address = address
	}
	if key := os.Getenv("KEY"); key != "" {
		cfg.Key = key
	}
	if cryptoKey := os.Getenv("CRYPTO_KEY"); cryptoKey != "" {
		cfg.CryptoKey = cryptoKey
	}

	if poolInterval, ok := os.LookupEnv("POOL_INTERVAL"); ok {
		if pool, err := strconv.Atoi(poolInterval); err == nil && pool > 0 {
			cfg.PollInterval = time.Second * time.Duration(pool)
		}
	}
	if reportInterval, ok := os.LookupEnv("REPORT_INTERVAL"); ok {
		if report, err := strconv.Atoi(reportInterval); err == nil && report > 0 {
			cfg.ReportInterval = time.Second * time.Duration(report)
		}
	}
	if rateLimit, ok := os.LookupEnv("RATE_LIMIT"); ok {
		if rate, err := strconv.Atoi(rateLimit); err == nil && rate > 1 {
			cfg.RateLimit = rate
		}
	}
}
