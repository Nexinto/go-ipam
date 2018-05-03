package ipam

import (
	"fmt"
	"net"
	"strings"

	log "github.com/sirupsen/logrus"

	ccidr "github.com/apparentlymart/go-cidr/cidr"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConfigMap struct {
	Kube kubernetes.Interface
	IpamData
}

const (
	MapName = "ipam-cm"
)

func NewConfigMapIpam(kube kubernetes.Interface, network string) (*ConfigMap, error) {

	_, nn, err := net.ParseCIDR(network)
	if err != nil {
		return &ConfigMap{}, err
	}

	c := &ConfigMap{Kube: kube, IpamData: IpamData{Network: nn}}

	_, err = c.Kube.CoreV1().ConfigMaps("kube-system").Get(MapName, metav1.GetOptions{})
	if err == nil {
		return c, nil
	}

	if errors.IsNotFound(err) {
		_, err = c.Kube.CoreV1().ConfigMaps("kube-system").Create(&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: MapName}, Data: map[string]string{}})
		if err != nil {
			return &ConfigMap{}, fmt.Errorf("error creating initial address management configmap '%s' in namespace '%s': %s", MapName, "kube-system", err.Error())
		} else {
			return c, nil
		}
	} else {
		return &ConfigMap{}, fmt.Errorf("error lookuping up address management configmap '%s' in namespace '%s': %s", MapName, "kube-system", err.Error())
	}
}

func (c *ConfigMap) Reset() error {
	_, err := c.Kube.CoreV1().ConfigMaps("kube-system").Get(MapName, metav1.GetOptions{})
	if err == nil {
		_ = c.Kube.CoreV1().ConfigMaps("kube-system").Delete(MapName, &metav1.DeleteOptions{})
	}

	cm := corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: MapName}, Data: map[string]string{}}

	_, err = c.Kube.CoreV1().ConfigMaps("kube-system").Create(&cm)
	if err != nil {
		return fmt.Errorf("error creating configmap '%s' in namespace '%s': %s", "kube-system", MapName, err.Error())
	}

	return nil
}

func (c *ConfigMap) Assign(description string) (string, error) {
	cm, err := c.Kube.CoreV1().ConfigMaps("kube-system").Get(MapName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("error fetching configmap '%s' in namespace '%s': %s", "kube-system", MapName, err.Error())
	}

	if cm.Data == nil {
		cm.Data = map[string]string{}
	}

	first, last := ccidr.AddressRange(c.Network)
	this := first

	for {
		if cm.Data[this.String()] != "" {
			this = ccidr.Inc(this)
		} else {
			ip := this.String()
			cm.Data[ip] = description

			_, err := c.Kube.CoreV1().ConfigMaps("kube-system").Update(cm)
			if err != nil {
				return "", fmt.Errorf("error updating configmap '%s' in namespace '%s': %s", "kube-system", MapName, err.Error())
			}

			log.Infof("assigned %s (%s)", ip, description)
			return ip, nil
		}

		if this.Equal(ccidr.Inc(last)) {
			return "", fmt.Errorf("no free addresses available")
		}
	}
}

func (c *ConfigMap) IsAssigned(ip string) (bool, error) {
	cm, err := c.Kube.CoreV1().ConfigMaps("kube-system").Get(MapName, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("error fetching configmap '%s' in namespace '%s': %s", "kube-system", MapName, err.Error())
	}

	return cm.Data[ip] != "", nil
}

func (c *ConfigMap) Unassign(ip string) error {
	cm, err := c.Kube.CoreV1().ConfigMaps("kube-system").Get(MapName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error fetching configmap '%s' in namespace '%s': %s", "kube-system", MapName, err.Error())
	}

	delete(cm.Data, ip)

	_, err = c.Kube.CoreV1().ConfigMaps("kube-system").Update(cm)
	if err != nil {
		return fmt.Errorf("error updating configmap '%s' in namespace '%s': %s", "kube-system", MapName, err.Error())
	}

	return nil
}

func (c *ConfigMap) Get(ip string) (string, error) {
	cm, err := c.Kube.CoreV1().ConfigMaps("kube-system").Get(MapName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("error fetching configmap '%s' in namespace '%s': %s", "kube-system", MapName, err.Error())
	}

	if cm.Data[ip] == "" {
		return "", fmt.Errorf("%s is not assigned", ip)
	}

	return cm.Data[ip], nil
}

func (c *ConfigMap) Cleanup(keep []string) error {
	cm, err := c.Kube.CoreV1().ConfigMaps("kube-system").Get(MapName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error fetching configmap '%s' in namespace '%s': %s", "kube-system", MapName, err.Error())
	}

	k := map[string]bool{}
	for _, ip := range keep {
		k[ip] = true
	}

	for ip := range cm.Data {
		if !k[ip] {
			delete(cm.Data, ip)
		}
	}

	_, err = c.Kube.CoreV1().ConfigMaps("kube-system").Update(cm)
	if err != nil {
		return fmt.Errorf("error updating configmap '%s' in namespace '%s': %s", "kube-system", MapName, err.Error())
	}

	return nil
}

func (c *ConfigMap) Search(search string, exact bool) ([]string, error) {
	cm, err := c.Kube.CoreV1().ConfigMaps("kube-system").Get(MapName, metav1.GetOptions{})
	if err != nil {
		return []string{}, fmt.Errorf("error fetching configmap '%s' in namespace '%s': %s", "kube-system", MapName, err.Error())
	}

	var found []string

	for ip, desc := range cm.Data {
		if exact && search == desc || !exact && strings.Contains(desc, search) {
			found = append(found, ip)
		}
	}

	return found, nil
}

func (c *ConfigMap) List() ([]string, error) {
	cm, err := c.Kube.CoreV1().ConfigMaps("kube-system").Get(MapName, metav1.GetOptions{})
	if err != nil {
		return []string{}, fmt.Errorf("error fetching configmap '%s' in namespace '%s': %s", "kube-system", MapName, err.Error())
	}

	var found []string

	for ip := range cm.Data {
		found = append(found, ip)
	}

	return found, nil
}

func (c *ConfigMap) Set(ip string, description string) error {
	cm, err := c.Kube.CoreV1().ConfigMaps("kube-system").Get(MapName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error fetching configmap '%s' in namespace '%s': %s", "kube-system", MapName, err.Error())
	}

	if cm.Data[ip] != "" && cm.Data[ip] != description {
		return fmt.Errorf("address %s is already assigned (%s)", ip, cm.Data[ip])
	}
	cm.Data[ip] = description

	_, err = c.Kube.CoreV1().ConfigMaps("kube-system").Update(cm)
	if err != nil {
		return fmt.Errorf("error updating configmap '%s' in namespace '%s': %s", "kube-system", MapName, err.Error())
	}

	return nil
}

func (c *ConfigMap) String() string {
	return "default ConfigMap"
}
