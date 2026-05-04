package ai

import (
	"fmt"
	"strings"
)

func Validate(sql string) error {
	s := strings.TrimSpace(sql)
	if s == "" {
		return fmt.Errorf("empty sql")
	}
	if strings.Contains(s, "--") || strings.Contains(s, "/*") || strings.Contains(s, "*/") {
		return fmt.Errorf("sql comments not allowed")
	}

	upper := strings.ToUpper(s)
	forbidden := []string{
		"INSERT", "UPDATE", "DELETE", "DROP", "CREATE",
		"ALTER", "TRUNCATE", "EXEC", "EXECUTE",
	}
	for _, w := range forbidden {
		if strings.Contains(upper, w) {
			return fmt.Errorf("forbidden keyword: %s", w)
		}
	}

	trimmed := strings.TrimSpace(strings.TrimSuffix(s, ";"))
	if strings.Contains(trimmed, ";") {
		return fmt.Errorf("statement chaining not allowed")
	}

	if !strings.HasPrefix(strings.TrimSpace(strings.ToUpper(trimmed)), "SELECT") {
		return fmt.Errorf("must start with SELECT")
	}
	return nil
}

// InjectAgentFilter replaces :agent_id with $1 and ensures assigned_agent_id filter for non-admin agents.
func InjectAgentFilter(sql, agentID, role string) (string, error) {
	if strings.TrimSpace(agentID) == "" {
		return "", fmt.Errorf("missing agent id")
	}
	if role == "admin" {
		return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(sql), ";")), nil
	}

	s := strings.TrimSpace(sql)
	s = strings.TrimSuffix(s, ";")
	// No FROM (scalar / noop SELECT): never append tenant filter — would be invalid SQL.
	if !strings.Contains(strings.ToUpper(s), "FROM") {
		return s, nil
	}
	orig := s
	hasPlaceholder := strings.Contains(orig, ":agent_id")
	s = strings.ReplaceAll(s, ":agent_id", "$1")

	upper := strings.ToUpper(s)
	if !hasPlaceholder && !strings.Contains(upper, "ASSIGNED_AGENT_ID") {
		if strings.Contains(upper, " WHERE ") {
			s = s + " AND assigned_agent_id = $1"
		} else {
			s = s + " WHERE assigned_agent_id = $1"
		}
	}
	return s, nil
}
