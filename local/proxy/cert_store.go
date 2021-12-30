package proxy

import (
	"crypto/tls"
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"github.com/symfony-cli/cert"
)

type certStore struct {
	proxyCfg *Config
	ca       *cert.CA
	lock     sync.Mutex
	cache    *lru.ARCCache
}

// newCertStore creates a store to keep SSL certificates in memory
func (p *Proxy) newCertStore(ca *cert.CA) *certStore {
	cache, _ := lru.NewARC(1024)
	return &certStore{
		proxyCfg: p.Config,
		ca:       ca,
		cache:    cache,
	}
}

// getCertificate returns a valid certificate for the given domain name
func (c *certStore) getCertificate(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	name := c.proxyCfg.NormalizeDomain(clientHello.ServerName)
	if val, ok := c.cache.Get(name); ok {
		cert := val.(tls.Certificate)
		return &cert, nil
	}
	cert, err := c.ca.CreateCert([]string{name})
	if err != nil {
		return nil, err
	}
	c.cache.Add(name, cert)
	return &cert, nil
}
