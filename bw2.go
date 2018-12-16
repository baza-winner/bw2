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
	"github.com/baza-winner/bwcore/bwstr"
	"github.com/baza-winner/bwcore/bwval"
	"github.com/urfave/cli"
)

func main() {
	if err := run(); err != nil {
		bwos.Exit(1, bwerr.FromA(bwerr.Err(err)).JustError())
	}
}

const (
	bwFileName   = "bw2"
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
	if executableFileName == bwFileName {
		var homeDir string
		if homeDir, err = filepath.Abs(os.Getenv("HOME")); err != nil {
			return
		}
		bwHolderDir := filepath.Clean(filepath.Join(path.Dir(executableFileSpec), "..", ".."))
		// bwdebug.Print("bwHolderDir", bwHolderDir)

		if bwHolderDir == homeDir {
			bwBinDir := path.Dir(executableFileSpec)
			bwDir := filepath.Clean(filepath.Join(bwBinDir, ".."))
			app := cli.NewApp()
			app.Usage = ansi.String("Базовая утилита bw-инфраструктуры")

			var projectDefs bwval.Holder
			if projectDefs, err = ProjectDefs(); err != nil {
				return
			}

			projectDescription := []string{`<ansiVar>Сокращенное-имя-проекта<ansi> - одно из следующих значений:`}

			projectDefs.ForEach(func(idx int, projShortcut string, projDef bwval.Holder) (needBreak bool, err error) {
				projName := strings.TrimSuffix(filepath.Base(projDef.MustKey("gitOrigin").MustString()), ".git")
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
						projBinFileSpec := path.Join(bwBinDir, projShortcut)
						if _, e := os.Lstat(projBinFileSpec); e != nil {
							if err = os.Symlink(
								executableFileSpec,
								projBinFileSpec,
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
		} else {

		}
	} else {
		var projDir string
		if projDir, err = GetProjDir(executableFileName); err != nil {
			return
		}
		var projConf bwval.Holder
		if _, projConf, err = ProjConf(projDir); err != nil {
			return
		}
		bwdebug.Print(
			"projDir", projDir,
			"version", projConf.MustPath(bwval.PathS{S: "bw.version"}).MustUint(func() uint { return 1 }),
		)
	}
	return
}

type ProjDir struct {
	path     string
	fileInfo os.FileInfo
}

type ProjDirs []ProjDir

func (v ProjDirs) Strings() (result []string) {
	for _, pd := range v {
		result = append(result,
			fmt.Sprintf(ansi.String("<ansiPath>%s"), bwos.ShortenFileSpec(pd.path)),
		)
	}
	return
}

func GetProjDir(projShortcut string) (result string, err error) {
	var projDirs ProjDirs
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

	bwBinDir := path.Dir(executableFileSpec)
	bwDir := filepath.Clean(filepath.Join(bwBinDir, ".."))
	var bwConf bwval.Holder
	var bwConfFileSpec string
	if bwConfFileSpec, bwConf, err = BwConf(bwDir); err != nil {
		return
	}
	var needUpdateConf bool
	hProjDirs := bwConf.MustPath(bwval.PathSS{SS: []string{"projects", projShortcut}})
	_ = hProjDirs.MustKeys(func(key string) (ok bool) {
		if fi, err := os.Stat(key); err == nil {
			projDirs = append(projDirs, ProjDir{path: key, fileInfo: fi})
		} else {
			hProjDirs.DelKey(key)
			needUpdateConf = true
		}
		return
	})
	if needUpdateConf {
		bwjson.ToFile(bwConfFileSpec, bwConf.Val)
	}
	switch len(projDirs) {
	case 0:
		err = bwerr.From("<ansiCmd>%s<ansi> is not installed, use <ansiCmd>%s p %s<ansi> first", projShortcut, bwFileName, projShortcut)
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
	default:
		if projDir.path == "" {
			var pwd string
			if pwd, err = os.Getwd(); err != nil {
				return
			}
			for _, pd := range projDirs {
				var ok bool
				if ok, err = bwos.IsInPath(pd.path, pwd); err != nil {
					return
				} else if ok {
					projDir = pd
					break
				}
			}
			if projDir.path == "" {
				err = bwerr.From(
					"must specify <ansiCmd>--proj-dir<ansi> as %s",
					bwstr.SmartJoin(bwstr.A{
						Source: projDirs,
					}),
				)
				return
			}
		}
	}
	result = projDir.path
	return
}

func BwConf(bwDir string) (fileSpec string, result bwval.Holder, err error) {
	var (
		confFromProvider     bwval.FromProvider
		confValPathProviders []bw.ValPathProvider
	)
	fileSpec = filepath.Join(bwDir, "conf.json")
	if _, e := os.Stat(fileSpec); e == nil {
		confFromProvider = bwval.F{S: fileSpec}
	} else {
		confFromProvider = bwval.V{Val: map[string]interface{}{}}
		confValPathProviders = append(confValPathProviders, bwval.PathS{S: "$BwConf"})
	}
	result, err = bwval.From(confFromProvider, confValPathProviders...)
	return
}

func ProjConf(projDir string) (fileSpec string, result bwval.Holder, err error) {
	var (
		confFromProvider     bwval.FromProvider
		confValPathProviders []bw.ValPathProvider
	)
	fileSpec = filepath.Join(projDir, "docker", "conf.json")
	if _, e := os.Stat(fileSpec); e == nil {
		confFromProvider = bwval.F{S: fileSpec}
	} else {
		confFromProvider = bwval.S{S: "{ bw { version 1 } }"}
		confValPathProviders = append(confValPathProviders, bwval.PathS{S: "$ProjConf"})
	}
	result, err = bwval.From(confFromProvider, confValPathProviders...)
	return
}
