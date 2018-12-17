package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/baza-winner/bwcore/ansi"
	"github.com/baza-winner/bwcore/bwerr"
	"github.com/baza-winner/bwcore/bwjson"
	"github.com/baza-winner/bwcore/bwos"
	"github.com/baza-winner/bwcore/bwval"
	"github.com/urfave/cli"
)

func runBw() (err error) {
	bwBinDir := path.Dir(executableFileSpec)
	bwDir := filepath.Clean(filepath.Join(bwBinDir, ".."))
	app := cli.NewApp()
	app.Usage = ansi.String("Базовая утилита bw-инфраструктуры")

	var projectsDef bwval.Holder
	if projectsDef, err = ProjectsDef(); err != nil {
		return
	}

	projectDescription := []string{`<ansiVar>Сокращенное-имя-проекта<ansi> - одно из следующих значений:`}

	projectsDef.ForEach(func(idx int, projShortcut string, projDef bwval.Holder) (needBreak bool, err error) {
		projectDescription = append(projectDescription,
			fmt.Sprintf(
				`      <ansiVal>%s<ansi> - сокращение для проекта <ansiVal>%s <ansiUrl>%s<ansi>`,
				projShortcut,
				projDef.MustKeyVal("projName"),
				projDef.MustKeyVal("gitOrigin"),
			),
		)
		return
	})
	app.Commands = []cli.Command{
		{
			Name:    "project",
			Aliases: []string{"p"},
			Usage:   "Разворачивает проект",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "proj-dir, p",
					Usage: ansi.String(projDirUsage),
				},
			},
			ArgsUsage:   ansi.String("<ansiVar>Сокращенное-имя-проекта"),
			Description: ansi.String(strings.Join(projectDescription, "\n")),
			Action: func(c *cli.Context) (err error) {
				args := c.Args()
				if !args.Present() {
					err = bwerr.From("Ожидается <ansiVar>Сокращенное-имя-проекта")
					return
				}
				projShortcut := args.Get(0)
				if !projectsDef.HasKey(projShortcut) {
					err = bwerr.From("Неизвестное <ansiVar>Сокращенное-имя-проекта <ansiVal>%s", projShortcut)
					return
				}
				projDef := projectsDef.MustKey(projShortcut)
				projFileSpec := path.Join(bwBinDir, projShortcut)
				var exists bool
				if exists, err = bwos.Exists(projFileSpec); err != nil {
					return
				}
				if !exists {
					if err = os.Symlink(
						executableFileSpec,
						projFileSpec,
					); err != nil {
						return
					}
				}

				projDir := c.String("proj-dir")
				if projDir == "" {
					projDir = filepath.Join(homeDir, projDef.MustKey("projName").MustString())
				} else if projDir, err = filepath.Abs(projDir); err != nil {
					return
				}

				{
					var bwConf bwval.Holder
					var bwConfFileSpec string
					if bwConfFileSpec, bwConf, err = BwConf(bwDir); err != nil {
						return
					}
					if err = bwConf.SetPathVal(
						map[string]interface{}{},
						bwval.MustPath(bwval.PathSS{SS: []string{"projects", projShortcut, projDir}}),
					); err != nil {
						return
					}
					bwjson.ToFile(bwConfFileSpec, bwConf.Val)
				}
				return
			},
		},
	}
	err = app.Run(os.Args)
	return
}
