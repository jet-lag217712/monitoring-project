package snmp

import (
	"context"
	"fmt"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/equate/ogsd/services/snmp-collector/internal/config"
)

// Client is a thin wrapper around gosnmp for a single device session.
type Client struct {
	params *gosnmp.GoSNMP
}

// NewClient builds an SNMPv2c client from device and shared SNMP settings.
func NewClient(device config.DeviceConfig, snmpCfg config.SNMPConfig) (*Client, error) {
	if device.Version != "2c" {
		return nil, fmt.Errorf("unsupported SNMP version %q", device.Version)
	}

	params := &gosnmp.GoSNMP{
		Target:    device.Host,
		Port:      device.Port,
		Community: device.Community,
		Version:   gosnmp.Version2c,
		Timeout:   snmpCfg.Timeout,
		Retries:   snmpCfg.Retries,
		MaxOids:   gosnmp.MaxOids,
	}

	return &Client{params: params}, nil
}

// Connect opens the underlying UDP connection.
func (c *Client) Connect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := c.params.Connect(); err != nil {
		return fmt.Errorf("snmp connect: %w", err)
	}
	return nil
}

// Close closes the underlying connection.
func (c *Client) Close() error {
	if c.params.Conn != nil {
		return c.params.Conn.Close()
	}
	return nil
}

// Get performs an SNMP GET for the given OIDs.
func (c *Client) Get(ctx context.Context, oids []string) (*gosnmp.SnmpPacket, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	pkt, err := c.params.Get(oids)
	if err != nil {
		return nil, fmt.Errorf("snmp get: %w", err)
	}
	return pkt, nil
}

// Walk performs a bulk walk under rootOID, invoking walkFn for each PDU.
func (c *Client) Walk(ctx context.Context, rootOID string, walkFn gosnmp.WalkFunc) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := c.params.BulkWalk(rootOID, walkFn); err != nil {
		return fmt.Errorf("snmp walk %s: %w", rootOID, err)
	}
	return nil
}

// WithTimeout returns a child context bounded by the SNMP client timeout multiplied by (retries + 1).
func (c *Client) WithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	timeout := c.params.Timeout * time.Duration(c.params.Retries+1)
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return context.WithTimeout(ctx, timeout)
}
