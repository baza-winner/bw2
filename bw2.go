package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/baza-winner/bwcore/ansi"
	_ "github.com/baza-winner/bwcore/ansi/tags"
	"github.com/baza-winner/bwcore/bw"
	"github.com/baza-winner/bwcore/bwerr"
	"github.com/baza-winner/bwcore/bwjson"
	"github.com/baza-winner/bwcore/bwos"
	"github.com/baza-winner/bwcore/bwrune"
	"github.com/baza-winner/bwcore/bwval"
	"github.com/urfave/cli"
)

func main() {
	if err := run(); err != nil {
		bwos.Exit(1, bwerr.FromA(bwerr.Err(err)).JustError())
	}
}

const (
	projDirUsage = "`<ansiVar>Путь-к-папке<ansi>`" + `, куда будет установлен проект
      По умолчанию, в качестве папки проекта используется ~/<ansiVar>Полное-имя-проекта<ansi>
      <ansiVar>Полное-имя-проекта<ansi> - имя проекта на github'е`
)

var (
	projectDefsDef     *bwval.Def
	executableFileSpec string
	executableFileName string
)

func init() {
	projectDefsDef = bwval.MustDefFrom(bwrune.S{S: `
    {
      type Map
      elem {
        type Map
        keys {
          gitOrigin String
          branch String
        }
      }
    }
  `})
	var err error
	if executableFileSpec, err = os.Executable(); err != nil {
		bwos.Exit(1, bwerr.FromA(bwerr.Err(err)).JustError())
	}
	executableFileName = filepath.Base(executableFileSpec)
}

func VerDir() (result string, err error) {
	var executableFileSpec string
	if executableFileSpec, err = os.Executable(); err != nil {
		return
	}
	var linkSourceFileSpec string
	if linkSourceFileSpec, err = os.Readlink(executableFileSpec); err != nil {
		return
	}
	result = filepath.Clean(filepath.Join(filepath.Dir(linkSourceFileSpec), "..", ".."))
	return
}

func ProjectDefs() (result bwval.Holder, err error) {
	var verDir string
	if verDir, err = VerDir(); err != nil {
		return
	}
	return bwval.From(
		bwval.F{
			S:   filepath.Join(verDir, "project.defs"),
			Def: projectDefsDef,
		},
	)
}

func run() (err error) {
	if executableFileName == "bw2" {
		var homeDir string
		if homeDir, err = filepath.Abs(os.Getenv("HOME")); err != nil {
			return
		}
		bw2HolderDir := filepath.Clean(filepath.Join(path.Dir(executableFileSpec), "..", ".."))

		if bw2HolderDir == homeDir {
			bw2BinDir := path.Dir(executableFileSpec)
			bw2Dir := filepath.Clean(filepath.Join(bw2BinDir, ".."))
			app := cli.NewApp()
			app.Usage = ansi.String("Базовая утилита bw-инфраструктуры")

			var projectDefs bwval.Holder
			if projectDefs, err = ProjectDefs(); err != nil {
				return
			}

			var projShortcuts []string
			for projShortcut, _ := range projectDefs.MustMap() {
				projShortcuts = append(projShortcuts, projShortcut)
				projDef := projectDefs.MustKey(projShortcut)
				projName := strings.TrimSuffix(filepath.Base(projDef.MustKey("gitOrigin").MustString()), ".git")
				projectDefs.MustKey(projShortcut).MustSetKeyVal("projName", projName)
			}
			sort.Strings(projShortcuts)
			projectDescription := []string{`<ansiVar>Сокращенное-имя-проекта<ansi> - одно из следующих значений:`}
			for _, projShortcut := range projShortcuts {
				projDef := projectDefs.MustKey(projShortcut)
				projectDescription = append(projectDescription,
					fmt.Sprintf(
						`      <ansiVal>%s<ansi> - сокращение для проекта <ansiVal>%s <ansiUrl>%s`,
						projShortcut,
						projDef.MustKeyVal("projName"),
						projDef.MustKeyVal("gitOrigin"),
					),
				)
			}
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
						if !projectDefs.HasKey(projShortcut) {
							err = bwerr.From("Неизвестное <ansiVar>Сокращенное-имя-проекта <ansiVal>%s", projShortcut)
							return
						}
						projDef := projectDefs.MustKey(projShortcut)
						projBinFileSpec := path.Join(bw2BinDir, projShortcut)
						if _, e := os.Lstat(projBinFileSpec); e != nil {
							if err = os.Symlink(
								executableFileSpec,
								path.Join(bw2BinDir, projShortcut),
							); err != nil {
								return
							}
							fmt.Printf(
								"%s => %s\n",
								path.Join(bw2BinDir, "bw2"),
								path.Join(bw2BinDir, projShortcut),
							)
						}
						projDir := c.String("proj-dir")
						if projDir == "" {
							projDir = filepath.Join(homeDir, projDef.MustKey("projName").MustString())
						} else if projDir, err = filepath.Abs(projDir); err != nil {
							return
						}

						{
							confFileSpec := filepath.Join(bw2Dir, ".conf")
							var conf bwval.Holder
							if conf, err = Conf(bw2Dir); err != nil {
								return
							}

							if err = conf.SetPathVal(map[string]interface{}{}, bwval.MustPath(bwval.PathSS{SS: []string{"projects", projShortcut, projDir}})); err != nil {
								bwerr.PanicErr(err)
								return
							}

							bwjson.ToFile(confFileSpec, conf.Val)
						}
						return
					},
				},
			}
			err = app.Run(os.Args)
		} else {

		}
	} else {
		// bw2BinDir := path.Dir(executableFileSpec)
		// bw2Dir := filepath.Clean(filepath.Join(bw2BinDir, ".."))
		// ProjectDefs
		fmt.Println(executableFileName)
	}
	return
}

func ConfFileSpec(bw2Dir string) (result string) {
	return filepath.Join(bw2Dir, ".conf")
}

func Conf(bw2Dir string) (result bwval.Holder, err error) {
	var (
		confFromProvider     bwval.FromProvider
		confValPathProviders []bw.ValPathProvider
	)
	confFileSpec := ConfFileSpec(bw2Dir)
	if _, e := os.Stat(confFileSpec); e == nil {
		confFromProvider = bwval.F{S: confFileSpec}
	} else {
		confFromProvider = bwval.V{Val: map[string]interface{}{}}
		confValPathProviders = append(confValPathProviders, bwval.PathS{S: "$conf"})
	}
	result, err = bwval.From(confFromProvider, confValPathProviders...)
	return
}
