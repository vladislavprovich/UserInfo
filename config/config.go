package config

import (
	"flag"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/vladislavprovich/UserInfo/internal/storage"
	"os"
	"time"
)

type Config struct {
	Logger  LoggerConfig
	Logging LoggingConfig
	Tracing TracingConfig
	Mongo   storage.ConfigMongo
}

type LoggerConfig struct {
	Level string `env:"LOG_LEVEL" envDefault:"info"`
}

type LoggingConfig struct {
	LevelLoki string `env:"LOG_LEVEL_LOKI" envDefault:"info"`
	Format    string `env:"LOG_FORMAT" envDefault:"text"`
	LokiURL   string `env:"LOG_LOKI_URL" envDefault:"http://loki:30000"`
	LogDir    string `env:"LOG_DIR" envDefault:"./logs"`
}

type TracingConfig struct {
	TempoURL  string `env:"TRACING_TEMPO_URL" envDefault:"http://tempo:30000"`
	NameSpase string `env:"TRACING_NAME_SPASE" envDefault:"qwerty"`
}

type GRPCConfig struct {
	PortGRPC    string        `env:"GRPC_PORT" envDefault:"50051"`
	TimeoutGRPC time.Duration `env:"GRPC_TIMEOUT" envDefault:"10s"`
}

func MustLoad() *Config {
	configPath := fetchConfigPath()
	if configPath == "" {
		panic("config path is empty")
	}

	return MustLoadPath(configPath)
}

func MustLoadPath(configPath string) *Config {
	// check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic("config file does not exist: " + configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic("cannot read config: " + err.Error())
	}

	return &cfg
}

// fetchConfigPath fetches config path from command line flag or environment variable.
// Priority: flag > env > default.
// Default value is empty string.
func fetchConfigPath() string {
	var res string

	flag.StringVar(&res, "config", "", "path to config file")
	flag.Parse()

	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}

	return res
}
