package repo

import (
	"github.com/google/go-containerregistry/pkg/authn"
)

type kubefoxKeychain struct {
	defaultKeychain authn.Keychain
	registry        string
	authToken       string
}

// Resolve tries to resolve the target using the default keychain. If the target
// is found the corresponding authenticator is returned. Otherwise, if the
// target is the KubeFox System registry an Authenticator is returned using the
// GitHub token.
func (kc *kubefoxKeychain) Resolve(target authn.Resource) (authn.Authenticator, error) {
	a, err := kc.defaultKeychain.Resolve(target)
	if err != nil {
		return nil, err
	}

	authz, err := a.Authorization()
	if target.RegistryStr() == kc.registry && (authn.AuthConfig{}) == *authz {
		return authn.FromConfig(authn.AuthConfig{
			Auth: kc.authToken,
		}), nil
	}

	return a, err
}
