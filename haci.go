package ipam

import (
	"fmt"
	"net"

	log "github.com/sirupsen/logrus"

	"github.com/Nexinto/go-haci-client/haci"
)

type HaCi struct {
	HaCi haci.Client

	// Address assignments are tagged with this so we do not touch anything else
	Tag string
	IpamData
}

func NewHaciIpamWithClient(client haci.Client, network, tag string) (*HaCi, error) {
	_, nn, err := net.ParseCIDR(network)
	if err != nil {
		return nil, fmt.Errorf("could not parse network '%s': %s", network, err.Error())
	}

	return &HaCi{HaCi: client, Tag: tag, IpamData: IpamData{Network: nn}}, nil

}

func NewHaciIpam(network, url, username, password, root, tag string) (*HaCi, error) {
	h, err := haci.NewWebClient(url, username, password, root)
	if err != nil {
		return nil, fmt.Errorf("could not create HaCi client: %s", err.Error())
	}

	return NewHaciIpamWithClient(h, network, tag)
}

func (c *HaCi) Reset() error {
	return c.HaCi.Reset()
}

func (c *HaCi) Assign(description string) (string, error) {
	n, err := c.HaCi.Assign(c.Network.String(), description, 32, []string{c.Tag})
	if err != nil {
		return "", fmt.Errorf("could not assign new network in %s: %s", c.Network.String(), err.Error())
	}

	ip, err := n.IP()
	if err != nil {
		return "", fmt.Errorf("could not parse result %s from haci: %s", n, err.Error())
	}

	return ip, nil

}

func (c *HaCi) IsAssigned(ip string) (bool, error) {
	_, err := c.HaCi.Get(ip + "/32")
	return err == nil, nil
}

func (c *HaCi) Unassign(ip string) error {
	n, err := c.HaCi.Get(ip + "/32")
	if err != nil {
		return fmt.Errorf("could not get %s from HaCi (to check before deleting): %s", ip, err.Error())
	}

	if len(n.Tags) != 1 || n.Tags[0] != c.Tag {
		return fmt.Errorf("will not unassign %s in HaCi as it is not managed by us", ip)
	}

	return c.HaCi.Delete(ip + "/32")
}

func (c *HaCi) Get(ip string) (string, error) {
	n, err := c.HaCi.Get(ip + "/32")
	if err != nil {
		return "", fmt.Errorf("could not get %s from HaCi: %s", ip, err.Error())
	}

	return n.Description, nil
}

func (c *HaCi) Cleanup(keep []string) error {
	networks, err := c.HaCi.List(c.Network.String())
	if err != nil {
		return fmt.Errorf("error listing supernet %s in HaCi: %s", c.Network.String(), err.Error())
	}

	k := map[string]bool{}
	for _, ip := range keep {
		k[ip] = true
	}

	for _, n := range networks {
		ip, err := n.IP()
		if err != nil {
			return fmt.Errorf("while cleaning up %s in HaCi, a network address could not be parsed (%v): %s", c.Network.String(), n, err.Error())
		}

		if len(n.Tags) != 1 || n.Tags[0] != c.Tag {
			log.Debugf("Ignoring %s as it does not have our tag", n.Network)
			continue
		}

		if !k[ip] {
			log.Infof("Deleting %s from HaCi", n.Network)
			c.HaCi.Delete(n.Network)
		}
	}

	return nil
}

func (c *HaCi) Search(search string, exact bool) ([]string, error) {
	nn, err := c.HaCi.Search(search, exact)
	if err != nil {
		return []string{}, fmt.Errorf("could not search for '%s' in HaCi: %s", search, err.Error())
	}

	found := []string{}

	for _, n := range nn {
		ip, _ := n.IP()
		found = append(found, ip)
	}

	return found, nil
}

func (c *HaCi) List() ([]string, error) {
	nn, err := c.HaCi.List(c.Network.String())
	if err != nil {
		return []string{}, fmt.Errorf("could list supernet '%s' in HaCi: %s", c.Network.String(), err.Error())
	}

	var found []string

	for _, n := range nn {
		ip, _ := n.IP()
		found = append(found, ip)
	}

	return found, nil
}

func (c *HaCi) Set(ip string, description string) error {
	desc, err := c.Get(ip)
	if err == nil { // found
		if desc != description {
			return fmt.Errorf("address %s is already assigned to %s", ip, desc)
		} else {
			return nil
		}
	}

	err = c.HaCi.Add(ip+"/32", description, []string{c.Tag})
	if err != nil {
		return fmt.Errorf("error setting address %s: %s", ip, err.Error())
	}

	return nil
}

func (c *HaCi) String() string {
	return fmt.Sprintf("%s tag %s network %s", c.HaCi.String(), c.Tag, c.Network)
}
