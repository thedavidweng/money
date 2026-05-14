package plaidlogin

import (
	"fmt"
	"strconv"
	"strings"
)

func SelectTeam(teams []Team, selector string) (Team, error) {
	if len(teams) == 0 {
		return Team{}, Error{Code: ErrorTeamSelectionRequired, Message: "Plaid Dashboard returned no teams"}
	}
	if selector == "" {
		if len(teams) == 1 {
			return teams[0], nil
		}
		return Team{}, Error{Code: ErrorTeamSelectionRequired, Message: "multiple Plaid Dashboard teams are available; choose one with --team"}
	}

	var matches []Team
	if index, err := strconv.Atoi(selector); err == nil && index >= 1 && index <= len(teams) {
		matches = appendUniqueTeam(matches, teams[index-1])
	}
	for _, team := range teams {
		switch {
		case team.TeamID == selector:
			matches = appendUniqueTeam(matches, team)
		case team.ClientID == selector:
			matches = appendUniqueTeam(matches, team)
		case strings.EqualFold(team.Name, selector):
			matches = appendUniqueTeam(matches, team)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return Team{}, Error{Code: ErrorTeamSelectionRequired, Message: fmt.Sprintf("no Plaid Dashboard team matched %q", selector)}
	default:
		return Team{}, Error{Code: ErrorTeamSelectionRequired, Message: fmt.Sprintf("Plaid Dashboard team selector %q is ambiguous", selector)}
	}
}

func SecretForEnvironment(keys Keys, environment string) (string, error) {
	if environment == "" {
		environment = "sandbox"
	}
	switch environment {
	case "sandbox", "production":
	default:
		return "", Error{Code: "INVALID_ENUM", Message: "--environment must be sandbox or production"}
	}
	secret := keys.Secrets[environment]
	if secret == "" {
		return "", Error{Code: ErrorPlaidEnvironmentNotProvided, Message: fmt.Sprintf("Plaid Dashboard did not return a %s secret", environment)}
	}
	return secret, nil
}

func appendUniqueTeam(teams []Team, team Team) []Team {
	for _, existing := range teams {
		if existing.TeamID == team.TeamID {
			return teams
		}
	}
	return append(teams, team)
}
