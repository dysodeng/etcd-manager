package etcd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/dysodeng/config-center/internal/config"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Client struct {
	cli *clientv3.Client
}

func NewClient(cfg config.EtcdConfig) (*Client, error) {
	etcdCfg := clientv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: 5 * time.Second,
		Username:    cfg.Username,
		Password:    cfg.Password,
	}
	if cfg.TLS.Enabled {
		tlsCfg, err := newTLSConfig(cfg.TLS)
		if err != nil {
			return nil, fmt.Errorf("etcd tls config: %w", err)
		}
		etcdCfg.TLS = tlsCfg
	}
	cli, err := clientv3.New(etcdCfg)
	if err != nil {
		return nil, err
	}
	return &Client{cli: cli}, nil
}

func (c *Client) Close() error { return c.cli.Close() }

func (c *Client) Get(ctx context.Context, key string) (*clientv3.GetResponse, error) {
	return c.cli.Get(ctx, key)
}

func (c *Client) GetWithPrefix(ctx context.Context, prefix string, limit int64) (*clientv3.GetResponse, error) {
	opts := []clientv3.OpOption{clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend)}
	if limit > 0 {
		opts = append(opts, clientv3.WithLimit(limit))
	}
	return c.cli.Get(ctx, prefix, opts...)
}

func (c *Client) Put(ctx context.Context, key, value string) (*clientv3.PutResponse, error) {
	return c.cli.Put(ctx, key, value)
}

func (c *Client) Delete(ctx context.Context, key string) (*clientv3.DeleteResponse, error) {
	return c.cli.Delete(ctx, key)
}

func (c *Client) DeleteWithPrefix(ctx context.Context, prefix string) (*clientv3.DeleteResponse, error) {
	return c.cli.Delete(ctx, prefix, clientv3.WithPrefix())
}

func (c *Client) Watch(ctx context.Context, prefix string, rev int64) clientv3.WatchChan {
	opts := []clientv3.OpOption{clientv3.WithPrefix()}
	if rev > 0 {
		opts = append(opts, clientv3.WithRev(rev))
	}
	return c.cli.Watch(ctx, prefix, opts...)
}

func (c *Client) MemberList(ctx context.Context) (*clientv3.MemberListResponse, error) {
	return c.cli.MemberList(ctx)
}

func (c *Client) Status(ctx context.Context, endpoint string) (*clientv3.StatusResponse, error) {
	return c.cli.Status(ctx, endpoint)
}

func (c *Client) Endpoints() []string { return c.cli.Endpoints() }

func newTLSConfig(cfg config.TLSConfig) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, err
	}
	caCert, err := os.ReadFile(cfg.CAFile)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caCert)
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
		MinVersion:   tls.VersionTLS12,
	}, nil
}
