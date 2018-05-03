package ipam

import (
	"fmt"
	"net"
	"strings"

	log "github.com/sirupsen/logrus"

	ccidr "github.com/apparentlymart/go-cidr/cidr"
)

type Fake struct {
	Assigned map[string]string
	IpamData
}

func NewFakeIpam(network string) *Fake {
	_, nn, err := net.ParseCIDR(network)
	if err != nil {
		panic(err)
	}

	return &Fake{Assigned: map[string]string{}, IpamData: IpamData{Network: nn}}
}

func (c *Fake) Reset() error {
	c.Assigned = map[string]string{}
	return nil
}

func (c *Fake) Assign(description string) (string, error) {
	first, last := ccidr.AddressRange(c.Network)
	this := first

	for {
		if c.Assigned[this.String()] != "" {
			this = ccidr.Inc(this)
		} else {
			ip := this.String()
			c.Assigned[ip] = description
			log.Infof("assigned %s", ip)
			return ip, nil
		}

		if this.Equal(ccidr.Inc(last)) {
			return "", fmt.Errorf("no free addresses available")
		}
	}
}

func (c *Fake) IsAssigned(ip string) (bool, error) {
	return c.Assigned[ip] != "", nil
}

func (c *Fake) Unassign(ip string) error {
	delete(c.Assigned, ip)
	return nil
}

func (c *Fake) Get(ip string) (string, error) {
	a := c.Assigned[ip]
	if a != "" {
		return c.Assigned[ip], nil
	} else {
		return "", fmt.Errorf("%s not found", ip)
	}
}

func (c *Fake) Cleanup(keep []string) error {
	k := map[string]bool{}
	for _, ip := range keep {
		k[ip] = true
	}

	for ip := range c.Assigned {
		if !k[ip] {
			delete(c.Assigned, ip)
		}
	}

	return nil
}

func (c *Fake) Search(search string, exact bool) ([]string, error) {
	found := []string{}

	for ip, desc := range c.Assigned {
		if exact && search == desc || !exact && strings.Contains(desc, search) {
			found = append(found, ip)
		}
	}

	return found, nil
}

func (c *Fake) List() ([]string, error) {
	found := []string{}

	for ip, _ := range c.Assigned {
		found = append(found, ip)
	}

	return found, nil
}

func (c *Fake) Set(ip string, description string) error {
	if c.Assigned[ip] != "" && c.Assigned[ip] != description {
		return fmt.Errorf("address %s is already assigned (%s)", ip, c.Assigned[ip])
	}
	c.Assigned[ip] = description
	return nil
}

func (c *Fake) String() string {
	return "Fake IPAM"
}
