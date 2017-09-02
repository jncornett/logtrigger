package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"

	"github.com/hpcloud/tail"
	"github.com/jncornett/patmatch"
	log "github.com/sirupsen/logrus"
)

type stopper interface {
	Stop() error
}

func logTrigger(filename string, triggers []Trigger) (stopper, error) {
	t, err := tail.TailFile(filename, tail.Config{
		Follow:   true,
		Location: &tail.SeekInfo{Offset: 0, Whence: os.SEEK_END},
	})
	if err != nil {
		return nil, err
	}
	logger := log.WithField("filename", filename)
	go func() {
		for line := range t.Lines {
			if line.Err != nil {
				logger.WithField("error", line.Err).Info("Caught tail error")
				continue
			}
			logger.WithField("line", line.Text).Debug("processing line")
			for _, trigger := range triggers {
				result := trigger.Apply(line.Text)
				if len(result) == 0 {
					continue
				}
				err := trigger.Act(result)
				if err != nil {
					logger.WithFields(log.Fields{
						"error":  err,
						"action": trigger.Action,
					}).Error("Caught action error")
				}
			}
		}
	}()
	return t, nil
}

const (
	defaultConfigFileName = "logtrigger.json"
)

func main() {
	u, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	defaultConfigPath := path.Join(u.HomeDir, defaultConfigFileName)
	var (
		configPath = flag.String("config", defaultConfigPath, "configuration file path")
		debugMode  = flag.Bool("debug", false, "debug mode")
	)
	flag.Parse()
	if *debugMode {
		log.SetLevel(log.DebugLevel)
	}
	raw, err := ioutil.ReadFile(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	var conf config
	err = json.Unmarshal(raw, &conf)
	if err != nil {
		log.Fatal(err)
	}
	if conf.Root == "" {
		conf.Root = fmt.Sprintf("%c", os.PathSeparator)
	}
	for pathName, triggerConfs := range conf.Triggers {
		lg := log.WithField("pathName", pathName)
		lg.Info("setting up watch")
		var triggers []Trigger
		for _, c := range triggerConfs {
			if c.Pattern == "" {
				lg.Fatal("Pattern must not be empty")
			}
			filter, err := patmatch.Parse(c.Pattern)
			if err != nil {
				lg.WithError(err).WithField("pattern", c.Pattern).Fatal("could not compile pattern")
			}
			var action Action
			if c.Action.Cmd != "" {
				action = NewShellAction(c.Action.Cmd)
			} else if len(c.Action.Args) != 0 {
				action = NewShellAction(c.Action.Args...)
			} else {
				lg.WithField("pattern", c.Pattern).Warn("skipping trigger with no action")
				continue
			}
			triggers = append(triggers, Trigger{
				Filter: filter,
				Action: action,
			})
		}
		if len(triggers) == 0 {
			lg.Warn("skipping watching file with no triggers")
			continue
		}
		stop, err := logTrigger(pathName, triggers)
		if err != nil {
			log.Fatal(err)
		}
		defer func() {
			err := stop.Stop()
			if err != nil {
				lg.WithError(err).Warn("caught error while stopping watch")
			}
		}()
	}
	waitForever := make(chan bool)
	<-waitForever
}
