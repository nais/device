package wireguard

const (
	PrometheusPeerName = "prometheus"
	APIServerPeerName  = "apiserver"
)

type Peer interface {
	GetName() string
	GetPublicKey() string
	GetAllowedIPs() []string
	GetEndpoint() string
}

func CastPeerList[T Peer](peers []T) []Peer {
	result := []Peer{}
	for _, peer := range peers {
		result = append(result, peer)
	}
	return result
}
