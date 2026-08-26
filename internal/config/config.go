package config

import "time"

type Config struct {
	Server ServerConfig
	DB     DBConfig
}

type ServerConfig struct {
	Port           string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
	RequestTimeout time.Duration
}

type DBConfig struct {
	Port     string
	Host     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}
