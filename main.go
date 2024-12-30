package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	probing "github.com/prometheus-community/pro-bing"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

type AppContext struct {
	AlertSent bool
	Mutex     sync.Mutex
	Logger    *logrus.Logger
	Config    *Config
	DB        *Database
}

var appCtx AppContext

type JsonResponse struct {
	Message string `json:"message"`
}

func initApp() {
	appCtx.Logger = CreateLogger(logrus.InfoLevel)

	config, err := loadConfig("config.json")
	if err != nil {
		appCtx.Logger.Fatalf("Failed to load configuration: %v", err)
		return
	}
	appCtx.Config = config

	db, err := initDB()
	if err != nil {
		appCtx.Logger.Fatalf("Failed to setup database: %v", err)
		return
	}
	appCtx.DB = db
}

func main() {
	initApp() // Must be called before using appCtx

	alertHtml, restoredHtml, err := getHTMLTemplates()
	if err != nil {
		appCtx.Logger.Fatalf("Failed to get HTML template: %v", err)
		return
	}

	pinger, err := probing.NewPinger(appCtx.Config.TargetIP)
	if err != nil {
		appCtx.Logger.Fatalf("Failed to create pinger: %v", err)
		return
	}

	pinger.Count = 1
	pinger.OnRecv = func(pkt *probing.Packet) {
		appCtx.Logger.Infof("Received response from %s: seq=%d time=%v", pkt.IPAddr, pkt.Seq, pkt.Rtt)
	}

	go startHTTPServer()

	for {
		if !checkConnection(pinger, appCtx.Config.TargetIP) {
			handleConnectionStatus("Failed", "Connection Alert", alertHtml)
		} else {
			handleConnectionStatus("Healthy", "Connection Restored", restoredHtml)
		}
		time.Sleep(appCtx.Config.CheckInterval * time.Millisecond)
	}
}

func handleConnectionStatus(status, emailSubject, emailContent string) {
	appCtx.Mutex.Lock()
	defer appCtx.Mutex.Unlock()

	if status == "Failed" && !appCtx.AlertSent {
		sendAlertEmail(emailSubject, emailContent)
		appCtx.AlertSent = true
	} else if status == "Healthy" && appCtx.AlertSent {
		sendAlertEmail(emailSubject, emailContent)
		appCtx.AlertSent = false
	}

	if err := appCtx.DB.updateConnectionStatus(status); err != nil {
		appCtx.Logger.Errorf("Failed to update connection status in DB: %v", err)
	}

	if err := appCtx.DB.logConnectionStatus(status); err != nil {
		appCtx.Logger.Errorf("Failed to log connection status: %v", err)
	}
}

func checkConnection(pinger *probing.Pinger, ip string) bool {
	if err := pinger.Run(); err != nil {
		appCtx.Logger.Warnf("Failed to ping %s: %v", ip, err)
		return false
	}
	return true
}

func createRateLimiter(limit int, interval time.Duration) *rate.Limiter {
	return rate.NewLimiter(rate.Every(interval), limit)
}

func startHTTPServer() {
	rateLimiterPerSecond := createRateLimiter(appCtx.Config.RateLimitThreshold, time.Second)
	rateLimiterPerMinute := createRateLimiter(appCtx.Config.RateLimitThreshold, time.Minute)

	http.HandleFunc("/reset-alert", rateLimitedHandler(rateLimiterPerSecond, resetAlertHandler))
	http.HandleFunc("/status", apiAuthMiddleware(rateLimitedHandler(rateLimiterPerMinute, statusHandler)))
	http.HandleFunc("/logs", apiAuthMiddleware(rateLimitedHandler(rateLimiterPerSecond, logsHandler)))

	http.HandleFunc("/login", authPageMiddleware(loginHandler))
	http.HandleFunc("/", authMiddleware(indexHandler))

	appCtx.Logger.Infof("HTTP server is running on port %s...", appCtx.Config.AppPort)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", appCtx.Config.AppPort), nil); err != nil {
		appCtx.Logger.Fatal("Server failed: ", err)
	}
}
