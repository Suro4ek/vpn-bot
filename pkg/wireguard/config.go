package wireguard

import (
	"bufio"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type parser struct {
}

type pair struct {
	key   string
	value string
}

type Config struct {
	Interface 
	Peers []Peer `json:"peers"`
}

type Interface struct {
	Address []myIPnet `json:"address"`
	ListenPort *int `json:"listenPort"`
	PrivateKey *Key `json:"privateKey"`
	PostUp string `json:"postUp"`
	PostDown string `json:"postDown"`
	IpServer string `json:"ipServer"`
}

type myIPnet struct{
	net.IPNet
}

type Peer struct {
	Client string `json:"client"`
	PublicKey Key `json:"publicKey"`
	PresharedKey *Key `json:"presharedKey"`
	AllowedIPs []myIPnet`json:"allowedIps"`
}

type PeerSave struct {
	Address []string `json:"address"`
	EndPoint string `json:"endpoint"`
	PrivateKey Key `json:"privateKey"`
	PublicKey Key `json:"publicKey"`
	PresharedKey *Key `json:"presharedKey"`
}

// func (u *Peer) MarshalJSON() ([]byte, error) {
	
//     return json.Marshal(&u)
// }

func NewParserConfig() parser{
	return parser{}
}

const (
	sectionInterface = "Interface"
	sectionPeer      = "Peer"
	sectionEmpty     = ""
)

type parseError struct {
	message string
	line    int
}

func (p parseError) Error() string {
	return fmt.Sprintf("Parse error: %s, (line %d)", p.message, p.line)
}

func (p parser) LoadConfig(path string) (*Config, error){
	if _, err := os.Stat("./wg0.json"); errors.Is(err, os.ErrNotExist) {
		cfg, err := p.ParseFile("/etc/wireguard/wg0.conf")
		if err != nil {
			return nil, err
		}
		envFile, _ := godotenv.Read("/etc/wireguard/params")
		cfg.IpServer = envFile["SERVER_PUB_IP"]

		err = p.SaveConfig(cfg)
		if err != nil {
			return nil, err
		}
	}
	file, err := os.Open(path)
	if err != nil {
	  fmt.Println("File reading error", err)
	  return nil, err
	}
	defer file.Close()
	byteValue, _ := io.ReadAll(file)
	var cfg Config
	err = json.Unmarshal(byteValue, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (p parser) ParseFile(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
	  fmt.Println("File reading error", err)
	  return nil, err
	}
	defer file.Close()
	var cfg *Config = nil
	var currentPeerConfig *Peer = nil
	currentSec := sectionEmpty

	peers := make([]Peer, 0, 5)
	sc := bufio.NewScanner(file)
	for lineNum := 0; sc.Scan(); lineNum++ {
		line := sc.Text()
		if strings.TrimSpace(line) == "" {
			// skip empty line for fast read
			continue
		}
		if sec, matched := matchClientHeader(line); matched {
			if currentPeerConfig != nil {
				peers = append(peers, *currentPeerConfig)
			}
			currentPeerConfig = &Peer{}
			currentPeerConfig.Client = sec
		}else if sec, matched := matchSectionHeader(line); matched {
			if sec == sectionInterface {
				if cfg != nil {
					return nil, parseError{message: "duplicated Interface section", line: lineNum}
				}
				cfg = &Config{}
			} else if sec == sectionPeer {

			} else {
				return nil, parseError{message: fmt.Sprintf("Unknown section: %s", sec), line: lineNum}
			}
			currentSec = sec
			continue
		} else if pair, matched := matchKeyValuePair(line); matched {
			var perr *parseError
			if currentSec == sectionEmpty {
				return nil, parseError{message: "invalid top level key-value pair", line: lineNum}
			}
			if currentSec == sectionInterface {
				perr = parseInterfaceField(cfg, pair)
			} else if currentSec == sectionPeer {
				perr = parsePeerField(currentPeerConfig, pair)
			}
			if perr != nil {
				perr.line = lineNum
				return nil, perr
			}
		}
	}
	if currentSec == sectionPeer {
		peers = append(peers, *currentPeerConfig)
	}
	if cfg == nil {
		return nil, parseError{message: "no Interface section found"}
	}
	cfg.Peers = peers
	if err := sc.Err(); err != nil {
	  log.Fatal(err)
	  return nil, err
	}
	return cfg, nil
}

// embed the "templates" directory
//
//go:embed templates/*
var embeddedTemplates embed.FS

func (p parser) SaveConfig(cfg *Config) error {
	tmplDir, _ := fs.Sub(fs.FS(embeddedTemplates), "templates")
	fileContent, err := StringFromEmbedFile(tmplDir, "wg.conf")
	if err != nil {
		return err
	}
	tmplWireguardConf := fileContent
	t, err := template.New("wg_config").Funcs(template.FuncMap{"unescape": unescape}).Funcs(template.FuncMap{"ipvs": joinarray}).Parse(tmplWireguardConf)
	if err != nil {
		return err
	}
	f, err := os.Create("/etc/wireguard/wg0.conf")
	if err != nil {
		return err
	}
	data, _ := json.Marshal(cfg)
	config := make(map[string]interface{})
    json.Unmarshal(data, &config)
	err = t.Funcs(template.FuncMap{"unescape": unescape}).Funcs(template.FuncMap{"ipvs": joinarray}).Execute(f, config)
	if err != nil {
		return err
	}
	f.Close()
	file, _ := json.MarshalIndent(cfg, "", " ")
	_ = os.WriteFile("wg0.json", file, 0644)
	return nil
}

func (k *myIPnet) UnmarshalJSON(data []byte) (error) {
	seg := strings.TrimSpace(strings.ReplaceAll(string(data), "\"", ""))
	ip, err := parseIPNet(seg)
	if err != nil {
		return err
	}
	*k = myIPnet{*ip}
    return nil
}


func (k myIPnet) MarshalJSON() ([]byte, error) {
	str := k.String()
    return json.Marshal(&str)
}

func parseInterfaceField(cfg *Config, p pair) *parseError {
	switch p.key {
	case "PrivateKey":
		key, err := decodeKey(p.value)
		if err != nil {
			return err
		}
		cfg.PrivateKey = &key
	case "ListenPort":
		port, err := strconv.Atoi(p.value)
		if err != nil {
			return &parseError{message: err.Error()}
		}
		cfg.ListenPort = &port
	case "Address":
		allowedIPs := make([]myIPnet, 0, 10)
		splitted := strings.Split(p.value, ",")
		for _, seg := range splitted {
			seg = strings.TrimSpace(seg)
			ip, err := parseIPNet(seg)
			if err != nil {
				return err
			}
			myIpNet := myIPnet{*ip}
			allowedIPs = append(allowedIPs, myIpNet)
		}
		cfg.Address = allowedIPs
	case "PostUp":
		cfg.PostUp = p.value
	case "PostDown":
		cfg.PostDown = p.value
	default:
		return &parseError{message: fmt.Sprintf("invalid key %s for Interface section", p.key)}
	}
	return nil
}

func parsePeerField(cfg *Peer, p pair) *parseError {
	switch p.key {
	case "PublicKey":
		key, err := decodeKey(p.value)
		if err != nil {
			return err
		}
		cfg.PublicKey = key
	case "PresharedKey":
		key, err := decodeKey(p.value)
		if err != nil {
			return err
		}
		cfg.PresharedKey = &key
	case "AllowedIPs":
		allowedIPs := make([]myIPnet, 0, 10)
		splitted := strings.Split(p.value, ",")
		for _, seg := range splitted {
			seg = strings.TrimSpace(seg)
			ip, err := parseIPNet(seg)
			if err != nil {
				return err
			}
			myIpNet := myIPnet{*ip}
			allowedIPs = append(allowedIPs, myIpNet)
		}
		cfg.AllowedIPs = allowedIPs
	default:
		return &parseError{message: fmt.Sprintf("invalid key %s for Peer section", p.key)}
	}
	return nil
}

func matchSectionHeader(s string) (string, bool) {
	re := regexp.MustCompile(`\[(?P<section>\w+)\]`)
	matched := re.MatchString(s)
	if !matched {
		return "", false
	}
	sec := re.ReplaceAllString(s, "${section}")
	return sec, true
}

func matchClientHeader(s string) (string, bool) {
	re := regexp.MustCompile(`\### Client (?P<section>\w+)`)
	matched := re.MatchString(s)
	re1 := regexp.MustCompile(`\###Client (?P<section>\w+)`)
	matched1 := re1.MatchString(s)
	if !matched && !matched1 {
		return "", false
	}
	sec := ""
	if matched {
		sec = re.ReplaceAllString(s, "${section}")
	}else {
		sec = re1.ReplaceAllString(s, "${section}")
	}
	return sec, true
}
func matchKeyValuePair(s string) (pair, bool) {
	re := regexp.MustCompile(`^\s*(?P<key>\w+)\s*=\s*(?P<value>.+)\s*$`)
	matched := re.MatchString(s)
	if !matched {
		return pair{}, false
	}
	key := re.ReplaceAllString(s, "${key}")
	value := re.ReplaceAllString(s, "${value}")
	return pair{key: key, value: value}, true
}