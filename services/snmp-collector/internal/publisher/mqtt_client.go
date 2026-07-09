package publisher

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"math/rand"
	"net/url"
	"os"
	"sync/atomic"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/equate/ogsd/services/snmp-collector/internal/config"
	"github.com/equate/ogsd/services/snmp-collector/internal/metrics"
)

// MQTTClient wraps autopaho with TLS auth and connection-state metrics.
type MQTTClient struct {
	cm        *autopaho.ConnectionManager
	qos       byte
	metrics   *metrics.Collector
	log       *slog.Logger
	connected atomic.Bool
}

// NewMQTTClient starts an autopaho connection manager that reconnects until ctx is cancelled.
func NewMQTTClient(ctx context.Context, cfg config.MQTTConfig, password string, m *metrics.Collector, log *slog.Logger) (*MQTTClient, error) {
	if log == nil {
		log = slog.Default()
	}
	if m == nil {
		return nil, fmt.Errorf("metrics collector is required")
	}

	u, err := url.Parse(cfg.Broker)
	if err != nil {
		return nil, fmt.Errorf("parse mqtt broker: %w", err)
	}

	tlsCfg, err := buildTLSConfig(cfg.TLS, u.Hostname())
	if err != nil {
		return nil, err
	}

	client := &MQTTClient{
		qos:     cfg.QoS,
		metrics: m,
		log:     log,
	}

	cliCfg := autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{u},
		KeepAlive:                     60,
		CleanStartOnInitialConnection: true,
		SessionExpiryInterval:         0,
		TlsCfg:                        tlsCfg,
		ConnectUsername:               cfg.Username,
		ConnectPassword:               []byte(password),
		ReconnectBackoff:              jitterBackoff(cfg.Reconnect.Initial, cfg.Reconnect.Max),
		OnConnectionUp: func(_ *autopaho.ConnectionManager, _ *paho.Connack) {
			client.connected.Store(true)
			m.MQTTConnected.Set(1)
			log.Info("mqtt connected", "broker", cfg.Broker, "client_id", cfg.ClientID)
		},
		OnConnectionDown: func() bool {
			client.connected.Store(false)
			m.MQTTConnected.Set(0)
			log.Warn("mqtt disconnected", "broker", cfg.Broker)
			return true // keep reconnecting
		},
		OnConnectError: func(err error) {
			log.Error("mqtt connect error", "err", err)
		},
		ClientConfig: paho.ClientConfig{
			ClientID: cfg.ClientID,
			OnClientError: func(err error) {
				log.Error("mqtt client error", "err", err)
			},
		},
	}

	cm, err := autopaho.NewConnection(ctx, cliCfg)
	if err != nil {
		return nil, fmt.Errorf("mqtt new connection: %w", err)
	}
	client.cm = cm
	return client, nil
}

// IsConnected reports whether the MQTT session is currently up.
func (c *MQTTClient) IsConnected() bool {
	return c.connected.Load()
}

// AwaitConnection blocks until MQTT is connected or ctx is done.
func (c *MQTTClient) AwaitConnection(ctx context.Context) error {
	return c.cm.AwaitConnection(ctx)
}

// Publish sends a QoS 1 message and waits for PUBACK.
func (c *MQTTClient) Publish(ctx context.Context, topic string, payload []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := c.cm.Publish(ctx, &paho.Publish{
		QoS:     c.qos,
		Topic:   topic,
		Payload: payload,
	})
	if err != nil {
		return fmt.Errorf("mqtt publish %s: %w", topic, err)
	}
	return nil
}

// Done returns a channel closed when the connection manager has shut down.
func (c *MQTTClient) Done() <-chan struct{} {
	return c.cm.Done()
}

func buildTLSConfig(cfg config.MQTTTLSConfig, serverName string) (*tls.Config, error) {
	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: serverName,
	}
	if config.MQTTInsecureSkipVerify() {
		tlsCfg.InsecureSkipVerify = true
		return tlsCfg, nil
	}
	if cfg.CAFile == "" {
		return nil, fmt.Errorf("mqtt tls ca_file is required")
	}
	pem, err := os.ReadFile(cfg.CAFile)
	if err != nil {
		return nil, fmt.Errorf("read ca file: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("parse ca file: no certificates found")
	}
	tlsCfg.RootCAs = pool

	if cfg.CertFile != "" || cfg.KeyFile != "" {
		if cfg.CertFile == "" || cfg.KeyFile == "" {
			return nil, fmt.Errorf("mqtt tls cert_file and key_file must both be set for mTLS")
		}
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("load client cert: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}
	return tlsCfg, nil
}

// jitterBackoff returns exponential backoff with up to 50% random jitter.
func jitterBackoff(initial, max time.Duration) func(int) time.Duration {
	return func(attempt int) time.Duration {
		if attempt < 0 {
			attempt = 0
		}
		// Cap shift to avoid overflow.
		shift := attempt
		if shift > 30 {
			shift = 30
		}
		base := initial * time.Duration(1<<uint(shift))
		if base > max || base <= 0 {
			base = max
		}
		if base <= 0 {
			return initial
		}
		jitter := time.Duration(rand.Int63n(int64(base/2) + 1))
		return base + jitter
	}
}
