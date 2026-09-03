package main

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	accent  = lipgloss.Color("#7D56F4")
	accent2 = lipgloss.Color("#FF87D7")
	good    = lipgloss.Color("#04B575")
	muted   = lipgloss.Color("#626262")
	bad     = lipgloss.Color("#FF5F5F")
	warn    = lipgloss.Color("#FFAF5F")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(accent).
			Padding(0, 1).
			MarginBottom(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accent2).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(accent)

	currentStyle = lipgloss.NewStyle().Foreground(good).Bold(true)
	branchStyle  = lipgloss.NewStyle().Foreground(accent2)
	pathStyle    = lipgloss.NewStyle().Foreground(muted)
	headStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#8B9CF0"))
	infoStyle    = lipgloss.NewStyle().Foreground(accent)
	okStyle      = lipgloss.NewStyle().Foreground(good).Bold(true)
	errStyle     = lipgloss.NewStyle().Foreground(bad).Bold(true)
	warnStyle    = lipgloss.NewStyle().Foreground(warn)
	dimStyle     = lipgloss.NewStyle().Foreground(muted).Italic(true)
)

func banner() string {
	logo := lipgloss.NewStyle().
		Bold(true).
		Foreground(accent2).
		Render("◆ wt")
	sub := dimStyle.Render("worktrees, beautifully")
	return lipgloss.JoinHorizontal(lipgloss.Center, logo, "  ", sub)
}
