package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"dockdash/internal/app"
	"dockdash/internal/docker"

	tea "github.com/charmbracelet/bubbletea"
)

const version = "1.0.0"

func main() {
	demoFlag := flag.Bool("demo", false, "Запуск в демо-режиме (без Docker)")
	versionFlag := flag.Bool("v", false, "Показать версию программы")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("DockDash v%s\n", version)
		os.Exit(0)
	}

	var service docker.Service

	if *demoFlag {
		service = docker.NewMockClient()
	} else {
		// Пробуем подключиться к реальному Docker, если не вышло — включаем демо
		realCli, err := docker.NewRealClient()
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			err = realCli.Ping(ctx)
			cancel()
		}

		if err == nil {
			service = realCli
		} else {
			service = docker.NewMockClient()
		}
	}

	// Запуск интерфейса в чистом окне терминала
	p := tea.NewProgram(
		app.New(service),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка запуска: %v\n", err)
		os.Exit(1)
	}
}
