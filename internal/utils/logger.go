package utils

import (
	"log"
	"os"
	"time"

	"github.com/charmbracelet/lipgloss"
	log1 "github.com/charmbracelet/log"
)

var (
	Info  = log.New(os.Stdout, "[INFO] ", log.LstdFlags|log.Lshortfile)
	Error = log.New(os.Stderr, "[ERROR] ", log.LstdFlags|log.Lshortfile)
)

var Print *log1.Logger

func Init() {
	Print = log1.NewWithOptions(os.Stderr, log1.Options{
		//ReportCaller:    true,
		ReportTimestamp: true,
		TimeFormat:      time.DateTime,
		//Prefix:          "🍪",
	})
	styles := log1.DefaultStyles()
	styles.Levels[log1.InfoLevel] = lipgloss.NewStyle().
		SetString("INFOF🌟").
		Padding(0, 1, 0, 1).
		Background(lipgloss.Color("#90EE9080")).
		Foreground(lipgloss.Color("#006400FF")).Bold(true)

	styles.Levels[log1.ErrorLevel] = lipgloss.NewStyle().
		SetString("ERROR🔥").
		Padding(0, 1, 0, 1).
		Background(lipgloss.Color("#FF0000FF")).
		Foreground(lipgloss.Color("#00FFFF00")).Bold(true)

	styles.Levels[log1.FatalLevel] = lipgloss.NewStyle().
		SetString("FATAL⚡️").
		Padding(0, 1, 0, 1).
		Background(lipgloss.Color("#000000FF")).
		Foreground(lipgloss.Color("#00FFFF00")).Bold(true)

	styles.Levels[log1.WarnLevel] = lipgloss.NewStyle().
		SetString("Powered by yuuuuuuan🍪").
		Padding(0, 1, 0, 1).
		Background(lipgloss.Color("#000000FF")).
		Foreground(lipgloss.Color("#00FFFF00")).Bold(true)
	Print.SetStyles(styles)
}
