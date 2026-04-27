package brokerapi

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type dependencyFetchHTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type publicRegistryHTTPFetcher struct {
	resolver hostResolver
	client   dependencyFetchHTTPDoer
}

func newPublicRegistryHTTPFetcher() publicRegistryHTTPFetcher {
	resolver := net.DefaultResolver
	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           dependencyFetchPinnedDialContext(resolver),
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          64,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return publicRegistryHTTPFetcher{
		resolver: resolver,
		client: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (f publicRegistryHTTPFetcher) Fetch(ctx context.Context, req DependencyFetchRequestObject, lease dependencyRegistryAuthLease) (io.ReadCloser, dependencyRegistryFetchMetadata, error) {
	host, err := validatePublicRegistryRequest(ctx, f.resolverOrDefault(), req, lease)
	if err != nil {
		return nil, dependencyRegistryFetchMetadata{}, err
	}
	resolvedPath, err := dependencyFetchPathForRequest(req)
	if err != nil {
		return nil, dependencyRegistryFetchMetadata{}, err
	}
	endpoint := &url.URL{Scheme: "https", Host: host, Path: resolvedPath}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, dependencyRegistryFetchMetadata{}, err
	}
	httpReq.Header.Set("accept", "application/octet-stream")
	httpResp, err := f.clientOrDefault().Do(httpReq)
	if err != nil {
		return nil, dependencyRegistryFetchMetadata{}, err
	}
	return successfulRegistryHTTPResponse(httpResp)
}

func validatePublicRegistryRequest(ctx context.Context, resolver hostResolver, req DependencyFetchRequestObject, lease dependencyRegistryAuthLease) (string, error) {
	if err := validatePublicRegistryIdentity(req, lease); err != nil {
		return "", err
	}
	host := strings.TrimSpace(strings.ToLower(req.RegistryIdentity.CanonicalHost))
	if host == "" {
		return "", errors.New("dependency registry canonical_host is required")
	}
	if ip := net.ParseIP(host); ip != nil {
		if isDeniedRuntimeIP(ip) {
			return "", fmt.Errorf("dependency registry destination blocked: %s", ip.String())
		}
		return host, nil
	}
	if err := enforceResolvedPublicIPs(ctx, resolver, host); err != nil {
		return "", err
	}
	return host, nil
}

func validatePublicRegistryIdentity(req DependencyFetchRequestObject, lease dependencyRegistryAuthLease) error {
	if req.RegistryIdentity.DescriptorKind != "package_registry" {
		return errors.New("dependency registry descriptor_kind must be package_registry")
	}
	if !req.RegistryIdentity.TLSRequired {
		return errors.New("dependency registry tls_required must be true")
	}
	if req.RegistryIdentity.PrivateRangeBlocking != "enforced" {
		return errors.New("dependency registry private_range_blocking must be enforced")
	}
	if req.RegistryIdentity.DNSRebindingProtection != "enforced" {
		return errors.New("dependency registry dns_rebinding_protection must be enforced")
	}
	if lease == nil {
		return errors.New("dependency registry auth lease is required")
	}
	if lease.Posture() != dependencyRegistryAuthPosturePublicNoAuth {
		return errors.New("public registry fetcher requires public_no_auth posture")
	}
	return nil
}

func successfulRegistryHTTPResponse(httpResp *http.Response) (io.ReadCloser, dependencyRegistryFetchMetadata, error) {
	if httpResp.StatusCode >= 300 && httpResp.StatusCode < 400 {
		location := strings.TrimSpace(httpResp.Header.Get("Location"))
		_ = httpResp.Body.Close()
		return nil, dependencyRegistryFetchMetadata{}, fmt.Errorf("dependency registry redirects are denied (location=%q)", location)
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(httpResp.Body, 1024))
		_ = httpResp.Body.Close()
		return nil, dependencyRegistryFetchMetadata{}, fmt.Errorf("dependency registry request failed with status %d: %s", httpResp.StatusCode, strings.TrimSpace(string(body)))
	}
	return httpResp.Body, dependencyRegistryFetchMetadata{ContentType: strings.TrimSpace(httpResp.Header.Get("Content-Type"))}, nil
}

