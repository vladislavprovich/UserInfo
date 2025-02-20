package config

import (
	"flag"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/vladislavprovich/UserInfo/internal/storage"
)

type Config struct {
	Logger  LoggerConfig
	Logging LoggingConfig
	Tracing TracingConfig
	GRPC    GRPCConfig
	Otel    OtelConfig
	Rabbit  RabbitMQConfig
	MongoDB storage.ConfigMongo
}

type LoggerConfig struct {
	Env string `env:"LOG_ENV"`
}

type LoggingConfig struct {
	LevelLoki string `env:"LOG_LEVEL_LOKI"`
	Format    string `env:"LOG_FORMAT"`
	LokiURL   string `env:"LOG_LOKI_URL"`
	LogDir    string `env:"LOG_DIR"`
}

type TracingConfig struct {
	TempoURL  string `env:"TRACING_TEMPO_URL"`
	NameSpase string `env:"TRACING_NAME_SPASE"`
}

type GRPCConfig struct {
	PortGRPC    int           `env:"GRPC_PORT"`
	TimeoutGRPC time.Duration `env:"GRPC_TIMEOUT"`
}

type OtelConfig struct {
	Endpoint          string        `env:"OTEL_ENDPOINT" envDefault:"http://otel-lgtm:4317"`
	MetricsPort       int           `env:"OTEL_METRICS_PORT" envDefault:"9090"`
	Adr               string        `env:"OTEL_ADR" envDefault:"localhost"`
	ReadTimeout       time.Duration `env:"OTEL_READ_TIMEOUT" envDefault:"5s"`
	WriteTimeout      time.Duration `env:"OTEL_WRITE_TIMEOUT" envDefault:"5s"`
	ReadHeaderTimeout time.Duration `env:"OTEL_READ_HEADER_TIMEOUT" envDefault:"5s"`
}

type RabbitMQConfig struct {
	User     string `env:"RABBIT_USER"`
	Password string `env:"RABBIT_PASSWORD"`
	Host     string `env:"RABBIT_HOST"`
	Port     int    `env:"RABBIT_PORT"`
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
