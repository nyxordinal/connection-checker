package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	probing "github.com/prometheus-community/pro-bing"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

var (
	alertSent bool
	mutex     sync.Mutex
	logger    *logrus.Logger
)

type JsonResponse struct {
	Message string `json:"message"`
}

func main() {
	logger = CreateLogger(logrus.InfoLevel)

	config, err := loadConfig("config.json")
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
		return
	}

	alertHtml, restoredHtml, err := getHTMLTemplates()
	if err != nil {
		logger.Fatalf("Failed to get HTML template: %v", err)
		return
	}

	db := &Database{}
	if err := db.initDB(); err != nil {
		logger.Fatalf("Failed to setup database: %v", err)
		return
	}

	pinger, err := probing.NewPinger(config.TargetIP)
	if err != nil {
		logger.Fatalf("Failed to create pinger: %v", err)
		return
	}

	pinger.Count = 1
	pinger.OnRecv = func(pkt *probing.Packet) {
		logger.Infof("Received response from %s: seq=%d time=%v", pkt.IPAddr, pkt.Seq, pkt.Rtt)
	}

	go startHTTPServer(db, config.AppPort, config.ResetToken, config.RateLimitThreshold)

	for {
		if !checkConnection(pinger, config.TargetIP) {
			db.logConnectionStatus("failed")
			mutex.Lock()
			if !alertSent {
				timestamp := time.Now().UTC().Format("2006-01-02 15:04:05 MST")
				if err := sendEmail(config, "Connection Alert", fmt.Sprintf(alertHtml, config.TargetIP, timestamp)); err != nil {
					logger.Errorf("Failed to send email: %v", err)
				} else {
					logger.Info("Alert email sent")
					alertSent = true
				}

				if err := db.updateConnectionStatus("failed", timestamp); err != nil {
					logger.Errorf("Failed to update connection status in DB: %v", err)
				}
			}
			mutex.Unlock()
		} else {
			db.logConnectionStatus("success")
			logger.Infof("Connection to %s is healthy.", config.TargetIP)
			mutex.Lock()
			if alertSent {
				timestamp := time.Now().UTC().Format("2006-01-02 15:04:05 MST")
				if err := sendEmail(config, "Connection Restored", fmt.Sprintf(restoredHtml, config.TargetIP, timestamp)); err != nil {
					logger.Errorf("Failed to send email: %v", err)
				} else {
					logger.Info("Restored email sent")
					alertSent = false
				}

				if err := db.updateConnectionStatus("failed", timestamp); err != nil {
					logger.Errorf("Failed to update connection status in DB: %v", err)
				}
			}
			mutex.Unlock()
		}
		time.Sleep(config.CheckInterval * time.Millisecond)
	}
}

func checkConnection(pinger *probing.Pinger, ip string) bool {
	if err := pinger.Run(); err != nil {
		logger.Warnf("Failed to ping %s: %v", ip, err)
		return false
	}
	return true
}

func startHTTPServer(db *Database, port, resetToken string, rateLimitThreshold int) {
	rateLimiterPerSecond := rate.NewLimiter(rate.Every(1*time.Second), rateLimitThreshold)
	rateLimiterPerMinute := rate.NewLimiter(rate.Every(1*time.Minute), rateLimitThreshold)

	http.HandleFunc("/reset-alert", func(w http.ResponseWriter, r *http.Request) {
		if !rateLimiterPerSecond.Allow() {
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

	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		if !rateLimiterPerMinute.Allow() {
			logger.Warn("Rate limit reached because of too many requests")
			return
		}

		status, lastEmailSent, err := db.getConnectionStatus()
		if err != nil {
			http.Error(w, "Failed to fetch connection status", http.StatusInternalServerError)
			return
		}

		response := map[string]string{
			"connection_status": status,
			"last_email_sent":   lastEmailSent,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	http.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		if !rateLimiterPerSecond.Allow() {
			logger.Warn("Rate limit reached because of too many requests")
			return
		}

		page := 1
		perPage := 25
		if r.URL.Query().Get("page") != "" {
			page = atoi(r.URL.Query().Get("page"))
		}
		if r.URL.Query().Get("per_page") != "" {
			perPage = atoi(r.URL.Query().Get("per_page"))
		}

		logs, err := db.getConnectionLogs(page, perPage)
		if err != nil {
			logger.Errorf("Failed to fetch logs: %v", err)
			http.Error(w, "Failed to fetch logs", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(logs); err != nil {
			logger.Errorf("Failed to encode logs: %v", err)
			http.Error(w, "Failed to encode logs", http.StatusInternalServerError)
		}
	})

	http.Handle("/", http.FileServer(http.Dir("./static")))

	logger.Infof("HTTP server is running on port %s...", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
		logger.Fatal("Server failed: ", err)
	}
}
