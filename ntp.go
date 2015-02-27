package main

import (
	"github.com/codegangsta/cli"
	mp "github.com/mackerelio/go-mackerel-plugin"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var graphdef map[string](mp.Graphs) = map[string](mp.Graphs){}

type NtpPlugin struct {
	DriftFile string
}

func (c NtpPlugin) GraphDefinition() map[string](mp.Graphs) {
	graphdef["ntp.drift"] = mp.Graphs{
		Label: "Drift",
		Unit:  "float",
		Metrics: [](mp.Metrics){
			mp.Metrics{Name: "drift", Label: "Drift", Diff: false},
		},
	}
	return graphdef
}

func (c NtpPlugin) FetchMetrics() (map[string]float64, error) {
	var err error
	p := make(map[string]float64)

	err = getDrift(c.DriftFile, &p)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func getDrift(path string, p *map[string]float64) error {
	value, err := exec.Command("cat", path).Output()
	if err != nil {
		return err
	}
	drift, err2 := strconv.ParseFloat(strings.TrimSpace(string(value[:])), 64)
	if err2 != nil {
		return err2
	}
	(*p)["drift"] = drift
	return nil
}

var Flags = []cli.Flag{
	cliDriftFile,
}

var cliDriftFile = cli.StringFlag{
	Name:   "driftfile",
	Value:  "/var/lib/ntp/ntp.drift",
	EnvVar: "ENVVAR_DRIFTFILE",
}

func doMain(c *cli.Context) {
	var ntp NtpPlugin
	ntp.DriftFile = c.String("driftfile")
	helper := mp.NewMackerelPlugin(ntp)

	if os.Getenv("MACKEREL_AGENT_PLUGIN_META") != "" {
		helper.OutputDefinitions()
	} else {
		helper.OutputValues()
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "mackerel-plugin-ntp"
	app.Version = "0.0.0"
	app.Author = "Yuichiro Hanada"
	app.Email = "i@chir.jp"
	app.Flags = Flags
	app.Action = doMain
	app.Run(os.Args)
}
