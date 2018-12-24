package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/baza-winner/bwcore/ansi"
	"github.com/baza-winner/bwcore/bwerr"
	"github.com/baza-winner/bwcore/bwjson"
	"github.com/baza-winner/bwcore/bwos"
	"github.com/baza-winner/bwcore/bwrune"
	"github.com/baza-winner/bwcore/bwstr"
	"github.com/baza-winner/bwcore/bwval"
)

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

func GetProjDir(projShortcut, bwDir string) (result string, remainedArgs []string, err error) {
	var projDirs ProjDirs
	var projDir ProjDir
	var expectsProjDirValue bool
	var projDirOption string
	for _, s := range os.Args[1:] {
		if !expectsProjDirValue {
			if expectsProjDirValue = s == "--proj-dir" || s == "-p"; !expectsProjDirValue {
				remainedArgs = append(remainedArgs, s)
			} else if projDir.path == "" {
				projDirOption = s
			} else {
				err = bwerr.From(
					"<ansiVar>projDir<ansi> can not be specified again by option <ansiCmd>%s<ansi> (was already specified as <ansiCmd>%s<ansi> <ansiPath>%s<ansi>)",
					s,
					projDirOption,
					projDir.path,
				)
			}
		} else if !strings.HasPrefix(s, "-") {
			if projDir.path, err = filepath.Abs(s); err != nil {
				return
			}
			var exists bool
			if exists, err = bwos.Exists(projDir.path); err != nil {
				return
			}
			if !exists {
				err = bwerr.From(
					"<ansiVar>projDir <ansiPath>%s<ansi> specified by option <ansiCmd>%s<ansi> <ansiErr>does not exist",
					projDir.path,
					projDirOption,
				)
				return
			}
			expectsProjDirValue = false
		} else {
			err = bwerr.From(
				"option <ansiCmd>%s<ansi> expects value, but found option <ansiCmd>%s",
				projDirOption, s,
			)
			return
		}
	}

	if expectsProjDirValue {
		err = bwerr.From("option <ansiCmd>%s<ansi> must have value", projDirOption)
	}

	var bwConf bwval.Holder
	var bwConfFileSpec string
	bwConfFileSpec, bwConf = BwConf(bwDir)
	var needUpdateConf bool
	// bwdebug.Print("bwConf.Pth", bwConf.Pth)
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

var bwTagDir string

func BwTagDir() string {
	if len(bwTagDir) == 0 {
		if linkSourceFileSpec, err := bwos.ResolveSymlink(executableFileSpec); err != nil {
			bwerr.PanicErr(err)
		} else {
			bwTagDir = filepath.Clean(filepath.Join(filepath.Dir(linkSourceFileSpec), "..", ".."))
		}
	}
	return bwTagDir
}

var bwTagConf *bwval.Holder

func BwTagConf() (result bwval.Holder) {
	if bwTagConf == nil {
		var h bwval.Holder
		bwTagDir := BwTagDir()
		h = bwval.MustFrom(
			bwval.F{S: filepath.Join(bwTagDir, "data", "conf.jlf")},
			// bwval.O{Def: bwval.MustDefFrom(bwrune.F{S: filepath.Join(bwTagDir, "data", "conf.jld")})},
		)
		_ = h.MustKey("projects").ForEach(func(idx int, projShortcut string, projDef bwval.Holder) (needBreak bool, err error) {
			projName := strings.TrimSuffix(filepath.Base(projDef.MustKey("gitOrigin").MustString()), ".git")
			projDef.MustSetKeyVal("projName", projName)
			return
		})
		_ = h.MustKey("services").ForEach(func(idx int, serviceName string, serviceDef bwval.Holder) (needBreak bool, err error) {
			serviceDef.MustKey("ports").ForEach(func(idx int, portName string, portValue bwval.Holder) (needBreak bool, err error) {
				if portName == "_" {
					portName = serviceName
				}
				bwval.MustSetPathVal(portValue.MustInt(), &h, bwval.PathSS{SS: []string{"availPorts", portName}})
				return
			})
			return
		})
		bwTagConf = &h
	}
	result = *bwTagConf
	return
}

var projConfs map[string]bwval.Holder

// func ProjConfFileSpec(projDir string) string { return filepath.Join(projDir, "docker", "conf.jlf") }

func ProjConf(projDir string) (result bwval.Holder) {
	if projConfs == nil {
		projConfs = map[string]bwval.Holder{}
	}
	var ok bool
	if result, ok = projConfs[projDir]; !ok {
		result = bwval.MustFrom(
			bwval.F{S: filepath.Join(projDir, "docker", "conf.jlf")},
			bwval.O{Def: bwval.MustDefFrom(bwrune.F{S: filepath.Join(BwTagDir(), "data", "proj.conf.jld")})},
			// bwval.O{Def: bwval.MustDefFrom(bwrune.F{S: filepath.Join(projDir, "docker", "conf.jld")})},
		)
		projConfs[projDir] = result
	}
	return
}

func BwConf(bwDir string) (fileSpec string, result bwval.Holder) {
	var (
		confFromProvider bwval.FromProvider
		confFromOpt      []bwval.O
	)
	fileSpec = filepath.Join(bwDir, "conf.json")
	var exists bool
	var err error
	if exists, err = bwos.Exists(fileSpec); err != nil {
		bwerr.PanicErr(err)
	}
	if exists {
		confFromProvider = bwval.F{S: fileSpec}
	} else {
		confFromProvider = bwval.V{Val: map[string]interface{}{}}
		confFromOpt = append(confFromOpt, bwval.O{PathProvider: bwval.PathS{S: "$BwConf"}})
	}
	result = bwval.MustFrom(confFromProvider, confFromOpt...)
	return
}

func IsInDocker() (ok bool, err error) {
	return bwos.Exists("/.dockerenv")
}

func Platform() string {
	return runtime.GOOS + "." + runtime.GOARCH
}
