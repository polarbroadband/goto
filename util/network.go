package util

import (
	"strconv"
	"strings"
	"time"
)

/* ****************************************
ip address functions
**************************************** */

// IP holds IPv4 and IPv6 data structure and provides operations on it
type IP struct {
	V6   bool // IPv4 - false, IPv6 - true
	Addr string
	Mask int
}

// StringToIP converts x.x.x.x/24 or f8ae:12::1/128 to IP obj, default mask is 32 or 128
func StringToIP(s string) *IP {
	var ip IP
	var err error
	if strings.Contains(s, ":") {
		ip.V6 = true
	} else if !strings.Contains(s, ".") {
		return nil
	}
	sst := strings.Split(s, "/")
	switch len(sst) {
	case 1:
		ip.Addr = sst[0]
		if ip.V6 {
			ip.Mask = 128
		} else {
			ip.Mask = 32
		}
	case 2:
		ip.Addr = sst[0]
		ip.Mask, err = strconv.Atoi(sst[1])
		if err != nil {
			return nil
		}
		// more strict check add here
	default:
		return nil
	}
	return &ip
}

// ListToIps converts a slice of IP address string to a IP obj slice
func ListToIps(l []string) (i []*IP) {
	for _, ip := range l {
		i = append(i, StringToIP(ip))
	}
	return
}

// String converts IP to a string like x.x.x.x/32
func (ip *IP) String() string {
	return ip.Addr + "/" + strconv.Itoa(ip.Mask)
}

// SameIP returns true if two IP have the same address and mask
func (ip *IP) SameIP(t *IP) bool {
	if ip.Addr == t.Addr && ip.Mask == t.Mask {
		return true
	}
	return false
}

/* ****************************************
Protocol structure
**************************************** */

// BGPRecvdRoutes holds BGP prefixes advertised to peer
type BGPRecvdRoutes struct {
	Peer     string // neighbor ip
	Local    string // local ip
	Type     string // Internal/External
	Prefixes map[string]BGPAttributes
}

// BGPAttributes holds properties of a BGP prefix
type BGPAttributes struct {
	Prefix    string
	Nexthop   string
	LocalPref int64
	MED       int64
	ASPath    []string
	Community []string
	Flags     map[string]bool
	Tag       string
	Age       time.Duration
}
