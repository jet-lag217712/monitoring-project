package snmp

import (
	"context"
	"fmt"
	"os"
	"strings"
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
	community := os.Getenv(device.CommunityEnv)
	if strings.TrimSpace(community) == "" {
		return nil, fmt.Errorf("SNMP community environment variable %q is not set", device.CommunityEnv)
	}

	params := &gosnmp.GoSNMP{
		Target:    device.Host,
		Port:      device.Port,
		Community: community,
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
	return c.WithScaledTimeout(ctx, 1)
}

// WithScaledTimeout returns a child context whose deadline is the single-operation
// SNMP budget multiplied by scale. Multi-walk stages such as IF-MIB inventory
// need a larger budget than a single GET.
func (c *Client) WithScaledTimeout(ctx context.Context, scale int) (context.Context, context.CancelFunc) {
	if scale < 1 {
		scale = 1
	}
	timeout := c.params.Timeout * time.Duration(c.params.Retries+1) * time.Duration(scale)
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return context.WithTimeout(ctx, timeout)
}
