package main

import (
	"os"
	"path"
	"path/filepath"

	_ "github.com/baza-winner/bwcore/ansi/tags"
	"github.com/baza-winner/bwcore/bwerr"
	"github.com/baza-winner/bwcore/bwexec"
	"github.com/baza-winner/bwcore/bwos"
	"github.com/baza-winner/bwcore/bwval"
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
	executableFileSpec string
	executableFileName string
	isInDocker         bool
	homeDir            string
)

func init() {
	var err error
	if err = doInit(); err != nil {
		bwos.Exit(1, bwerr.FromA(bwerr.Err(err)).JustError())
	}
}

func doInit() (err error) {
	if executableFileSpec, err = os.Executable(); err != nil {
		return
	}
	executableFileName = filepath.Base(executableFileSpec)
	if isInDocker, err = IsInDocker(); err != nil {
		return
	}
	if homeDir, err = filepath.Abs(os.Getenv("HOME")); err != nil {
		return
	}
	return
}

func run() (err error) {
	var isInDocker bool
	if isInDocker, err = IsInDocker(); err != nil {
		return
	}
	if executableFileName == bwFileName {
		err = runBw()
	} else if isInDocker {
		err = runProjShortcut(executableFileName, filepath.Join(os.Getenv("HOME"), "proj"))
	} else {
		projShortcut := executableFileName
		var sourceFileSpec string
		if sourceFileSpec, err = bwos.ResolveSymlink(executableFileSpec, 1); err != nil {
			return
		}
		var isSymlink bool
		if isSymlink, err = bwos.IsSymlink(sourceFileSpec); err != nil {
			return
		}
		var bwDir string
		{
			ss := []string{path.Dir(executableFileSpec), ".."}
			if !isSymlink {
				ss = append(ss, "..", "..")
			}
			bwDir = filepath.Clean(filepath.Join(ss...))
		}
		var projDir string
		var remainedArgs []string
		if projDir, remainedArgs, err = GetProjDir(projShortcut, bwDir); err != nil {
			return
		}
		if !isSymlink {
			err = runProjShortcut(projShortcut, projDir)
		} else {
			var projConf bwval.Holder
			if _, projConf, err = ProjConf(projDir); err != nil {
				return
			}
			bwTag := projConf.MustPath(bwval.PathS{S: "bw.tag"}).MustString(func() string { return "v1" })
			platformSpecificBinDir := filepath.Join(bwDir, bwTag, "bin", Platform())
			specificBwFileSpec := filepath.Join(platformSpecificBinDir, bwFileName)
			specificProjFileSpec := filepath.Join(platformSpecificBinDir, projShortcut)
			var exists bool
			if exists, err = bwos.Exists(specificProjFileSpec); err != nil {
				return
			}
			if !exists {
				if err = os.Symlink(
					specificBwFileSpec,
					specificProjFileSpec,
				); err != nil {
					return
				}
			}
			args := []string{"--proj-dir", projDir}
			args = append(args, remainedArgs...)
			_, err = bwexec.Cmd(
				bwexec.Args(specificProjFileSpec, args...),
				bwexec.MustCmdOpt(
					bwval.S{S: `{exitOnError true silent none verbosity all}`},
				),
			)
		}
	}

	return
}
