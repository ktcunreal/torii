package main

import (
	"../smux"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type Config struct {
	Ingress string `json:"ingress"`
	Mode    string `json:"mode"`
	Egress  string `json:"egress"`
	PSK     string `json:"key"`
	keyring smux.Keyring
}

func readFromConfig() *Config {
	config := flag.String("c", "", "Configuration path")
	i := flag.String("i", "", "ingress listen address")
	e := flag.String("e", "", "egress address")
	p := flag.String("p", "", "pre shared key")
	m := flag.String("m", "", "mode")
	flag.Parse()

	c := &Config{}
	if *config != "" {
		file, err := os.Open(*config)
		defer file.Close()
		log.Printf("LOADING CONFIG FROM %s", *config)
		if err != nil {
			log.Fatalf("LOADING CONFIG FAILED %v", err)
		}
		if err := json.NewDecoder(file).Decode(c); err != nil {
			log.Fatalf("PARSE CONFIG ERROR: %v", err)
		}
		fmt.Printf("%v\n", c)
		return c
	}

	c.Ingress = *i
	c.Egress = *e
	c.PSK = *p
	c.Mode = *m
	fmt.Printf("%v\n", c)
	return c
}
func (c *Config) ValidateConf() {
	if len(c.Ingress)*len(c.Egress) > 0 {

	}
}

func validateIP(s string) bool {
	var ip, port string
	i := strings.LastIndexByte(s, ':')
	if i == -1 {
		return false
	}
	ip, port = s[:i], s[i+1:]
	if len(ip) == 0 || net.ParseIP(ip) == nil {
		return false
	}
	if p, err := strconv.Atoi(port); err != nil || p > 65535 || p < 1 {
		return false
	}
	return true
}

func validateDomain(s string) bool {
	RegExp := regexp.MustCompile(`^(([a-zA-Z]{1})|([a-zA-Z]{1}[a-zA-Z]{1})|([a-zA-Z]{1}[0-9]{1})|([0-9]{1}[a-zA-Z]{1})|([a-zA-Z0-9][a-zA-Z0-9-_]{1,61}[a-zA-Z0-9]))\.([a-zA-Z]{2,6}|[a-zA-Z0-9-]{2,30}\.[a-zA-Z
 ]{2,3}):\d{1,5}$`)
	if RegExp.MatchString(s) {
		return true
	}
	return false
}
