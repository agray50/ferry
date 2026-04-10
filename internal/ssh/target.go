package ssh

import (
	"fmt"
	"net"
	"os/user"
	"strconv"
	"strings"
)

// ParsedTarget holds a parsed SSH target string.
type ParsedTarget struct {
	User string
	Host string
	Port int
}

// ParseTarget parses "user@host" or "user@host:port".
// Uses current OS user if user not specified.
func ParseTarget(target string) (ParsedTarget, error) {
	pt := ParsedTarget{Port: 22}

	// split user@rest
	if idx := strings.Index(target, "@"); idx >= 0 {
		pt.User = target[:idx]
		target = target[idx+1:]
	} else {
		u, err := user.Current()
		if err != nil {
			return pt, fmt.Errorf("could not determine current user: %w", err)
		}
		pt.User = u.Username
	}

	// split host:port
	host, portStr, err := net.SplitHostPort(target)
	if err != nil {
		// no port — target is just host
		pt.Host = target
	} else {
		pt.Host = host
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return pt, fmt.Errorf("invalid port %q", portStr)
		}
		pt.Port = port
	}

	if pt.Host == "" {
		return pt, fmt.Errorf("empty host in target")
	}
	return pt, nil
}

// String returns the target in user@host:port form.
func (pt ParsedTarget) String() string {
	if pt.Port == 22 {
		return fmt.Sprintf("%s@%s", pt.User, pt.Host)
	}
	return fmt.Sprintf("%s@%s:%d", pt.User, pt.Host, pt.Port)
}

// Addr returns host:port.
func (pt ParsedTarget) Addr() string {
	return fmt.Sprintf("%s:%d", pt.Host, pt.Port)
}
