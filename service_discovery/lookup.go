package service_discovery

import (
	"context"
	"log"
	"net"
	"sort"
	"strconv"
)

// Query the SRV records for the receiving entity's domain.
// The returned records are sorted based on Priority and Weight.
func LookupServerSRVRecords(domain string) ([]*net.SRV, error) {
    var resolver net.Resolver = net.Resolver{PreferGo: true, StrictErrors: false}
    const service string = "xmpp-client"
    const proto string = "tcp"
    
    // TODO: ADD CONTEXT
    _, addrs, err := resolver.LookupSRV(context.TODO(), service, proto, domain)
    if err != nil {
	return nil, err
    }
    //log.Println("CNAME: " + cname) 
    log.Println("TOTAL ADDRS: " + strconv.Itoa(len(addrs)))
    log.Printf("ADDR 0 : %v\n", addrs[0])

    // Sort the records
    sortSRVRecords(addrs)
    
    return addrs, nil
}

// Sort a slice of SRV records on Priority and then Weight.
func sortSRVRecords(addrs []*net.SRV) {
    var totalRecords int = len(addrs)
    // Check if there is only one record
    if totalRecords == 1 {
	return
    }
    
    // Sort the records based on Priority and Weight
    sort.SliceStable(addrs, func(i, j int) bool {
	if addrs[i].Priority > addrs[j].Priority {
	    return true
	} else {
	    return addrs[i].Priority == addrs[j].Priority && addrs[i].Weight > addrs[j].Weight
	} 
    })
}

// Resolve the IPv4 and IPv6 addresses of the domain specified.
// This function can be used as the fallback process for resolving
// the IP addresses of the receiving entity. 
func ResolveServerIPAddresses(domain string) ([]net.IP, error) {
    // TODO: ADD CONTEXT
    var resolver net.Resolver = net.Resolver{PreferGo: true, StrictErrors: false}
    ips, err := resolver.LookupIP(context.TODO(), "ip", domain)
    if err != nil {
	return nil, err 
    }
    return ips, err
}
