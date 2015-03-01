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
		Label: "NTP Drift",
		Unit:  "float",
		Metrics: [](mp.Metrics){
			mp.Metrics{Name: "drift", Label: "NTP Drift", Diff: false},
		},
	}

	// Add peer information
	peers, err := getPeerInformation()
	peerMetrics := make([]mp.Metrics, 0, len(peers))
	if err == nil {
		for _, peer := range peers {
			peerMetrics = append(peerMetrics, mp.Metrics{Name: peer.host + ".delay", Label: peer.host + " Delay", Diff: false})
			peerMetrics = append(peerMetrics, mp.Metrics{Name: peer.host + ".offset", Label: peer.host + " Offset", Diff: false})
			peerMetrics = append(peerMetrics, mp.Metrics{Name: peer.host + ".jitter", Label: peer.host + " Jitter", Diff: false})
		}
	}
	graphdef["ntp.peers"] = mp.Graphs{
		Label:   "NTP Peers",
		Unit:    "float",
		Metrics: peerMetrics,
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
	var peers []NtpPeerInfo
	peers, err = getPeerInformation()
	if err != nil {
		return nil, err
	}
	for _, peer := range peers {
		p[peer.host+".delay"] = peer.delay
		p[peer.host+".offset"] = peer.offset
		p[peer.host+".jitter"] = peer.jitter
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

type NtpPeerInfo struct {
	host   string
	delay  float64
	offset float64
	jitter float64
}

func getPeerInformation() ([]NtpPeerInfo, error) {
	value, err := exec.Command("ntpq", "-p").Output()
	if err != nil {
		return nil, err
	}
	// cut first 2 lines
	lines := strings.Split(strings.TrimRight(string(value[:]), "\n"), "\n")
	lines = lines[2:]
	peers := make([]NtpPeerInfo, 0, len(lines))
	for _, line := range lines {
		values := strings.Fields(line[1:])
		peer, err := makeNtpPeerInfo(values)
		if err != nil {
			return nil, err
		}
		peers = append(peers, peer)
	}
	return peers, nil
}

func makeNtpPeerInfo(l []string) (NtpPeerInfo, error) {
	host := l[0]
	delay, err := strconv.ParseFloat(l[7], 64)
	if err != nil {
		return NtpPeerInfo{}, err
	}
	offset, err := strconv.ParseFloat(l[8], 64)
	if err != nil {
		return NtpPeerInfo{}, err
	}
	jitter, err := strconv.ParseFloat(l[9], 64)
	if err != nil {
		return NtpPeerInfo{}, err
	}
	return NtpPeerInfo{host, delay, offset, jitter}, nil
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
