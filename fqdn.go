package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

func fqdn() string {
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("Error getting hostname: %v\n", err)
		return ""
	}

	addrs, err := net.LookupHost(hostname)
	if err != nil {
		log.Fatalf("Error looking up host: %v\n", err)
		return ""
	}

	for _, addr := range addrs {
		names, err := net.LookupAddr(addr)
		if err != nil {
			fmt.Printf("Error looking up address: %v\n", err)
			continue
		}
		if len(names) == 0 {
			continue
		}

		return strings.TrimSuffix(names[0], ".")
	}

	log.Fatalf("Could not find FQDN for %s\n", hostname)

	return ""
}
