package usecase

import (
	"context"
	"net"
)

// DNSResolver is an injectable port for DNS TXT record lookups.
// Production implementation: NetResolver() (stdlib).
// Test implementation: a stub returning preset records without network calls.
type DNSResolver interface {
	LookupTXT(ctx context.Context, name string) ([]string, error)
}

type netResolver struct{}

func (r *netResolver) LookupTXT(ctx context.Context, name string) ([]string, error) {
	return net.DefaultResolver.LookupTXT(ctx, name)
}

// NetResolver returns the production stdlib DNS resolver.
func NetResolver() DNSResolver {
	return &netResolver{}
}
