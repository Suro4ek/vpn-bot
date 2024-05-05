package wireguard

import (
	"html/template"
	"io"
	"io/fs"
	"net"
	"slices"
	"strings"
)


func parseIPNet(s string) (*net.IPNet, *parseError) {
	_, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		return nil, &parseError{message: err.Error()}
	}
	if ipnet == nil {
		return nil, &parseError{message: "invalid cidr string"}
	}
	return ipnet, nil
}

func StringFromEmbedFile(embed fs.FS, filename string) (string, error) {
	file, err := embed.Open(filename)
	if err != nil {
		return "", err
	}
	content, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func unescape(s string) template.HTML {
	return template.HTML(s)
}

func joinarray(s []interface{}) template.HTML {
	ipv4 := s[0].(string)
	ipv6 := s[1].(string)
	return template.HTML(strings.Join([]string{ipv4, ipv6}, ","))
}

func Delete(collection []Peer, el string) []Peer {
    idx := Find(collection, el)
    if idx > -1 {
        return slices.Delete(collection, idx, idx+1)
    }
    return collection
}

func Find(collection []Peer, el string) int {
    for i := range collection {
        if collection[i].Client == el {
            return i
        }
    }
    return -1
}