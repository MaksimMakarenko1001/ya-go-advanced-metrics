package config

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/caarlos0/env/v6"

	"github.com/MaksimMakarenko1001/ya-go-advanced/internal/config/db"
	"github.com/MaksimMakarenko1001/ya-go-advanced/internal/grpc/server"
	"github.com/MaksimMakarenko1001/ya-go-advanced/internal/logger"
	auditFileService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/auditFileService/v0"
	auditRemoteService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/auditRemoteService/v0"
	decryptService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/decryptService/v0"
	dumpMetricService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/dumpMetricService/v0"
	hashService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/hashService/v0"
	subnetService "github.com/MaksimMakarenko1001/ya-go-advanced/internal/service/subnetService/v0"
	"github.com/MaksimMakarenko1001/ya-go-advanced/internal/worker/sworker"
)

type diConfig struct {
	HTTP               HTTPServerConfig          `envPrefix:"HTTP_" json:"http"`
	GRPC               server.Config             `envPrefix:"GRPC_" json:"grpc"`
	Logger             logger.Config             `envPrefix:"LOGGER_" json:"logger"`
	StoreInterval      time.Duration             `env:"STORE_INTERVAL" json:"storeInterval"`
	FileStoragePath    string                    `env:"FILE_STORAGE_PATH" json:"fileStoragePath"`
	Restore            bool                      `env:"RESTORE" json:"restore"`
	Database           db.Config                 `envPrefix:"DATABASE_" json:"database"`
	HashService        hashService.Config        `envPrefix:"HASH_SERVICE_" json:"hashService"`
	DecryptService     decryptService.Config     `envPrefix:"DECRYPT_SERVICE_" json:"decryptService"`
	SubnetService      subnetService.Config      `envPrefix:"SUBNET_SERVICE_" json:"subnetService"`
	DumpService        dumpMetricService.Config  `envPrefix:"DUMP_SERVICE_" json:"dumpService"`
	DumpSyncService    dumpMetricService.Config  `envPrefix:"DUMP_SYNC_SERVICE_" json:"dumpSyncService"`
	AuditFileService   auditFileService.Config   `envPrefix:"AUDIT_FILE_SERVICE_" json:"auditFileService"`
	AuditRemoteService auditRemoteService.Config `envPrefix:"AUDIT_REMOTE_SERVICE_" json:"auditRemoteService"`
	Worker             struct {
		AuditFile   sworker.Config `envPrefix:"AUDIT_FILE_" json:"auditFile"`
		AuditRemote sworker.Config `envPrefix:"AUDIT_REMOTE_" json:"auditRemote"`
	} `envPrefix:"WORKER_" json:"worker"`
	AuditFile   string `env:"AUDIT_FILE" json:"auditFile"`
	AuditRemote string `env:"AUDIT_URL" json:"auditRemote"`
	ConfigJSON  struct {
		Config string `env:"CONFIG" json:"config"`
	} `json:"configJSON"`
}

func (cfg *diConfig) loadConfig(envPrefix string) {
	cfg.loadDefaults(envPrefix)
	cfg.loadFromJSON(envPrefix)
	cfg.loadFromArg()
	cfg.loadFromEnv(envPrefix)
	cfg.loadFromEnvToPassTests() // meets tests required

	if cfg.StoreInterval > 0 {
		cfg.DumpSyncService.WriteDumpEnable = false
	}
	if cfg.FileStoragePath == "" {
		cfg.DumpService.ReadDumpEnable = false
		cfg.DumpService.WriteDumpEnable = false
	}
	if cfg.AuditFile == "" {
		cfg.AuditFileService.AuditEnabled = false
	}
	if cfg.AuditRemote == "" {
		cfg.AuditRemoteService.AuditEnabled = false
	}
	if cfg.DecryptService.CryptoKey == "" {
		cfg.DecryptService.DecryptEnabled = false
	}
	if cfg.SubnetService.TrustedSubnet == "" {
		cfg.SubnetService.ValidateEnabled = false
	}
}

func (cfg *diConfig) loadDefaults(envPrefix string) {
	if err := env.Parse(cfg, env.Options{Prefix: envPrefix}); err != nil {
		log.Printf("env defaults not ok, %s\n", err.Error())
	}
}

func (cfg *diConfig) loadFromJSON(envPrefix string) {
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
		Address       string `json:"address"`
		RemoteAddress string `json:"remote_address"`
		Restore       bool   `json:"restore"`
		StoreInterval string `json:"store_interval"`
		StoreFile     string `json:"store_file"`
		DatabaseDsn   string `json:"database_dsn"`
		CryptoKey     string `json:"crypto_key"`
		TrustedSubnet string `json:"trusted_subnet"`
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
		cfg.HTTP.Address = address
	}
	if address := config.RemoteAddress; address != "" {
		cfg.GRPC.Address = address
	}
	if restore := config.Restore; restore {
		cfg.Restore = restore
	}
	if storeFile := config.StoreFile; storeFile != "" {
		cfg.FileStoragePath = storeFile
	}
	if dsn := config.DatabaseDsn; dsn != "" {
		cfg.Database.DSN = dsn
	}
	if cryptoKey := config.CryptoKey; cryptoKey != "" {
		cfg.DecryptService.CryptoKey = cryptoKey
	}
	if trustedSubnet := config.TrustedSubnet; trustedSubnet != "" {
		cfg.SubnetService.TrustedSubnet = trustedSubnet
	}
	if storeInterval := config.StoreInterval; storeInterval != "" {
		if store, err := time.ParseDuration(storeInterval); err == nil && store > 0 {
			cfg.StoreInterval = store
		}
	}
}

