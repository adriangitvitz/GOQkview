package parser

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type BigIPConfig struct {
	VirtualServers map[string]*VirtualServerConfig
	Pools          map[string]*PoolConfig
}

type VirtualServerConfig struct {
	Name        string
	Pool        string // Pool reference (cleaned name)
	Disabled    bool
	Destination string
}

type PoolConfig struct {
	Name    string
	Members []PoolMember
	Monitor string
}

type PoolMember struct {
	Name     string // Node:port
	Address  string
	Disabled bool // session user-disabled
	Down     bool // state user-down
}

type parseState int

const (
	stateNone parseState = iota
	stateVirtual
	statePool
	statePoolMembers
	stateMember
)

var (
	virtualPattern = regexp.MustCompile(`^ltm virtual\s+(/\S+)\s*\{`)
	poolPattern    = regexp.MustCompile(`^ltm pool\s+(/\S+)\s*\{`)
	memberPattern  = regexp.MustCompile(`^\s*(/\S+:\d+)\s*\{`)
	poolRefPattern = regexp.MustCompile(`^\s*pool\s+(/\S+)`)
	destPattern    = regexp.MustCompile(`^\s*destination\s+(/\S+)`)
	addressPattern = regexp.MustCompile(`^\s*address\s+(\S+)`)
	monitorPattern = regexp.MustCompile(`^\s*monitor\s+(/\S+)`)
)

func ParseBigIPConfig(filePath string) (*BigIPConfig, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("bigip: failed to open config file: %w", err)
	}
	defer file.Close()

	config := &BigIPConfig{
		VirtualServers: make(map[string]*VirtualServerConfig),
		Pools:          make(map[string]*PoolConfig),
	}

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var state parseState
	var braceDepth int          // Total brace depth
	var blockStartDepth int     // Depth when we entered current block
	var membersStartDepth int   // Depth when we entered members block
	var memberStartDepth int    // Depth when we entered individual member

	var currentVS *VirtualServerConfig
	var currentPool *PoolConfig
	var currentMember *PoolMember

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		openBraces := strings.Count(trimmed, "{")
		closeBraces := strings.Count(trimmed, "}")

		if matches := virtualPattern.FindStringSubmatch(line); matches != nil {
			state = stateVirtual
			blockStartDepth = braceDepth
			braceDepth += openBraces
			name := cleanName(matches[1])
			currentVS = &VirtualServerConfig{Name: name}
			continue
		}

		if matches := poolPattern.FindStringSubmatch(line); matches != nil {
			state = statePool
			blockStartDepth = braceDepth
			braceDepth += openBraces
			name := cleanName(matches[1])
			currentPool = &PoolConfig{Name: name, Members: []PoolMember{}}
			continue
		}

		if state == statePool && strings.Contains(trimmed, "members {") {
			state = statePoolMembers
			membersStartDepth = braceDepth
			braceDepth += openBraces
			continue
		}

		if state == statePoolMembers {
			if matches := memberPattern.FindStringSubmatch(line); matches != nil {
				state = stateMember
				memberStartDepth = braceDepth
				braceDepth += openBraces
				name := cleanName(matches[1])
				currentMember = &PoolMember{Name: name}
				continue
			}
		}

		braceDepth += openBraces
		braceDepth -= closeBraces

		if closeBraces > 0 {
			if state == stateMember && braceDepth <= memberStartDepth {
				if currentPool != nil && currentMember != nil {
					currentPool.Members = append(currentPool.Members, *currentMember)
				}
				currentMember = nil
				state = statePoolMembers
			}

			if state == statePoolMembers && braceDepth <= membersStartDepth {
				state = statePool
			}

			if state == statePool && braceDepth <= blockStartDepth {
				if currentPool != nil {
					config.Pools[currentPool.Name] = currentPool
				}
				currentPool = nil
				state = stateNone
			}

			if state == stateVirtual && braceDepth <= blockStartDepth {
				if currentVS != nil {
					config.VirtualServers[currentVS.Name] = currentVS
				}
				currentVS = nil
				state = stateNone
			}
		}

		if closeBraces == 0 || openBraces > 0 {
			switch state {
			case stateVirtual:
				if currentVS != nil {
					if matches := poolRefPattern.FindStringSubmatch(line); matches != nil {
						currentVS.Pool = cleanName(matches[1])
					} else if matches := destPattern.FindStringSubmatch(line); matches != nil {
						currentVS.Destination = cleanName(matches[1])
					} else if trimmed == "disabled" {
						currentVS.Disabled = true
					}
				}

			case statePool, statePoolMembers:
				if currentPool != nil {
					if matches := monitorPattern.FindStringSubmatch(line); matches != nil {
						currentPool.Monitor = cleanName(matches[1])
					}
				}

			case stateMember:
				if currentMember != nil {
					if matches := addressPattern.FindStringSubmatch(line); matches != nil {
						currentMember.Address = matches[1]
					} else if strings.Contains(trimmed, "session user-disabled") {
						currentMember.Disabled = true
					} else if strings.Contains(trimmed, "state user-down") {
						currentMember.Down = true
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("bigip: scanner error: %w", err)
	}

	return config, nil
}

func cleanName(fullName string) string {
	parts := strings.Split(fullName, "/")
	if len(parts) >= 3 {
		return parts[len(parts)-1]
	}
	return strings.TrimPrefix(fullName, "/")
}

func (p *PoolConfig) GetActiveMembers() int {
	active := 0
	for _, m := range p.Members {
		if !m.Disabled && !m.Down {
			active++
		}
	}
	return active
}

func (p *PoolConfig) GetTotalMembers() int {
	return len(p.Members)
}
