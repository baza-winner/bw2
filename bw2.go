package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"runtime"

	"github.com/baza-winner/bwcore/ansi"
	_ "github.com/baza-winner/bwcore/ansi/tags"
	"github.com/baza-winner/bwcore/bwos"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()

	app.Usage = ansi.String("Базовая утилита bw-инфраструктуры")
	app.Commands = []cli.Command{
		{
			Name:    "project",
			Aliases: []string{"p"},
			Usage:   "Разворачивает проект",
			Action: func(c *cli.Context) (err error) {
				if c.NArg() < 1 {
					bwos.Exit(1, "<ansiErr>ERR:<ansi> Ожидается <ansiVar>Сокращенное-имя-проекта")
				}
				var executable string
				if executable, err = os.Executable(); err != nil {
					return
				}
				bw2BinDir := path.Clean(path.Join(path.Dir(executable), ".."))
				_ = bw2BinDir
				// fmt.Println("GOOS: ", runtime.GOOS)
				// return
				projShortcut := c.Args()[0]
				switch projShortcut {
				case "dip":
					fmt.Printf(
						"%s => %s\n",
						path.Join(bw2BinDir, runtime.GOOS+"."+runtime.GOARCH, "bw2"),
						path.Join(bw2BinDir, runtime.GOOS+"."+runtime.GOARCH, projShortcut),
					)
					if err = os.Symlink(
						path.Join(bw2BinDir, runtime.GOOS+"."+runtime.GOARCH, "bw2"),
						path.Join(bw2BinDir, runtime.GOOS+"."+runtime.GOARCH, projShortcut),
					); err != nil {
						return
					}
				default:
					bwos.Exit(1, "<ansiErr>ERR:<ansi> Неизвестное <ansiVar>Сокращенное-имя-проекта <ansiVal>%s", projShortcut)
				}
				// fmt.Println(c.Args()[0])
				return
			},
			// Subcommands: []cli.Command{
			// 	{
			// 		// Category: "Основные команды",
			// 		Name:  "dip",
			// 		Usage: ansi.String("Разворачивает проект <ansiVal>dip2"),
			// 		// Flags: dockerUpFlags,
			// 		Action: func(c *cli.Context) (err error) {

			// 			return
			// 		},
			// 	},
			// },
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
