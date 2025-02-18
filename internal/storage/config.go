package storage

type ConfigMongo struct {
	Host       string `env:"MONGO_HOST,required"`
	Port       int    `env:"MONGO_PORT,required"`
	User       string `env:"MONGO_USER,required"`
	Password   string `env:"MONGO_PASSWORD,required"`
	Database   string `env:"MONGO_DATABASE,required"`
	AuthSource string `env:"MONGO_AUTH_SOURCE,required"`
}
