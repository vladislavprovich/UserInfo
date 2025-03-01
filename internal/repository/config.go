package repository

import "time"

type ConfigMongo struct {
	URL               string        `env:"MONGO_URL,required"`
	Host              string        `env:"MONGO_HOST,required"`
	Port              int           `env:"MONGO_PORT,required"`
	User              string        `env:"MONGO_USER,required"`
	Password          string        `env:"MONGO_PASSWORD,required"`
	Database          string        `env:"MONGO_DATABASE,required"`
	AuthSource        string        `env:"MONGO_AUTH_SOURCE,required"`
	CollectionName    string        `env:"MONGO_COLLECTION,required"`
	ConnectionTimeout time.Duration `env:"MONGO_CONN_TIMEOUT"`
	EnsureIdxTimeout  time.Duration `env:"MONGO_IDX_TIMEOUT"`
}
