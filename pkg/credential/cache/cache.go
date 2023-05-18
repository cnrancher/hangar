package cache

import "github.com/cnrancher/hangar/pkg/utils"

type CredentialCache struct {
	Data []struct {
		Username string
		Password string
		Registry string
	}
}

var credentialCache = CredentialCache{
	Data: make([]struct {
		Username string
		Password string
		Registry string
	}, 0),
}

func Add(u, p, r string) {
	if r == "" {
		r = utils.DockerHubRegistry
	}
	credentialCache.Data = append(credentialCache.Data, struct {
		Username string
		Password string
		Registry string
	}{
		Username: u,
		Password: p,
		Registry: r,
	})
}

func Cached(u, p, r string) bool {
	if r == "" {
		r = utils.DockerHubRegistry
	}
	for _, v := range credentialCache.Data {
		if v.Password == p && v.Username == u && v.Registry == r {
			return true
		}
	}
	return false
}

func Get(r string) (user, passwd string) {
	if r == "" {
		r = utils.DockerHubRegistry
	}
	for _, v := range credentialCache.Data {
		if v.Registry == r {
			return v.Username, v.Password
		}
	}
	return "", ""
}
