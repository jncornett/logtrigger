package main

type actionConfig struct {
	Cmd  string
	Args []string
}

type triggerConfig struct {
	Pattern string
	Action  actionConfig
}

type config struct {
	Root     string
	Triggers map[string][]triggerConfig
}
