package service

import (
	"errors"
	"fmt"
	"h-ui/dao"
	"h-ui/model/constant"
	"h-ui/util"
	"strings"
)

var (
	netManager       string
	nftCommand       string
	iptablesCommand  string
	ip6tablesCommand string
	ingressInterface string
	Add              = "add"
	Delete           = "delete"
	Comment          = "hui_hysteria_porthopping"
)

func init() {
	if nft, err := util.Exec("command -v nft"); err == nil && nft != "" {
		netManager = "nft"
		nftCommand = nft
	} else if iptables, err := util.Exec("command -v iptables"); err == nil && iptables != "" {
		netManager = "iptables"
		iptablesCommand = iptables
		if ip6tablesC, err := util.Exec("command -v ip6tables"); err == nil && ip6tablesC != "" {
			ip6tablesCommand = ip6tablesC
		}
	}

	if ii, err := util.Exec("ls /sys/class/net | grep -E '^en|^eth'"); err == nil && ii != "" {
		ingressInterface = ii
	}
}

func InitTableAndChain() error {
	if netManager == "nft" {
		if nftCommand == "" {
			return fmt.Errorf("nftables not found on the system")
		}

		_, err := util.Exec(fmt.Sprintf("%s add table inet hysteria_porthopping", nftCommand))
		if err != nil {
			return err
		}
		_, err = util.Exec(fmt.Sprintf("%s add chain inet hysteria_porthopping prerouting { type nat hook prerouting priority dstnat\\; policy accept\\; }", nftCommand))
		if err != nil {
			return err
		}
	}
	return nil
}

func InitPortHopping() error {
	if err := removeByComment(); err != nil {
		return err
	}

	hysteria2Config, err := GetHysteria2Config()
	if err != nil {
		return err
	}

	// set port forward
	hysteria2ConfigPortHopping, err := dao.GetConfig("key = ?", constant.Hysteria2ConfigPortHopping)
	if err != nil {
		return err
	}
	if *hysteria2ConfigPortHopping.Value != "" {
		listen := strings.Split(*hysteria2Config.Listen, ":")
		if len(listen) == 2 {
			if err := portForward(*hysteria2ConfigPortHopping.Value, listen[1], Add); err != nil {
				return err
			}
		}
	}
	return nil
}

func portForward(rules string, target string, option string) error {
	switch netManager {
	case "nft":
		if nftCommand == "" {
			return fmt.Errorf("nftables not found on the system")
		}
		switch option {
		case Add, Delete:
			return nftForward(rules, target, option)
		default:
			return errors.New("unsupported command option")
		}
	case "iptables":
		if iptablesCommand == "" || ip6tablesCommand == "" {
			return fmt.Errorf("iptables not found on the system")
		}
		switch option {
		case Add:
			return iptablesForward(rules, target, "-A")
		case Delete:
			return iptablesForward(rules, target, "-D")
		default:
			return errors.New("unsupported command option")
		}
	default:
		return errors.New("port hopping not supported on this system")
	}
}

func removeByComment() error {
	switch netManager {
	case "nft":
		if nftCommand == "" {
			return fmt.Errorf("nftables not found on the system")
		}
		return ntfRemoveByComment(Comment)
	case "iptables":
		if iptablesCommand == "" || ip6tablesCommand == "" {
			return fmt.Errorf("iptables not found on the system")
		}
		return iptablesRemoveByComment(Comment)
	default:
		return errors.New("port hopping not supported on this system")
	}
}

func nftForward(rules string, target string, option string) error {
	if ingressInterface == "" {
		return fmt.Errorf("no network interface detected")
	}
	// nft list ruleset
	// 创建表：nft add table inet hysteria_porthopping
	// 创建链：nft add chain inet hysteria_porthopping prerouting { type nat hook prerouting priority dstnat\; policy accept\; }
	// 添加规则：nft add rule inet hysteria_porthopping prerouting iifname enp1s0 udp dport {30000-40000} counter redirect to :444 comment hui_hysteria_porthopping
	_, err := util.Exec(fmt.Sprintf("%s %s rule inet hysteria_porthopping prerouting iifname %s udp dport {%s} counter redirect to :%s comment %s", nftCommand, option, ingressInterface, rules, target, Comment))
	if err != nil {
		return err
	}

	return nil
}

