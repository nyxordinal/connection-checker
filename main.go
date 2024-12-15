package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

type Config struct {
	TargetIP           string        `json:"target_ip"`
	TargetPort         string        `json:"target_port"`
	SMTPServer         string        `json:"smtp_server"`
	SMTPPort           string        `json:"smtp_port"`
	SenderEmail        string        `json:"sender_email"`
	SenderPassword     string        `json:"sender_password"`
	RecipientEmail     string        `json:"recipient_email"`
	CheckInterval      time.Duration `json:"check_interval"`
	AppPort            string        `json:"app_port"`
	ResetToken         string        `json:"reset_token"`
	RateLimitThreshold int           `json:"rate_limit_threshold"`
}

var (
	alertSent bool
	mutex     sync.Mutex
	logger    *logrus.Logger
)

type JsonResponse struct {
	Message string `json:"message"`
}

func CreateLogger(level logrus.Level) *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05Z07:00",
	})
	logger.SetLevel(level)
	return logger
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

func loadHTMLTemplate(filename string) (string, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func main() {
	logger = CreateLogger(logrus.InfoLevel)

	config, err := loadConfig("config.json")
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
		return
	}

	alertHtml, err := loadHTMLTemplate("email_alert.html")
	if err != nil {
		logger.Fatalf("Failed to load alert email template: %v", err)
		return
	}

	restoredHtml, err := loadHTMLTemplate("email_restored.html")
	if err != nil {
		logger.Fatalf("Failed to load restored email template: %v", err)
		return
	}

	go startHTTPServer(config.AppPort, config.ResetToken, config.RateLimitThreshold)

	for {
		if !checkConnection(config.TargetIP, config.TargetPort) {
			mutex.Lock()
			if !alertSent {
				timestamp := time.Now().UTC().Format("2006-01-02 15:04:05 MST")
				if err := sendEmail(config, "Connection Alert", fmt.Sprintf(alertHtml, config.TargetIP, config.TargetPort, timestamp)); err != nil {
					logger.Errorf("Failed to send email: %v", err)
				} else {
					logger.Info("Alert email sent")
					alertSent = true
				}
			}
			mutex.Unlock()
		} else {
			logger.Infof("Connection to %s:%s is healthy.", config.TargetIP, config.TargetPort)
			mutex.Lock()
			if alertSent {
				timestamp := time.Now().UTC().Format("2006-01-02 15:04:05 MST")
				if err := sendEmail(config, "Connection Restored", fmt.Sprintf(restoredHtml, config.TargetIP, config.TargetPort, timestamp)); err != nil {
					logger.Errorf("Failed to send email: %v", err)
				} else {
					logger.Info("Restored email sent")
					alertSent = false
				}
			}
			mutex.Unlock()
		}
		time.Sleep(config.CheckInterval * time.Millisecond)
	}
}

func checkConnection(ip, port string) bool {
	address := net.JoinHostPort(ip, port)
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		logger.Warnf("Connection failed: %v", err)
		return false
	}
	defer conn.Close()
	return true
}

func sendEmail(config *Config, subject, body string) error {
	auth := smtp.PlainAuth("", config.SenderEmail, config.SenderPassword, config.SMTPServer)
	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s\r\n", config.RecipientEmail, subject, body))

	err := smtp.SendMail(
		net.JoinHostPort(config.SMTPServer, config.SMTPPort),
		auth,
		config.SenderEmail,
		[]string{config.RecipientEmail},
		msg,
	)
	return err
}

func startHTTPServer(port, resetToken string, rateLimitThreshold int) {
	rateLimiter := rate.NewLimiter(rate.Every(1*time.Second), rateLimitThreshold)

	http.HandleFunc("/reset-alert", func(w http.ResponseWriter, r *http.Request) {
		if !rateLimiter.Allow() {
			logger.Warn("Rate limit reached because of too many requests")
			return
		}

		if r.Method != http.MethodPost {
			return
		}

		token := r.Header.Get("Authorization")
		if token == "" || token != resetToken {
			logger.Warn("Unauthorized attempt to reset alert status")
			return
		}

		mutex.Lock()
		alertSent = false
		mutex.Unlock()

		logger.Info("Alert status reset via HTTP endpoint")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := JsonResponse{
			Message: "Alert status reset successfully",
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			logger.Error("Failed to encode JSON response: ", err)
		}
	})

	logger.Infof("HTTP server is running on port %s...", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
		logger.Fatal("Server failed: ", err)
	}
}
