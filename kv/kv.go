// Package kv provides version agnostic methods for read, write and list of secrets from @hashicorp Vault's KV secret engines
package kv

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/hashicorp/vault/api"
)

// Constants
const (
	ReadPrefix  = "data"
	WritePrefix = ReadPrefix
	ListPrefix  = "metadata"
)

// Client represents a KV client
type Client struct {
	client  *api.Client
	Version int
}

// New creates a new kv.Client with the Vault client c and a path p long enough to determine the mount path of the engine
// p = secret/ -> K/V engine mount path secret/
// p = secret  -> error
func New(c *api.Client, p string) (*Client, error) {
	if !strings.ContainsRune(p, '/') {
		return nil, fmt.Errorf("path %s must contain at least one '/'", p)
	}
	version, err := getVersion(c, p)
	if err != nil {
		return nil, err
	}
	return &Client{client: c, Version: version}, nil
}

// Read a secret from a K/V version 1 or 2
func (c *Client) Read(p string) (map[string]interface{}, error) {
	origPath := p
	if c.Version == 2 {
		p = fixPath(p, ReadPrefix)
	}
	s, err := c.client.Logical().Read(p)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, fmt.Errorf("failed to read path %s", origPath)
	}
	if c.Version == 2 {
		return s.Data["data"].(map[string]interface{}), nil
	}
	return s.Data, nil
}

// Write a secret to a K/V version 1 or 2
func (c *Client) Write(p string, data map[string]interface{}) error {
	if c.Version == 2 {
		p = fixPath(p, WritePrefix)
		data = map[string]interface{}{
			"data": data,
		}
	}
	_, err := c.client.Logical().Write(p, data)
	return err
}

// List secrets from a K/V version 1 or 2
func (c *Client) List(p string) ([]string, error) {
	origPath := p
	if c.Version == 2 {
		p = fixPath(p, ListPrefix)
	}
	s, err := c.client.Logical().List(p)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, fmt.Errorf("failed to list path %s", origPath)
	}
	keys := []string{}
	for _, v := range s.Data["keys"].([]interface{}) {
		keys = append(keys, v.(string))
	}
	return keys, nil
}

// fixPath inserts the API prefix for the appropriate version if necessary
func fixPath(p, prefix string) string {
	pp := strings.Split(p, "/")
	return path.Join(append(pp[:1], append([]string{prefix}, pp[1:]...)...)...)
}

// getVersion of the KV engine
func getVersion(c *api.Client, p string) (int, error) {
	mounts, err := c.Sys().ListMounts()
	if err != nil {
		return 0, err
	}
	for k, m := range mounts {
		if !strings.HasPrefix(p, k) {
			continue
		}
		switch m.Type {
		case "kv":
			return strconv.Atoi(m.Options["version"])
		case "generic":
			return 1, nil
		default:
			return 0, fmt.Errorf("matching mount %s for path %s is not of type kv", k, p)
		}
	}
	return 0, fmt.Errorf("failed to get mount for path: %s", p)
}