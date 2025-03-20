package authprovider

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	authutil "github.com/containerd/containerd/v2/core/remotes/docker/auth"
	remoteserrors "github.com/containerd/containerd/v2/core/remotes/errors"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/types"
	http "github.com/hashicorp/go-cleanhttp"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth"
	"github.com/moby/buildkit/util/progress/progresswriter"
	"github.com/moby/buildkit/util/tracing"
	"github.com/pkg/errors"
	"golang.org/x/crypto/nacl/sign"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultExpiration      = 60
	dockerHubConfigfileKey = "https://index.docker.io/v1/"
	dockerHubRegistryHost  = "registry-1.docker.io"
)

type DockerAuthProviderConfig struct {
	// ConfigFile is the docker config file
	ConfigFile *configfile.ConfigFile
	// TLSConfigs is a map of host to TLS config
	TLSConfigs map[string]*AuthTLSConfig
	// ExpireCachedAuth is a function that returns true auth config should be refreshed
	// instead of using a pre-cached result.
	// If nil then the cached result will expire after 10 minutes.
	// The function is called with the time the cached auth config was created
	// and the server URL the auth config is for.
	ExpireCachedAuth func(created time.Time, serverURL string) bool
}

type authConfigCacheEntry struct {
	Created time.Time
	Auth    *types.AuthConfig
}

func NewDockerAuthProvider(cfg DockerAuthProviderConfig) session.Attachable {
	if cfg.ExpireCachedAuth == nil {
		cfg.ExpireCachedAuth = func(created time.Time, _ string) bool {
			return time.Since(created) > 10*time.Minute
		}
	}
	return &authProvider{
		authConfigCache: map[string]authConfigCacheEntry{},
		expireAc:        cfg.ExpireCachedAuth,
		config:          cfg.ConfigFile,
		seeds:           &tokenSeeds{dir: config.Dir()},
		loggerCache:     map[string]struct{}{},
		tlsConfigs:      cfg.TLSConfigs,
	}
}

type authProvider struct {
	authConfigCache map[string]authConfigCacheEntry
	expireAc        func(time.Time, string) bool
	config          *configfile.ConfigFile
	seeds           *tokenSeeds
	logger          progresswriter.Logger
	loggerCache     map[string]struct{}
	tlsConfigs      map[string]*AuthTLSConfig

	// The need for this mutex is not well understood.
	// Without it, the docker cli on OS X hangs when
	// reading credentials from docker-credential-osxkeychain.
	// See issue https://github.com/docker/cli/issues/1862
	mu sync.Mutex
}

func (ap *authProvider) SetLogger(l progresswriter.Logger) {
	ap.mu.Lock()
	ap.logger = l
	ap.mu.Unlock()
}

func (ap *authProvider) Register(server *grpc.Server) {
	auth.RegisterAuthServer(server, ap)
}

func (ap *authProvider) FetchToken(ctx context.Context, req *auth.FetchTokenRequest) (rr *auth.FetchTokenResponse, err error) {
	ac, err := ap.getAuthConfig(ctx, req.Host)
	if err != nil {
		return nil, err
	}

	// check for statically configured bearer token
	if ac.RegistryToken != "" {
		return toTokenResponse(ac.RegistryToken, time.Time{}, 0), nil
	}

	creds, err := ap.credentials(ctx, req.Host)
	if err != nil {
		return nil, err
	}

	to := authutil.TokenOptions{
		Realm:    req.Realm,
		Service:  req.Service,
		Scopes:   req.Scopes,
		Username: creds.Username,
		Secret:   creds.Secret,
	}

	httpClient := tracing.DefaultClient
	if tc, err := ap.tlsConfig(req.Host); err == nil && tc != nil {
		transport := http.DefaultTransport()
		transport.TLSClientConfig = tc
		httpClient.Transport = tracing.NewTransport(transport)
	}

	if creds.Secret != "" {
		done := func(progresswriter.SubLogger) error {
			return err
		}
		defer func() {
			err = errors.Wrap(err, "failed to fetch oauth token")
		}()
		ap.mu.Lock()
		name := fmt.Sprintf("[auth] %v token for %s", strings.Join(trimScopePrefix(req.Scopes), " "), req.Host)
		if _, ok := ap.loggerCache[name]; !ok {
			progresswriter.Wrap(name, ap.logger, done)
		}
		ap.mu.Unlock()
		// credential information is provided, use oauth POST endpoint
		resp, err := authutil.FetchTokenWithOAuth(ctx, httpClient, nil, "buildkit-client", to)
		if err != nil {
			var errStatus remoteserrors.ErrUnexpectedStatus
			if errors.As(err, &errStatus) {
				// Registries without support for POST may return 404 for POST /v2/token.
				// As of September 2017, GCR is known to return 404.
				// As of February 2018, JFrog Artifactory is known to return 401.
				if (errStatus.StatusCode == 405 && to.Username != "") || errStatus.StatusCode == 404 || errStatus.StatusCode == 401 {
					resp, err := authutil.FetchToken(ctx, httpClient, nil, to)
					if err != nil {
						return nil, err
					}
					return toTokenResponse(resp.Token, resp.IssuedAt, resp.ExpiresInSeconds), nil
				}
			}
			return nil, err
		}
		return toTokenResponse(resp.AccessToken, resp.IssuedAt, resp.ExpiresInSeconds), nil
	}
	// do request anonymously
	resp, err := authutil.FetchToken(ctx, httpClient, nil, to)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch anonymous token")
	}
	return toTokenResponse(resp.Token, resp.IssuedAt, resp.ExpiresInSeconds), nil
}

