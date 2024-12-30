package main

import (
	"encoding/json"
	"os"
	"time"
)

type Config struct {
	TargetIP           string        `json:"target_ip"`
	SMTPServer         string        `json:"smtp_server"`
	SMTPPort           string        `json:"smtp_port"`
	SenderEmail        string        `json:"sender_email"`
	SenderPassword     string        `json:"sender_password"`
	RecipientEmail     string        `json:"recipient_email"`
	CheckInterval      time.Duration `json:"check_interval"`
	AppPort            string        `json:"app_port"`
	ResetToken         string        `json:"reset_token"`
	RateLimitThreshold int           `json:"rate_limit_threshold"`
	JWTSecret          string        `json:"jwt_secret"`
	Username           string        `json:"username"`
	Password           string        `json:"password"`
}

func loadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
