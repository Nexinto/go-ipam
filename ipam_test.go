package ipam

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	log "github.com/sirupsen/logrus"

	"github.com/Nexinto/go-haci-client/haci"

	"k8s.io/client-go/kubernetes/fake"
)

type IpamTestSuite struct {
	suite.Suite
	Ipam Ipam
}

func (s *IpamTestSuite) TestAssignAndGet() {
	ip, err := s.Ipam.Assign("myservice.com")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "10.0.0.0", ip)
	desc, err := s.Ipam.Get(ip)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "myservice.com", desc)
}

func (s *IpamTestSuite) TestOutOfAddresses() {
	for i := 0; i <= 15; i++ {
		_, err := s.Ipam.Assign("x")
		assert.Nil(s.T(), err)
	}
	_, err := s.Ipam.Assign("x")
	assert.Error(s.T(), err)
}

func (s *IpamTestSuite) TestUnassign() {
	ip, err := s.Ipam.Assign("x")
	assert.Nil(s.T(), err)
	ia, err := s.Ipam.IsAssigned(ip)
	assert.True(s.T(), ia)
	assert.Nil(s.T(), err)
	err = s.Ipam.Unassign(ip)
	assert.Nil(s.T(), err)
	ia, err = s.Ipam.IsAssigned(ip)
	assert.False(s.T(), ia)
	assert.Nil(s.T(), err)
}

func (s *IpamTestSuite) TestCleanup() {
	keepme := []string{"10.0.0.1", "10.0.0.4", "10.0.0.9"}
	for i := 0; i <= 15; i++ {
		_, err := s.Ipam.Assign("x")
		assert.Nil(s.T(), err)
	}

	err := s.Ipam.Cleanup(keepme)
	assert.Nil(s.T(), err)

	for i := 0; i <= 15; i++ {
		ip := fmt.Sprintf("10.0.0.%d", i)
		d, err := s.Ipam.Get(ip)
		if i == 1 || i == 4 || i == 9 {
			assert.Nil(s.T(), err, ip+" should still be assigned")
			assert.NotEmpty(s.T(), d, ip+" should still be assigned")
		} else {
			assert.Error(s.T(), err, ip+" should not be assigned")
		}
	}
}

func (s *IpamTestSuite) TestSearch() {
	_, err := s.Ipam.Assign("myservice")
	assert.Nil(s.T(), err)
	i2, err := s.Ipam.Assign("secondservice")
	assert.Nil(s.T(), err)
	_, err = s.Ipam.Assign("thirdservice")
	assert.Nil(s.T(), err)
	results, err := s.Ipam.Search("secondservice", true)
	assert.Nil(s.T(), err)
	if assert.Equal(s.T(), 1, len(results)) {
		assert.Equal(s.T(), i2, results[0])
	}
	results, err = s.Ipam.Search("secondservice", false)
	assert.Nil(s.T(), err)
	if assert.Equal(s.T(), 1, len(results)) {
		assert.Equal(s.T(), i2, results[0])
	}
	results, err = s.Ipam.Search("second", true)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 0, len(results))
	results, err = s.Ipam.Search("second", false)
	assert.Nil(s.T(), err)
	if assert.Equal(s.T(), 1, len(results)) {
		assert.Equal(s.T(), i2, results[0])
	}
}

func (s *IpamTestSuite) SetupTest() {
	_ = s.Ipam.Reset()
}

func TestFakeIpam(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	c := NewFakeIpam("10.0.0.0/28")
	suite.Run(t, &IpamTestSuite{Ipam: c})
}

func TestHaCiIpam(t *testing.T) {
	c, _ := NewHaciIpamWithClient(haci.NewFakeClientUsesFirst(), "10.0.0.0/28", "testing")
	suite.Run(t, &IpamTestSuite{Ipam: c})
}

func TestConfigMapIpam(t *testing.T) {
	c, _ := NewConfigMapIpam(fake.NewSimpleClientset(), "10.0.0.0/28")
	suite.Run(t, &IpamTestSuite{Ipam: c})
}
