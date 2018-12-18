package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/baza-winner/bwcore/ansi"
	"github.com/baza-winner/bwcore/bwexec"
	"github.com/baza-winner/bwcore/bwos"
	"github.com/baza-winner/bwcore/bwosutil"
	"github.com/baza-winner/bwcore/bwval"
	"github.com/iancoleman/strcase"
	"github.com/urfave/cli"
)

func runProjShortcut(projShortcut string, projDir string) (err error) {
	app := cli.NewApp()

	// var projectsDef bwval.Holder
	// if projectsDef, err = ProjectsDef(); err != nil {
	// 	return
	// }
	projectsDef := BwTagConf().MustKey("projects")
	projName := projectsDef.MustPath(bwval.PathSS{SS: []string{projShortcut, "projName"}}).MustString()

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "proj-dir, p",
			Usage: ansi.String("`<ansiVar>Папка-проекта<ansi>`"),
		},
	}

	app.Usage = fmt.Sprintf(ansi.String("Базовая утилита проекта <ansiVal>%s"), projName)

	portFlags := []cli.Flag{
		cli.UintFlag{
			Value: 0,
			Name:  "port-increment, i",
			Usage: fmt.Sprintf(ansi.String(`Смещение относительно базовых значений для всех портов
      Полезно для старта второго экземпляра docker-приложения <ansiVal>dip<ansi>
      Примечание: второй экземпляр следует запускать из второй копии проекта, которую можно установить командой:
      <ansiCmd>bw p %s -p <ansiVar>Абсолютный-путь-к-папке-второй-копии-проекта<ansi>
      `+"`<ansiVar>Значение<ansi>` - неотрицательное целое число"+`
      <ansiVar>Значение<ansi> по умолчанию: 0
    `), projShortcut),
		},
	}
	type Port struct {
		name string
		base uint
	}
	ports := []Port{
		{name: "ssh", base: 2200},
		{name: "http", base: 8000},
		{name: "https", base: 4400},
		{name: "mysql", base: 3300},
		{name: "redis", base: 6300},
		{name: "webdis", base: 7300},
		{name: "rabbitmq", base: 5600},
		{name: "rabbitmq-management", base: 15600},
	}
	projBase := uint(8)
	for _, v := range ports {
		portFlags = append(portFlags,
			cli.UintFlag{
				Value: 0,
				Name:  v.name,
				Usage: fmt.Sprintf(ansi.String(`
      %s-порт по которому будет доступно docker-приложение
      `+"`<ansiVar>Значение<ansi>` - целое число из диапазона <ansiVal>1024..65535<ansi>"+`
      <ansiVar>Значение<ansi> по умолчанию: <ansiVar>%d + portIncrement
            `), v.name, v.base+projBase),
			},
		)
	}
	portFlags = append(portFlags,
		cli.UintFlag{
			Value: 3000,
			Name:  "upstream",
			Usage: fmt.Sprintf(ansi.String(`
        %s-порт по которому будет доступно docker-приложение
        `+"`<ansiVar>Значение<ansi>` - целое число из диапазона <ansiVal>1024..65535<ansi>"+`
        <ansiVar>Значение<ansi> по умолчанию: <ansiVal>%d
        `), "upstream", 3000),
		},
	)
	dockerUpFlags := []cli.Flag{
		cli.BoolFlag{
			Name:  "no-check, n",
			Usage: ansi.String("Не проверять актуальность docker-образа <ansiVal>bazawinner/dev-agate"),
		},
	}
	dockerUpFlags = append(dockerUpFlags, portFlags...)
	dockerUpFlags = append(dockerUpFlags,
		cli.BoolFlag{
			Name: "no-test-access-message, m",
			Usage: ansi.String(`
        Не выводить сообщение о проверке доступности docker-приложения`),
		},
		// projDirFlag,
		cli.BoolFlag{
			Name: "force-recreate, f",
			Usage: ansi.String(`
        Поднимает <ansiVal>dev-agate dev-agate-nginx<ansi> с опцией <ansiCmd>--force-recreate<ansi>, передаваемой <ansiCmd>docker-compose up`),
		},
	)
	dockerSubcommands := []cli.Command{
		{
			Category: "Основные команды",
			Name:     "up",
			Usage:    ansi.String("Поднимает (<ansiCmd>docker-compose up<ansi>) следующие контейнеры: <ansiVal>dev-dip dev-dip-nginx"),
			Flags:    dockerUpFlags,
			Action: func(c *cli.Context) (err error) {
				// dockerImageName := "dev-dip"
				// projShortcut := "dip"
				// var pwd string
				// if pwd, err = os.Getwd(); err != nil {
				//  return
				// }
				var executable string
				if executable, err = os.Executable(); err != nil {
					return
				}
				// fmt.Println(os.Getenv("HOME"))
				// return

				var dir string
				dir, err = filepath.Abs(filepath.Dir(executable))
				if err != nil {
					return
				}
				projDir := filepath.Clean(filepath.Join(dir, ".."))
				dockerDir := filepath.Join(projDir, "docker")

				dockerContainerEnvFileName := filepath.Join(dockerDir, "main.env")
				dockerImageName := "bazawinner/dev-" + projShortcut

				portIncrement := c.Uint("port-increment")
				actualPorts := map[string]string{}
				for _, port := range ports {
					portValue := c.Uint(port.name)
					if portValue == 0 {
						portValue = port.base + projBase + portIncrement
					}
					portName := strcase.ToLowerCamel(port.name)
					actualPorts[portName] = fmt.Sprintf("%d", portValue)
				}

				{
					buf := new(bytes.Buffer)
					// t := template.Must(template.New("main.env").Parse(mainEnv))
					if err = mainEnvTemplate.Execute(buf, map[string]interface{}{
						"projShortcut":          projShortcut,
						"_isBwDevelop":          true,
						"BW_SELF_UPDATE_SOURCE": "feature/perl",
						"projName":              projName,
						"whoami":                "yurybikuzin",
						"ports": func() string {
							var ss []string
							// portIncrement := c.Uint("port-increment")
							actualPorts := map[string]string{}
							for portName, portValue := range actualPorts {
								ss = append(ss, fmt.Sprintf("export _%s=%d", portName, portValue))
							}
							return strings.Join(ss, "\n")
						}(),
						"promptHolder": "`" + `_psPrepare_error` + "``" + `_psPrepare_git` + "`" + `\[\e[36m\e[1m\]dip \[\e[0m\]\[\e[97m\]\w \[\e[0m\]\[\e[90m\]` + "`" + `_psIf_git none ` + "\"" + `("` + "`" + `\[\e[0m\]\[\e[90m\]` + "`" + `_ps_gitBranch none` + "`" + `\[\e[0m\]\[\e[90m\]` + "`" + `_psIf_git after ")"` + "`" + `\[\e[0m\]\[\e[90m\]` + "`" + `_psIf_error none "\\\$?"` + "`" + `\[\e[0m\]\[\e[0m\]` + "`" + `_psIf_error none "="` + "`" + `\[\e[0m\]\[\e[31m\e[1m\]` + "`" + `_ps_errorCode after` + "`" + `\[\e[0m\]\[\e[97m\]\$ \[\e[0m\]`,
					}); err != nil {
						return
					}
					bytes := regexp.MustCompile("&#34;").ReplaceAll(buf.Bytes(), []byte(`"`))

					if err = ioutil.WriteFile(dockerContainerEnvFileName, bytes, 0644); err != nil {
						return
					}
				}

				mainContainerName := strings.Map(
					func(r rune) (result rune) {
						if 'a' <= r && r <= 'z' || 'A' <= r && r <= 'Z' || '0' <= r && r <= '9' || r == '.' || r == '-' {
							result = r
						} else {
							result = -1
						}
						return
					},
					"dev-"+bwos.ShortenFileSpec(projDir),
				)
				bwDir := path.Join(os.Getenv("HOME"), "bw")
				dockerComposeVars := map[string]string{
					"mainContainerName":    mainContainerName,
					"nginxContainerName":   mainContainerName + "-nginx",
					"mainImageName":        dockerImageName,
					"_bwDir":               bwDir,
					"_bwFileSpec":          path.Join(bwDir, "bw.bash"),
					"_bwNginxConfDir":      path.Join(bwDir, "docker", "nginx", "conf.bw"),
					"_bwSslFileSpecPrefix": path.Join(bwDir, "ssl", "server."),
					"upstream":             "3000",
				}
				for portName, portValue := range actualPorts {
					dockerComposeVars[portName] = portValue
				}
				{
					var args []string
					for _, s := range []string{"docker-compose.main", "docker-compose.nginx", "docker-compose.proj"} {
						targetFileSpec := filepath.Join(dockerDir, s)
						templateFileSpec := targetFileSpec + ".yml"
						bwosutil.CreateFileFromTemplate(targetFileSpec, templateFileSpec, dockerComposeVars)
						// bwdebug.Print("s", s)
						// var file *os.File
						// if file, err = os.Create(filepath.Join(dockerDir, s)); err != nil {
						//  return
						// }
						// defer func() {
						//  e := file.Close()
						//  if err == nil {
						//    err = e
						//  }
						// }()
						// w := bufio.NewWriter(file)
						// p := bwparse.MustFrom(bwrune.F{filepath.Join(dockerDir, s+".yml")})
						// p.Forward(bwparse.Initial)

						// for !p.Curr().IsEOF() {
						//  for !p.Curr().IsEOF() && (p.Curr().Rune() != '$' || p.LookAhead(1).Rune() != '{') {
						//    w.WriteRune(p.Curr().Rune())
						//    p.Forward(1)
						//  }
						//  if !p.Curr().IsEOF() {
						//    p.Forward(2)
						//    start := p.Start()
						//    defer p.Stop(start)
						//    var id string
						//    for !p.Curr().IsEOF() && p.Curr().Rune() != '}' {
						//      id += string(p.Curr().Rune())
						//      p.Forward(1)
						//    }
						//    if p.Curr().IsEOF() {
						//      err = bwparse.Unexpected(p)
						//      return
						//    }
						//    if s, ok := dockerComposeVars[id]; ok {
						//      w.WriteString(s)
						//    } else {
						//      err = p.Error(bwparse.E{Start: start, Fmt: bw.Fmt("unexpected var <ansiVar>%s<ansi>", id)})
						//      return
						//    }
						//    p.Forward(1)
						//  }
						// }
						// if err = w.Flush(); err != nil {
						//  return
						// }

						args = append(args, "-f", s)
					}
					args = append(args, "config")
					cmdResult := bwexec.MustCmd(
						bwexec.Args("docker-compose", args...),
						bwexec.MustCmdOpt(bwval.S{
							Vars: map[string]interface{}{"dockerDir": dockerDir},
							S: `{
                    verbosity "err"
                    exitOnError true
                    silent "stdout"
                    workDir $dockerDir
                    captureStdout true
                  }`,
						}),
					)
					if err = ioutil.WriteFile(
						filepath.Join(dockerDir, "docker-compose.yml"),
						[]byte(strings.Join(cmdResult.Stdout, "\n")),
						0644,
					); err != nil {
						return
					}
				}

				{
					args := []string{"up", "-d", "--remove-orphans"}
					if c.Bool("force-recreate") {
						args = append(args, "--force-recreate")
					}
					bwexec.MustCmd(
						bwexec.Args("docker-compose", args...),
						bwexec.MustCmdOpt(bwval.S{
							Vars: map[string]interface{}{"dockerDir": dockerDir},
							S: `{
                    verbosity "err"
                    exitOnError true
                    silent "none"
                    workDir $dockerDir
                  }`,
						}),
					)
				}

				{
					args := []string{"exec", "-T", "main", "sudo", "usermod", "-u", fmt.Sprintf("%d", os.Geteuid()), "dev"}
					bwexec.MustCmd(
						bwexec.Args("docker-compose", args...),
						bwexec.MustCmdOpt(bwval.S{
							Vars: map[string]interface{}{"dockerDir": dockerDir},
							S: `{
                    verbosity "err"
                    exitOnError true
                    silent "all"
                    workDir $dockerDir
                  }`,
						}),
					)
				}

				// if
				{
					whoamiDir := path.Join(dockerDir, "nginx", "whoami")
					os.MkdirAll(whoamiDir, 0666)

				}

				return
			},
		},
		{
			Category: "Основные команды",
			Name:     "shell",
			Usage:    ansi.String("Запускает bash в docker-контейнере"),
			// Flags: []cli.Flag{
			//  projDirFlag,
			// },
			ArgsUsage: ansi.String("[ <ansiVar>Имя-сервиса<ansi> [<ansiVar>Командная-оболочка<ansi>]]"),
			Action: func(c *cli.Context) error {
				fmt.Println("shell")
				return nil
			},
		},
		{
			Category: "Основные команды",
			Name:     "down",
			Usage:    ansi.String("Останавливает (<ansiCmd>docker-compose down<ansi>) следующие контейнеры: <ansiVal>dev-dip dev-dip-nginx"),
			// Flags: []cli.Flag{
			//  projDirFlag,
			// },
			Action: func(c *cli.Context) error {
				fmt.Println("down")
				return nil
			},
		},
		{
			Category: "Специальные команды",
			Name:     "build",
			Usage:    ansi.String("Собирает docker-образ <ansiVal>bazawinner/dev-dip"),
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "force, f",
					Usage: ansi.String("Невзирая на возможное отсутствие изменений в docker/Dockerfile"),
				},
				// projDirFlag,
			},
			Action: func(c *cli.Context) error {
				fmt.Println("build image")
				return nil
			},
		},
		{
			Category: "Специальные команды",
			Name:     "push",
			Usage:    ansi.String("Push-ит docker-образ <ansiVal>bazawinner/dev-dip"),
			Action: func(c *cli.Context) error {
				fmt.Println("push image")
				return nil
			},
		},
	}
	app.Commands = []cli.Command{
		{
			Name:        "docker",
			Usage:       "docker-операции",
			Subcommands: dockerSubcommands,
		},
		{
			Name:    "self-test",
			Aliases: []string{"st"},
			Usage:   "Самопроверка",
			// Flags:   selfTestFlags,
			Action: func(c *cli.Context) error {
				fmt.Println("self-test")
				return nil
			},
		},
		// {
		//  Name:  "update",
		//  Usage: ansi.String("Обновляет команду <ansiCmd>dip"),
		//  Action: func(c *cli.Context) error {
		//    fmt.Println("update")
		//    return nil
		//  },
		// },
	}
	err = app.Run(os.Args)
	return
}

const mainEnv = `#!/bin/bash
# file generated by {{.projShortcut}}_docker_up
export _isBwDevelopInherited={{._isBwDevelop}}
export BW_SELF_UPDATE_SOURCE={{.BW_SELF_UPDATE_SOURCE}}
export _bwProjName={{.projName}}
export _bwProjShortcut={{.projShortcut}}
export _hostUser={{.whoami}}
{{.ports}}
export _prompt='{{.promptHolder}}'
`

const whoami = `<pre>
projname: {{.projName}}
`

var mainEnvTemplate *template.Template

func init() {
	mainEnvTemplate = template.Must(template.New("main.env").Parse(mainEnv))
}
