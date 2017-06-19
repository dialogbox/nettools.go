package pinger

import (
	"net"
	"errors"
)

func lookupIP(host string) (net.IP, error) {
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}

	for _, ip := range ips {
		if ip.To4() != nil {
			return ip.To4(), nil
		}
	}

	return nil, errors.New("no A or AAAA record or only have IPv6 IP")
}

func resolveHosts(hosts []string) []net.IP {
	iplist := make([]net.IP, len(hosts))
	for i := range hosts {
		ip, err := lookupIP(hosts[i])
		if err != nil {
			panic(err)
		}
		iplist[i] = ip
	}

	return iplist
}

