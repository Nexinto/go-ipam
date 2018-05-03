package ipam

import (
	"net"
	"os"

	log "github.com/sirupsen/logrus"

	"k8s.io/client-go/kubernetes"
)

type IpamData struct {
	Network *net.IPNet
}

type Ipam interface {
	String() string
	Reset() error
	Assign(description string) (string, error)
	IsAssigned(ip string) (bool, error)
	Unassign(ip string) error
	Get(ip string) (string, error)
	Cleanup(keep []string) error
	Search(search string, exact bool) ([]string, error)
	List() ([]string, error)
	Set(ip string, description string) error
}

func InitFromEnvironment(kube kubernetes.Interface, network, tag string) (i Ipam, err error) {
	if os.Getenv("HACI_URL") != "" {
		i, err = NewHaciIpam(network, os.Getenv("HACI_URL"), os.Getenv("HACI_USERNAME"), os.Getenv("HACI_PASSWORD"), os.Getenv("HACI_ROOT"), tag)
		log.Debugf("Using HaCi ipam at %s with root %s and network %s", os.Getenv("HACI_URL"), os.Getenv("HACI_ROOT"), network)
	} else {
		i, err = NewConfigMapIpam(kube, network)
		log.Debugf("Using ConfigMap ipam with network %s", network)
	}

	return
}
