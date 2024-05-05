package utils

import (
	"net"
)

func DupIP(ip net.IP) net.IP {
    // // To save space, try and only use 4 bytes
    // if x := ip.To4(); x != nil {
    //     ip = x
    // }
    dup := make(net.IP, len(ip))
    copy(dup, ip)
    return dup
}