func (ap *authProvider) tlsConfig(host string) (*tls.Config, error) {
	if ap.tlsConfigs == nil {
		return nil, nil
	}
	c, ok := ap.tlsConfigs[host]
	if !ok {
		return nil, nil
	}
	tc := &tls.Config{}
	if len(c.RootCAs) > 0 {
		systemPool, err := x509.SystemCertPool()
		if err != nil {
			if runtime.GOOS == "windows" {
				systemPool = x509.NewCertPool()
			} else {
				return nil, errors.Wrapf(err, "unable to get system cert pool")
			}
		}
		tc.RootCAs = systemPool
	}

	for _, p := range c.RootCAs {
		dt, err := os.ReadFile(p)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read %s", p)
		}
		tc.RootCAs.AppendCertsFromPEM(dt)
	}

	for _, kp := range c.KeyPairs {
		cert, err := tls.LoadX509KeyPair(kp.Certificate, kp.Key)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to load keypair for %s", kp.Certificate)
		}
		tc.Certificates = append(tc.Certificates, cert)
	}
	if c.Insecure {
		tc.InsecureSkipVerify = true
	}
	return tc, nil
}

func (ap *authProvider) credentials(ctx context.Context, host string) (*auth.CredentialsResponse, error) {
	ac, err := ap.getAuthConfig(ctx, host)
	if err != nil {
		return nil, err
	}
	res := &auth.CredentialsResponse{}
	if ac.IdentityToken != "" {
		res.Secret = ac.IdentityToken
	} else {
		res.Username = ac.Username
		res.Secret = ac.Password
	}
	return res, nil
}

func (ap *authProvider) Credentials(ctx context.Context, req *auth.CredentialsRequest) (*auth.CredentialsResponse, error) {
	resp, err := ap.credentials(ctx, req.Host)
	if err != nil || resp.Secret != "" {
		ap.mu.Lock()
		defer ap.mu.Unlock()
		_, ok := ap.loggerCache[req.Host]
		ap.loggerCache[req.Host] = struct{}{}
		if !ok && ap.logger != nil {
			return resp, progresswriter.Wrap(fmt.Sprintf("[auth] sharing credentials for %s", req.Host), ap.logger, func(progresswriter.SubLogger) error {
				return err
			})
		}
	}
	return resp, err
}

func (ap *authProvider) GetTokenAuthority(ctx context.Context, req *auth.GetTokenAuthorityRequest) (*auth.GetTokenAuthorityResponse, error) {
	key, err := ap.getAuthorityKey(ctx, req.Host, req.Salt)
	if err != nil {
		return nil, err
	}

	return &auth.GetTokenAuthorityResponse{PublicKey: key[32:]}, nil
}

func (ap *authProvider) VerifyTokenAuthority(ctx context.Context, req *auth.VerifyTokenAuthorityRequest) (*auth.VerifyTokenAuthorityResponse, error) {
	key, err := ap.getAuthorityKey(ctx, req.Host, req.Salt)
	if err != nil {
		return nil, err
	}

	priv := new([64]byte)
	copy((*priv)[:], key)

	return &auth.VerifyTokenAuthorityResponse{Signed: sign.Sign(nil, req.Payload, priv)}, nil
}

func (ap *authProvider) getAuthConfig(ctx context.Context, host string) (*types.AuthConfig, error) {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	if host == dockerHubRegistryHost {
		host = dockerHubConfigfileKey
	}

	entry, exists := ap.authConfigCache[host]
	if exists && !ap.expireAc(entry.Created, host) {
		return entry.Auth, nil
	}

	span, _ := tracing.StartSpan(ctx, fmt.Sprintf("load credentials for %s", host))
	ac, err := ap.config.GetAuthConfig(host)
	tracing.FinishWithError(span, err)
	if err != nil {
		return nil, err
	}
	entry = authConfigCacheEntry{
		Created: time.Now(),
		Auth:    &ac,
	}

	ap.authConfigCache[host] = entry

	return entry.Auth, nil
}

func (ap *authProvider) getAuthorityKey(ctx context.Context, host string, salt []byte) (ed25519.PrivateKey, error) {
	if v, err := strconv.ParseBool(os.Getenv("BUILDKIT_NO_CLIENT_TOKEN")); err == nil && v {
		return nil, status.Errorf(codes.Unavailable, "client side tokens disabled")
	}

	creds, err := ap.credentials(ctx, host)
	if err != nil {
		return nil, err
	}
	seed, err := ap.seeds.getSeed(host)
	if err != nil {
		return nil, err
	}

	mac := hmac.New(sha256.New, salt)
	if creds.Secret != "" {
		mac.Write(seed)
	}

	sum := mac.Sum(nil)

	return ed25519.NewKeyFromSeed(sum[:ed25519.SeedSize]), nil
}

func toTokenResponse(token string, issuedAt time.Time, expires int) *auth.FetchTokenResponse {
	if expires == 0 {
		expires = defaultExpiration
	}
	resp := &auth.FetchTokenResponse{
		Token:     token,
		ExpiresIn: int64(expires),
	}
	if !issuedAt.IsZero() {
		resp.IssuedAt = issuedAt.Unix()
	}
	return resp
}

func trimScopePrefix(scopes []string) []string {
	out := make([]string, len(scopes))
	for i, s := range scopes {
		out[i] = strings.TrimPrefix(s, "repository:")
	}
	return out
}
