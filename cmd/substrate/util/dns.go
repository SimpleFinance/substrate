package util

import (
	"fmt"
	"net"
	"strings"

	"github.com/miekg/dns"
)

// LookupNSUsingServer looks up NS records for `target` using the DNS server `server`
func LookupNSUsingServer(target string, server string) ([]string, error) {

	// create a DNS client
	client := dns.Client{}

	// create an NS query for our target
	message := dns.Msg{}
	message.SetQuestion(target+".", dns.TypeNS)

	// send the query to the server
	reply, _, err := client.Exchange(&message, server+":53")
	if err != nil {
		return []string{}, err
	}

	// collect the results into a simple string array (stripping off trailing dots)
	result := []string{}
	for _, ans := range reply.Ns {
		if ns, ok := ans.(*dns.NS); ok {
			result = append(result, strings.TrimSuffix(ns.Ns, "."))
		}
	}
	return result, nil
}

// FindFirstSuffixWithWorkingDNS finds the longest suffix of `domain` that has properly configured nameservers. On success it returns the longest suffix and the nameservers for that suffix.
func FindFirstSuffixWithWorkingDNS(domain string) (string, []string, error) {
	suffixParts := strings.Split(domain, ".")[1:]
	for i := range suffixParts {
		suffix := strings.Join(suffixParts[i:], ".")

		nameservers, err := net.LookupNS(suffix)
		if dnsErr, ok := err.(*net.DNSError); ok && dnsErr.Err == "no such host" {
			// treat an NXDOMAIN response the same as an empty array
			nameservers = []*net.NS{}
		} else if err != nil {
			return "", []string{}, fmt.Errorf("fatal error looking up %q: %v", suffix, err)
		}

		// if we find a suffix with valid nameservers, that's our result
		if len(nameservers) > 0 {
			result := []string{}
			for _, ns := range nameservers {
				result = append(result, ns.Host)
			}
			return suffix, result, nil
		}
	}
	return "", []string{}, fmt.Errorf("no suffix of %q found w/working DNS", domain)
}
