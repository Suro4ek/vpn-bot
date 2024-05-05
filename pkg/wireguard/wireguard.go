package wireguard

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"strings"
	"vpn-bot/pkg/utils"
)

type WiregaurdConfig interface {
	GetClients() ([]string)
	AddClient(name string) error
	RemoveClient(name string) error
}

type wireguardConfig struct {
	cfg *Config
}

func NewWireguardConfig(cfg *Config) WiregaurdConfig {
	return wireguardConfig{
		cfg: cfg,
	}
}

// AddClient implements WiregaurdConfig.
func (w wireguardConfig) AddClient(name string) error {
	name = strings.ToLower(name)
	for _, client := range w.GetClients() {
		if client == name {
			return errors.New("client exists")
		}
	}
	getlastIP := w.cfg.Peers[len(w.cfg.Peers)-1].AllowedIPs
	ips, err := w.getIPS(getlastIP, w.cfg.Address)
	if err != nil {
		return err
	}
	privKey, err := GeneratePrivateKey()
	if err != nil {
		return err
	}
	preshKey, err := GenerateKey()
	if err != nil {
		return err
	}
	peerSave := PeerSave{
		Address: []string{ips[0].String(), ips[1].String()}, 
		EndPoint: fmt.Sprintf("%s:%d", w.cfg.IpServer, *w.cfg.ListenPort),
		PrivateKey: privKey,
		PresharedKey: &preshKey,
		PublicKey: w.cfg.PrivateKey.PublicKey(),
	}
	err = w.saveClient(name, &peerSave)
	if err != nil {
		return err
	}
	peer := Peer{
		Client: name,
		PublicKey: privKey.PublicKey(),
		PresharedKey: &preshKey,
		AllowedIPs: ips,
	}
	w.cfg.Peers = append(w.cfg.Peers, peer)
	wg := NewParserConfig()
	wg.SaveConfig(w.cfg)
	w.syncWireguard()
	return nil
}

func (w wireguardConfig) getIPS(getlastIP []myIPnet, subnet []myIPnet) ([]myIPnet, error) {
	newIps := []myIPnet{}
	ipv4 := getlastIP[0]
	ip := utils.DupIP(ipv4.IP)
	ip[len(ip)-1]++
	if !subnet[0].Contains(ip) {
		return nil, errors.New("not ips1")
	}
	ipNet := net.IPNet{IP: ip, Mask: ipv4.Mask}
	ip1 := myIPnet{ipNet }
	newIps = append(newIps, ip1)
	ipv6 := getlastIP[1]
	ip = utils.DupIP(ipv6.IP)
	ip[len(ip)-1]++
	if !subnet[1].Contains(ip) {
		return nil, errors.New("not ips")
	}
	ipNet = net.IPNet{IP: ip, Mask: ipv6.Mask}
	ip1 = myIPnet{ipNet }
	newIps = append(newIps, ip1)
	return newIps, nil
}

// GetClients implements WiregaurdConfig.
func (w wireguardConfig) GetClients() ([]string) {
	clients := []string{}
	for _, client := range w.cfg.Peers {
		clients = append(clients, client.Client)
	}
	return clients
}

// RemoveClient implements WiregaurdConfig.
func (w wireguardConfig) RemoveClient(name string) error {
	w.cfg.Peers = Delete(w.cfg.Peers, name)
	wg := NewParserConfig()
	wg.SaveConfig(w.cfg)
	w.syncWireguard()
	return nil
}

func (w wireguardConfig) saveClient(client string, peer *PeerSave) error {
	tmplDir, _ := fs.Sub(fs.FS(embeddedTemplates), "templates")
	fileContent, err := StringFromEmbedFile(tmplDir, "wg-client.conf")
	if err != nil {
		return err
	}
	tmplWireguardConf := fileContent
	t, err := template.New("wg-client_config").Funcs(template.FuncMap{"unescape": unescape}).Funcs(template.FuncMap{"ipvs": joinarray}).Parse(tmplWireguardConf)
	if err != nil {
		return err
	}
	f, err := os.Create(fmt.Sprintf("./client-%s.conf", client))
	if err != nil {
		return err
	}
	defer f.Close()
	data, _ := json.Marshal(peer)
	config := make(map[string]interface{})
    json.Unmarshal(data, &config)
	err = t.Funcs(template.FuncMap{"unescape": unescape}).Funcs(template.FuncMap{"ipvs": joinarray}).Execute(f, config)
	if err != nil {
		return err
	}
	return nil
}

func (w wireguardConfig) syncWireguard() error {
	cmd := exec.Command("bash", "-c", "wg syncconf wg0 <(wg-quick strip wg0)")
    _, err := cmd.Output()
	fmt.Println(err)
	return err
}