func (cfg *diConfig) loadFromArg() {
	var config struct {
		Address         string
		RemoteAddress   string
		Store           int
		FileStoragePath string
		Restore         bool
		DSN             string
		Key             string
		AuditFile       string
		AuditRemote     string
		CryptoKey       string
		TrustedSubnet   string
	}

	flag.StringVar(&config.Address, "a", "", "server net address")
	flag.StringVar(&config.RemoteAddress, "remote-address", "", "remote server net address")
	flag.IntVar(&config.Store, "i", 0, "store interval in seconds")
	flag.StringVar(&config.FileStoragePath, "f", "", "dump file path")
	flag.BoolVar(&config.Restore, "r", false, "restore dump file on start")
	flag.StringVar(&config.DSN, "d", "", "data source name")
	flag.StringVar(&config.Key, "k", "", "hash key")
	flag.StringVar(&config.AuditFile, "audit-file", "", "audit file name")
	flag.StringVar(&config.AuditRemote, "audit-url", "", "audit full url")
	flag.StringVar(&config.CryptoKey, "crypto-key", "", "crypto key path")
	flag.StringVar(&config.TrustedSubnet, "t", "", "trusted subnet")

	flag.Parse()

	if address := config.Address; address != "" {
		cfg.HTTP.Address = address
	}
	if address := config.RemoteAddress; address != "" {
		cfg.GRPC.Address = address
	}
	if store := config.Store; store > 0 {
		cfg.StoreInterval = time.Second * time.Duration(store)
	}
	if fname := config.FileStoragePath; fname != "" {
		cfg.FileStoragePath = fname
	}
	if restore := config.Restore; restore {
		cfg.Restore = restore
	}
	if key := config.Key; key != "" {
		cfg.HashService.Key = key
	}
	if auditFile := config.AuditFile; auditFile != "" {
		cfg.AuditFile = auditFile
	}
	if auditRemote := config.AuditRemote; auditRemote != "" {
		cfg.AuditRemote = auditRemote
	}
	if cryptoKey := config.CryptoKey; cryptoKey != "" {
		cfg.DecryptService.CryptoKey = cryptoKey
	}
	if trustedSubnet := config.TrustedSubnet; trustedSubnet != "" {
		cfg.SubnetService.TrustedSubnet = trustedSubnet
	}
	if dsn := config.DSN; dsn != "" {
		cfg.Database.DSN = dsn
	}
}

func (cfg *diConfig) loadFromEnv(envPrefix string) {
	var config diConfig

	if err := env.Parse(&config, env.Options{Prefix: envPrefix /*, RequiredIfNoDef: true*/}); err != nil {
		log.Printf("env not ok, %s\n", err.Error())
		return
	}

	if address := config.HTTP.Address; address != "" {
		cfg.HTTP.Address = address
	}
	if address := config.GRPC.Address; address != "" {
		cfg.GRPC.Address = address
	}
	if store := config.StoreInterval; store.String() != "0s" {
		cfg.StoreInterval = store
	}
	if fname := config.FileStoragePath; fname != "" {
		cfg.FileStoragePath = fname
	}
	if restore := config.Restore; restore {
		cfg.Restore = restore
	}
	if auditFile := config.AuditFile; auditFile != "" {
		cfg.AuditFile = auditFile
	}
	if auditRemote := config.AuditRemote; auditRemote != "" {
		cfg.AuditRemote = auditRemote
	}
	if cryptoKey := config.DecryptService.CryptoKey; cryptoKey != "" {
		cfg.DecryptService.CryptoKey = cryptoKey
	}
	if trustedSubnet := config.SubnetService.TrustedSubnet; trustedSubnet != "" {
		cfg.SubnetService.TrustedSubnet = trustedSubnet
	}
	if dsn, err := config.Database.ToDSN(); err == nil {
		cfg.Database.DSN = dsn
	}
}

func (cfg *diConfig) loadFromEnvToPassTests() {
	if address := os.Getenv("ADDRESS"); address != "" {
		cfg.HTTP.Address = address
	}
	if fname := os.Getenv("FILE_STORAGE_PATH"); fname != "" {
		cfg.FileStoragePath = fname
	}
	if restore := os.Getenv("RESTORE"); restore != "" {
		cfg.Restore = restore == "true"
	}
	if dsn := os.Getenv("DATABASE_DSN"); dsn != "" {
		cfg.Database.DSN = dsn
	}
	if key := os.Getenv("KEY"); key != "" {
		cfg.HashService.Key = key
	}
	if auditFile := os.Getenv("AUDIT_FILE"); auditFile != "" {
		cfg.AuditFile = auditFile
	}
	if auditRemote := os.Getenv("AUDIT_URL"); auditRemote != "" {
		cfg.AuditRemote = auditRemote
	}
	if cryptoKey := os.Getenv("CRYPTO_KEY"); cryptoKey != "" {
		cfg.DecryptService.CryptoKey = cryptoKey
	}
	if trustedSubnet := os.Getenv("TRUSTED_SUBNET"); trustedSubnet != "" {
		cfg.SubnetService.TrustedSubnet = trustedSubnet
	}
	if storeInterval, ok := os.LookupEnv("STORE_INTERVAL"); ok {
		if store, err := strconv.Atoi(storeInterval); err == nil && store > 0 {
			cfg.StoreInterval = time.Second * time.Duration(store)
		}
	}
}

type HTTPServerConfig struct {
	Address string `env:"ADDRESS" json:"address"`
}
