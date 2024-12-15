# Connection Checker

The **Connection Checker** is a lightweight Go-based worker that monitors the connectivity to a specified server. If the connection cannot be established, the application sends an email alert to a designated recipient. To prevent repeated notifications, the worker uses a flag to ensure only one email is sent per failure incident. The application also exposes an HTTP endpoint to manually reset the flag, allowing for subsequent notifications if needed.

## Features

- **Connection Monitoring**: Periodically checks connectivity to the target server (IP and port).
- **Email Notifications**: Sends an email alert to the configured recipient when a connection failure is detected.
- **Flag Control**: Prevents multiple consecutive email notifications for the same failure.
- **Reset Endpoint**: An HTTP endpoint allows manual resetting of the notification flag.
- **Rate Limiting**: Prevents excessive requests to the reset endpoint.

## Tools and Libraries Used

- **[logrus](https://github.com/sirupsen/logrus)**: Provides structured logging for the application.
- **[golang.org/x/time](https://pkg.go.dev/golang.org/x/time)**: Used for implementing rate limiting in the HTTP endpoint.
- **[golang.org/x/sys](https://pkg.go.dev/golang.org/x/sys)**: Provides low-level OS functionality (indirect dependency).

## Configuration

The application requires a JSON configuration file to set up its parameters. Below is an explanation of each configuration item:

```json
{
  "app_port": "8080",
  "target_ip": "192.168.1.2",
  "target_port": "51820",
  "smtp_server": "smtp.example.com",
  "smtp_port": "587",
  "sender_email": "your_email@example.com",
  "sender_password": "your_password",
  "recipient_email": "admin@example.com",
  "check_interval": 30,
  "reset_token": "resettoken",
  "rate_limit_threshold": 5
}
```

### Configuration Items

- **`app_port`**: The port on which the HTTP server will run (used for the reset endpoint).
- **`target_ip`**: The IP address of the server to monitor.
- **`target_port`**: The port of the server to monitor.
- **`smtp_server`**: The SMTP server used to send email notifications.
- **`smtp_port`**: The port for the SMTP server (e.g., `587` for TLS, `465` for SSL).
- **`sender_email`**: The email address used to send the notification emails.
- **`sender_password`**: The password for the sender email account.
- **`recipient_email`**: The email address that will receive the alert notifications.
- **`check_interval`**: The interval (in seconds) at which the connection to the target server is checked.
- **`reset_token`**: The token required to authorize requests to the reset endpoint.
- **`rate_limit_threshold`**: The maximum number of reset endpoint requests allowed per second.

## Endpoint Details

### Reset Alert Endpoint

- **URL**: `/reset-alert`
- **Method**: `POST`
- **Headers**:
  - `Authorization`: The reset token specified in the configuration file (`reset_token`).
- **Rate Limit**: Limited to the number of requests specified by the `rate_limit_threshold` in the configuration.

#### Example Request

```bash
curl -X POST http://localhost:8080/reset-alert \
     -H "Authorization: resettoken" \
     -H "Content-Type: application/json"
```

#### Description

This endpoint resets the notification flag, allowing the application to send a new alert email if the connection issue persists or recurs.

- **Success Response**: If the reset is successful, a JSON response is returned:
  ```json
  {
    "message": "Alert status reset successfully"
  }
  ```

## Setup Instructions

### 1. Install Go

Make sure you have Go installed on your system. You can download it from [Go's official website](https://golang.org/dl/).

### 2. Clone the Repository

Clone the project repository to your local machine:

```bash
git clone https://nyxordinal.dev/connection-checker.git
cd connection-checker
```

### 3. Install Dependencies

Run the following command to download the required dependencies:

```bash
go mod tidy
```

### 4. Create the Configuration File

Copy the provided `config.json.example` file to a new file named `config.json`:

```bash
cp config.json.example config.json
```

Edit the `config.json` file and populate it with the required configuration. Replace placeholders like `your_email@example.com` and `your_password` with actual values.

### 5. Build and Run the Application

Build and run the application:

```bash
go build -o connection-checker
./connection-checker
```

### 6. Test the Application

- The application will start monitoring the target server.
- If the connection fails, an email alert will be sent to the configured recipient.
- You can manually reset the notification flag by sending a POST request to the `/reset-alert` endpoint.

### 7. Monitor Logs

The application logs are printed to the console. You can monitor them to observe connection checks, email notifications, and HTTP endpoint requests.

## Future Improvements

- Adding support for multiple targets.
- Improving authentication for the reset endpoint (e.g., using OAuth or API keys).
- Extending alert mechanisms (e.g., SMS, Slack notifications).

## License

This project is licensed under the MIT License. Feel free to use and modify it as needed.

## Developer Team

Developed with passion by [Nyxordinal](https://nyxordinal.dev)
