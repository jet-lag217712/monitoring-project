package mqtt

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"math/rand"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/equate/ogsd/services/ingestion-service/internal/config"
	"github.com/equate/ogsd/services/ingestion-service/internal/handler"
	"github.com/equate/ogsd/services/ingestion-service/internal/metrics"
)

// Subscriber consumes MQTT telemetry with deferred (manual) ACK.
type Subscriber struct {
	cm         *autopaho.ConnectionManager
	handler    *handler.Handler
	metrics    *metrics.Ingestion
	log        *slog.Logger
	connected  atomic.Bool
	subscribed atomic.Bool
	mu         sync.Mutex // serializes message handling for ACK ordering
}

// NewSubscriber starts an autopaho connection that subscribes and processes messages.
// CleanStart is false so unacked QoS 1 messages are redelivered after reconnect.
func NewSubscriber(ctx context.Context, cfg config.MQTTConfig, password string, h *handler.Handler, m *metrics.Ingestion, log *slog.Logger) (*Subscriber, error) {
	if log == nil {
		log = slog.Default()
	}
	if m == nil {
		return nil, fmt.Errorf("metrics are required")
	}
	if h == nil {
		return nil, fmt.Errorf("handler is required")
	}

	u, err := url.Parse(cfg.Broker)
	if err != nil {
		return nil, fmt.Errorf("parse mqtt broker: %w", err)
	}

	tlsCfg, err := buildTLSConfig(cfg.TLS, u.Hostname())
	if err != nil {
		return nil, err
	}

	sub := &Subscriber{
		handler: h,
		metrics: m,
		log:     log,
	}

	cliCfg := autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{u},
		KeepAlive:                     60,
		CleanStartOnInitialConnection: false,
		SessionExpiryInterval:         86400, // 24h
		TlsCfg:                        tlsCfg,
		ConnectUsername:               cfg.Username,
		ConnectPassword:               []byte(password),
		ReconnectBackoff:              jitterBackoff(cfg.Reconnect.Initial, cfg.Reconnect.Max),
		OnConnectionUp: func(cm *autopaho.ConnectionManager, _ *paho.Connack) {
			sub.connected.Store(true)
			m.MQTTConnected.Set(1)
			log.Info("mqtt connected", "broker", cfg.Broker, "client_id", cfg.ClientID)
			topics := cfg.SubscribeTopics()
			subs := make([]paho.SubscribeOptions, 0, len(topics))
			for _, topic := range topics {
				subs = append(subs, paho.SubscribeOptions{Topic: topic, QoS: cfg.QoS})
			}
			if _, err := cm.Subscribe(context.Background(), &paho.Subscribe{
				Subscriptions: subs,
			}); err != nil {
				sub.subscribed.Store(false)
				m.MQTTSubscribed.Set(0)
				log.Error("mqtt subscribe failed", "topics", topics, "err", err)
				return
			}
			sub.subscribed.Store(true)
			m.MQTTSubscribed.Set(1)
			log.Info("mqtt subscribed", "topics", topics)
		},
		OnConnectionDown: func() bool {
			sub.connected.Store(false)
			sub.subscribed.Store(false)
			m.MQTTConnected.Set(0)
			m.MQTTSubscribed.Set(0)
			log.Warn("mqtt disconnected", "broker", cfg.Broker)
			return true
		},
		OnConnectError: func(err error) {
			log.Error("mqtt connect error", "err", err)
		},
		ClientConfig: paho.ClientConfig{
			ClientID:                   cfg.ClientID,
			EnableManualAcknowledgment: true,
			OnClientError: func(err error) {
				log.Error("mqtt client error", "err", err)
			},
			OnPublishReceived: []func(paho.PublishReceived) (bool, error){
				func(pr paho.PublishReceived) (bool, error) {
					sub.mu.Lock()
					defer sub.mu.Unlock()
					ack := sub.handler.Handle(context.Background(), pr.Packet.Topic, pr.Packet.Payload)
					if !ack {
						return false, nil
					}
					if err := pr.Client.Ack(pr.Packet); err != nil {
						log.Error("mqtt ack failed", "topic", pr.Packet.Topic, "err", err)
						return false, err
					}
					return true, nil
				},
			},
		},
	}

	cm, err := autopaho.NewConnection(ctx, cliCfg)
	if err != nil {
		return nil, fmt.Errorf("mqtt new connection: %w", err)
	}
	sub.cm = cm
	return sub, nil
}

// IsConnected reports whether the MQTT session is currently up.
func (s *Subscriber) IsConnected() bool {
	return s.connected.Load()
}

// IsReady reports whether MQTT is connected and the telemetry topic is subscribed.
func (s *Subscriber) IsReady() bool {
	return s.connected.Load() && s.subscribed.Load()
}

// AwaitConnection blocks until MQTT is connected or ctx is done.
func (s *Subscriber) AwaitConnection(ctx context.Context) error {
	return s.cm.AwaitConnection(ctx)
}

// AwaitReady blocks until MQTT is connected and subscribed or ctx is done.
func (s *Subscriber) AwaitReady(ctx context.Context) error {
	if s.IsReady() {
		return nil
	}
	if err := s.AwaitConnection(ctx); err != nil {
		return err
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		if s.IsReady() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// Done returns a channel closed when the connection manager has shut down.
func (s *Subscriber) Done() <-chan struct{} {
	return s.cm.Done()
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

func jitterBackoff(initial, max time.Duration) func(int) time.Duration {
	return func(attempt int) time.Duration {
		if attempt < 0 {
			attempt = 0
		}
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
