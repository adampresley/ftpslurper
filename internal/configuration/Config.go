package configuration

import "github.com/app-nerds/configinator"

type Config struct {
	Address          string `flag:"address" env:"ADDRESS" default:"localhost:8080" description:"The address and port to bind the HTTP server to"`
	FTPAddress       string `flag:"ftpaddress" env:"FTP_ADDRESS" default:"0.0.0.0:2022" description:"The address and port to bind the SFTP server to"`
	DSN              string `flag:"dsn" env:"DSN" default:"host=localhost user=inlingo password=password dbname=inlingo port=5432 sslmode=disable" description:"The connection string to the database. See gorm documentation"`
	HttpReadTimeout  int    `flag:"httpreadtimeout" env:"HTTP_READ_TIMEOUT" default:"10" description:"The number of seconds to wait for the client to send a request"`
	HttpWriteTimeout int    `flag:"httpwritetimeout" env:"HTTP_WRITE_TIMEOUT" default:"10" description:"The number of seconds to wait for the server to write a response"`
	LogLevel         string `flag:"loglevel" env:"LOG_LEVEL" default:"info" description:"The log level to use. Valid values are 'debug', 'info', 'warn', and 'error'"`
	PageSize         int    `flag:"pagesize" env:"PAGE_SIZE" default:"25" description:"The number of items to return per page"`
}

func LoadConfig() Config {
	config := Config{}
	configinator.Behold(&config)
	return config
}
