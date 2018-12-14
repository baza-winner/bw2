package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/baza-winner/bwcore/ansi"
	_ "github.com/baza-winner/bwcore/ansi/tags"
	"github.com/baza-winner/bwcore/bw"
	"github.com/baza-winner/bwcore/bwdebug"
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
      type OrderedMap
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
	var linkSourceFileSpec string
	if linkSourceFileSpec, err = bwos.ResolveSymlink(executableFileSpec); err != nil {
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

			projectDescription := []string{`<ansiVar>Сокращенное-имя-проекта<ansi> - одно из следующих значений:`}

			projectDefs.ForEach(func(idx int, projShortcut string, projDef bwval.Holder) (needBreak bool, err error) {
				projName := strings.TrimSuffix(filepath.Base(projDef.MustKey("gitOrigin").MustString()), ".git")
				// bwdebug.Print("!projName", "projName", projName)
				projDef.MustSetKeyVal("projName", projName)
				projectDescription = append(projectDescription,
					fmt.Sprintf(
						`      <ansiVal>%s<ansi> - сокращение для проекта <ansiVal>%s <ansiUrl>%s<ansi>`,
						projShortcut,
						projName,
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
							if err = conf.SetPathVal(
								map[string]interface{}{},
								bwval.MustPath(bwval.PathSS{SS: []string{"projects", projShortcut, projDir}}),
							); err != nil {
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
		type ProjDir struct {
			path     string
			fileInfo os.FileInfo
		}
		var projDirs []ProjDir
		var projDir ProjDir
		var expectsProjDirValue bool
		var projDirOption string
		for _, s := range os.Args[1:] {
			if !expectsProjDirValue {
				expectsProjDirValue = s == "--proj-dir" || s == "-p"
				projDirOption = s
			} else {
				if !strings.HasPrefix(s, "-") {
					if projDir.path, err = filepath.Abs(s); err != nil {
						return
					}
					if projDir.fileInfo, err = os.Stat(projDir.path); err != nil {
						err = bwerr.From(
							"<ansiVar>projDir <ansiPath>%s<ansi> specified by option <ansiCmd>%s<ansi> <ansiErr>does not exist",
							projDir.path,
							projDirOption,
						)
						return
					}
					expectsProjDirValue = false
				}
				break
			}
		}
		if expectsProjDirValue {
			err = bwerr.From("option <ansiCmd>%s<ansi> must have value", projDirOption)
		}

		bw2BinDir := path.Dir(executableFileSpec)
		bw2Dir := filepath.Clean(filepath.Join(bw2BinDir, ".."))
		var conf bwval.Holder
		if conf, err = Conf(bw2Dir); err != nil {
			return
		}
		projShortcut := executableFileName
		var needUpdateConf bool
		hProjDirs := conf.MustPath(bwval.PathSS{SS: []string{"projects", projShortcut}})
		_ = hProjDirs.MustKeys(func(key string) (ok bool) {
			if fi, err := os.Stat(key); err == nil {
				projDirs = append(projDirs, ProjDir{path: key, fileInfo: fi})
			} else {
				hProjDirs.DelKey(key)
				needUpdateConf = true
			}
			return
		})
		bwdebug.Print("executableFileName", executableFileName, "conf:json", conf.Val, "projDirs:json", projDirs)
		switch len(projDirs) {
		case 0:
			err = bwerr.From("<ansiCmd>%s<ansi> is not installed, use <ansiCmd>bw2 p %s<ansi> first", projShortcut, projShortcut)
		case 1:
			if projDir.path == "" {
				projDir = projDirs[0]
			} else if projDir.path != projDirs[0].path {
				if !os.SameFile(projDir.fileInfo, projDirs[0].fileInfo) {
					err = bwerr.From(
						"<ansiVar>projDir <ansiPath>%s<ansi> specified by option <ansiCmd>%s<ansi> differs from installed <ansiVar>projDir<ansi> (<ansiPath>%s<ansi>)",
						projDir,
						projDirOption,
						projDirs[0],
					)
					return
				}
			}
			bwdebug.Print("projDir", projDir)
		default:
			bwerr.TODO()
		}
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