func (f publicRegistryHTTPFetcher) resolverOrDefault() hostResolver {
	if f.resolver != nil {
		return f.resolver
	}
	return net.DefaultResolver
}

func (f publicRegistryHTTPFetcher) clientOrDefault() dependencyFetchHTTPDoer {
	if f.client != nil {
		return f.client
	}
	return newPublicRegistryHTTPFetcher().client
}

func enforceResolvedPublicIPs(ctx context.Context, resolver hostResolver, host string) error {
	ips, err := lookupValidatedPublicIPs(ctx, resolver, host)
	if err != nil {
		return err
	}
	_ = ips
	return nil
}

func dependencyFetchPinnedDialContext(resolver hostResolver) func(context.Context, string, string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, err := lookupValidatedPublicIPs(ctx, resolver, host)
		if err != nil {
			return nil, err
		}
		return dialDependencyFetchResolvedIPs(ctx, dialer, network, port, host, ips)
	}
}

func dialDependencyFetchResolvedIPs(ctx context.Context, dialer *net.Dialer, network, port, host string, ips []net.IP) (net.Conn, error) {
	var lastErr error
	for _, ip := range ips {
		conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if dialErr == nil {
			return conn, nil
		}
		lastErr = dialErr
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("dependency registry dns resolution empty for %q", host)
	}
	return nil, lastErr
}

func lookupValidatedPublicIPs(ctx context.Context, resolver hostResolver, host string) ([]net.IP, error) {
	lookupCtx, cancel := context.WithTimeout(ctx, gatewayDNSLookupTimeout)
	defer cancel()
	ips, err := resolver.LookupIP(lookupCtx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("dependency registry dns resolution failed for %q: %w", host, err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("dependency registry dns resolution empty for %q", host)
	}
	for _, ip := range ips {
		if isDeniedRuntimeIP(ip) {
			return nil, fmt.Errorf("dependency registry dns rebind/private ip blocked for %q (%s)", host, ip.String())
		}
	}
	return ips, nil
}

func dependencyFetchPathForRequest(req DependencyFetchRequestObject) (string, error) {
	base := normalizeDestinationPathPrefix(req.RegistryIdentity.CanonicalPathPrefix)
	ecosystem := strings.TrimSpace(strings.ToLower(req.Ecosystem))
	packageName := strings.TrimSpace(req.PackageName)
	version := strings.TrimSpace(req.PackageVersion)
	if packageName == "" || version == "" {
		return "", errors.New("dependency package_name and package_version are required")
	}
	var segment string
	switch ecosystem {
	case "npm":
		segment = path.Join(packageName, "-/", packageName+"-"+version+".tgz")
	default:
		return "", fmt.Errorf("dependency ecosystem %q is not yet supported", ecosystem)
	}
	full := path.Clean(path.Join(base, segment))
	if !strings.HasPrefix(full, base) {
		return "", errors.New("dependency request path escapes canonical_path_prefix")
	}
	if !strings.HasPrefix(full, "/") {
		full = "/" + full
	}
	return full, nil
}

type publicRegistryNoAuthSource struct{}

func (publicRegistryNoAuthSource) AcquireLease(_ context.Context, _ DependencyFetchRequestObject) (dependencyRegistryAuthLease, error) {
	return publicRegistryNoAuthLease{}, nil
}

type publicRegistryNoAuthLease struct{}

func (publicRegistryNoAuthLease) Posture() dependencyRegistryAuthPosture {
	return dependencyRegistryAuthPosturePublicNoAuth
}

func (publicRegistryNoAuthLease) LeaseID() string      { return "" }
func (publicRegistryNoAuthLease) ExpiresAt() time.Time { return time.Time{} }
