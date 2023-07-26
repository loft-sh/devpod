package port

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

type Address struct {
	Protocol string
	Address  string
}

type Mapping struct {
	Host      Address
	Container Address
}

func ParsePortSpec(port string) (Mapping, error) {
	hostIP, hostPort, containerIP, containerPort, err := splitParts(port)
	if err != nil {
		return Mapping{}, err
	}

	hostAddress, err := toAddress(hostIP, hostPort)
	if err != nil {
		return Mapping{}, fmt.Errorf("parse host address: %w", err)
	}

	containerAddress, err := toAddress(containerIP, containerPort)
	if err != nil {
		return Mapping{}, fmt.Errorf("parse container address: %w", err)
	}

	return Mapping{
		Host:      hostAddress,
		Container: containerAddress,
	}, nil
}

func toAddress(ip, port string) (Address, error) {
	// check if port is integer
	_, err := strconv.Atoi(port)
	if err == nil {
		if ip == "" {
			ip = "localhost"
		}

		if ip != "localhost" && net.ParseIP(ip) == nil {
			return Address{}, fmt.Errorf("not an ip address %s", ip)
		}

		return Address{
			Protocol: "tcp",
			Address:  ip + ":" + port,
		}, nil
	}

	if ip != "" {
		return Address{}, fmt.Errorf("unexpected ip address for unix socket: %s", ip)
	}

	return Address{
		Protocol: "unix",
		Address:  port,
	}, nil
}

func splitParts(rawport string) (string, string, string, string, error) {
	parts := strings.Split(rawport, ":")
	n := len(parts)
	containerport := parts[n-1]

	switch n {
	case 1:
		return "", containerport, "", containerport, nil
	case 2:
		return "", parts[0], "", containerport, nil
	case 3:
		if parts[1] == "localhost" || net.ParseIP(parts[1]) != nil {
			return "", parts[0], parts[1], containerport, nil
		}

		return parts[0], parts[1], "", containerport, nil
	case 4:
		return parts[0], parts[1], parts[2], parts[3], nil
	default:
		return "", "", "", "", fmt.Errorf("unexpected port format: %s", rawport)
	}
}
