package app

import "charm.land/lipgloss/v2"

const accentColor = "#128182"

type viewTheme struct {
	brand       lipgloss.Style
	breadcrumb  lipgloss.Style
	status      lipgloss.Style
	primary     lipgloss.Style
	muted       lipgloss.Style
	separator   lipgloss.Style
	selected    lipgloss.Style
	section     lipgloss.Style
	label       lipgloss.Style
	value       lipgloss.Style
	shortcutKey lipgloss.Style
	modal       lipgloss.Style
	modalTitle  lipgloss.Style
	notice      lipgloss.Style
}

func newViewTheme(darkBackground bool) viewTheme {
	lightDark := lipgloss.LightDark(darkBackground)
	primary := lightDark(lipgloss.Color("#252525"), lipgloss.Color("#E7E7E7"))
	muted := lightDark(lipgloss.Color("#686868"), lipgloss.Color("#9A9A9A"))
	separator := lightDark(lipgloss.Color("#D7D7D7"), lipgloss.Color("#3D3D3D"))
	accent := lipgloss.Color(accentColor)

	return viewTheme{
		brand:       lipgloss.NewStyle().Bold(true).Foreground(accent),
		breadcrumb:  lipgloss.NewStyle().Foreground(primary),
		status:      lipgloss.NewStyle().Foreground(accent),
		primary:     lipgloss.NewStyle().Foreground(primary),
		muted:       lipgloss.NewStyle().Foreground(muted),
		separator:   lipgloss.NewStyle().Foreground(separator),
		selected:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(accent),
		section:     lipgloss.NewStyle().Bold(true).Foreground(accent),
		label:       lipgloss.NewStyle().Foreground(muted),
		value:       lipgloss.NewStyle().Foreground(primary),
		shortcutKey: lipgloss.NewStyle().Bold(true).Foreground(accent),
		modal: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(separator).
			Padding(1, 2),
		modalTitle: lipgloss.NewStyle().Bold(true).Foreground(accent),
		notice:     lipgloss.NewStyle().Foreground(primary),
	}
}