func ntfRemoveByComment(comment string) error {
	rules, err := nftRules()
	if err != nil {
		return err
	}
	comment = fmt.Sprintf("comment \"%s\"", comment)
	for _, rule := range rules {
		if strings.Contains(rule, comment) {
			parts := strings.Fields(rule)
			handle := parts[len(parts)-1]
			_, err := util.Exec(fmt.Sprintf("%s delete rule inet hysteria_porthopping prerouting handle %s", nftCommand, strings.TrimSpace(handle)))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func nftRules() ([]string, error) {
	output, err := util.Exec(fmt.Sprintf("%s --handle list chain inet hysteria_porthopping prerouting", nftCommand))
	if err != nil {
		return nil, err
	}

	rules := strings.Split(output, "\n")
	return rules, nil
}

func iptablesForward(rules string, target string, option string) error {
	if ingressInterface == "" {
		return fmt.Errorf("no network interface detected")
	}

	rulePairs := strings.Split(rules, ",")
	for _, pair := range rulePairs {
		ports := ""
		portRange := strings.Split(pair, "-")
		if len(portRange) == 1 {
			ports = strings.TrimSpace(portRange[0])
		} else if len(portRange) == 2 {
			startPort := strings.TrimSpace(portRange[0])
			endPort := strings.TrimSpace(portRange[1])
			ports = startPort + ":" + endPort
		} else {
			return fmt.Errorf("invalid port range format: %s", pair)
		}

		if len(ports) != 0 {
			if err := iptablesAddRule(option, ports, target); err != nil {
				return err
			}
		}
	}

	return nil
}

func iptablesAddRule(option, ports, target string) error {
	// iptables -t nat -A PREROUTING -i enp1s0 -p udp --dport 30000:40000 -j REDIRECT --to-port 444 -m comment --comment hui_hysteria_porthopping
	_, err := util.Exec(fmt.Sprintf("%s -t nat %s PREROUTING -i %s -p udp --dport %s -j REDIRECT --to-port %s -m comment --comment %s", iptablesCommand, option, ingressInterface, ports, target, Comment))
	if err != nil {
		return err
	}
	// ip6tables -t nat -A PREROUTING -i enp1s0 -p udp --dport 30000:40000 -j REDIRECT --to-port 444 -m comment --comment hui_hysteria_porthopping
	_, err = util.Exec(fmt.Sprintf("%s -t nat %s PREROUTING -i %s -p udp --dport %s -j REDIRECT --to-port %s -m comment --comment %s", ip6tablesCommand, option, ingressInterface, ports, target, Comment))
	if err != nil {
		return err
	}
	return nil
}

func iptablesRemoveByComment(comment string) error {
	protocols := [2]string{"iptables", "ip6tables"}
	for _, protocol := range protocols {
		rules, err := iptablesRules(protocol)
		if err != nil {
			return err
		}
		for _, rule := range rules {
			if strings.Contains(rule, comment) {
				parts := strings.Fields(rule)
				handle := parts[len(parts)-1]
				_, err := util.Exec(fmt.Sprintf("%s -t nat -D PREROUTING %s", protocol, strings.TrimSpace(handle)))
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func iptablesRules(protocol string) ([]string, error) {
	// iptables -t nat -L PREROUTING -v --line-numbers
	output, err := util.Exec(fmt.Sprintf("%s -t nat -L PREROUTING -v --line-numbers", protocol))
	if err != nil {
		return nil, err
	}

	rules := strings.Split(output, "\n")
	return rules, nil
}
