package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/sensu-community/sensu-plugin-sdk/sensu"
	"github.com/sensu/sensu-go/types"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Config represents the check plugin config.
type Config struct {
	sensu.PluginConfig
	LogPath   string
	LogRegex  string
	Match     string
	StateFile string
	NumProcs  int
}

var (
	plugin = Config{
		PluginConfig: sensu.PluginConfig{
			Name:     "sensu-check-log-regex",
			Short:    "Check Log Regex",
			Keyspace: "sensu.io/plugins/sensu-check-log-regex/config",
		},
	}

	options = []*sensu.PluginConfigOption{
		&sensu.PluginConfigOption{
			Path:      "logpath",
			Env:       "CHECK_LOG_PATH",
			Argument:  "logpath",
			Shorthand: "l",
			Default:   "",
			Usage:     "Path of logs to examine",
			Value:     &plugin.LogPath,
		},
		&sensu.PluginConfigOption{
			Path:      "logregex",
			Env:       "CHECK_LOG_REGEX",
			Argument:  "logregex",
			Shorthand: "r",
			Default:   "",
			Usage:     "Regex of log names to examine",
			Value:     &plugin.LogRegex,
		},
		&sensu.PluginConfigOption{
			Path:      "match",
			Env:       "CHECK_MATCH",
			Argument:  "match",
			Shorthand: "m",
			Default:   "",
			Usage:     "Keyword to match in logs",
			Value:     &plugin.Match,
		},
		&sensu.PluginConfigOption{
			Path:      "statefile",
			Env:       "CHECK_STATE_FILE",
			Argument:  "state",
			Shorthand: "s",
			Default:   "",
			Usage:     "Path to root state file",
			Value:     &plugin.StateFile,
		},
		&sensu.PluginConfigOption{
			Path:      "numprocs",
			Env:       "CHECK_NUM_PROCS",
			Argument:  "numprocs",
			Shorthand: "n",
			Default:   runtime.NumCPU(),
			Usage:     "Number of processors to use",
			Value:     &plugin.NumProcs,
		},
	}
)

func main() {
	check := sensu.NewGoCheck(&plugin.PluginConfig, options, checkArgs, executeCheck, false)
	check.Execute()
}

func checkArgs(event *types.Event) (int, error) {
	if len(plugin.LogPath) == 0 {
		return sensu.CheckStateWarning, fmt.Errorf("--logpath or CHECK_LOG_PATH environment variable is required")
	}
	if len(plugin.LogRegex) == 0 {
		return sensu.CheckStateWarning, fmt.Errorf("--logregex or CHECK_LOG_REGEX environment variable is required")
	}
	if len(plugin.Match) == 0 {
		return sensu.CheckStateWarning, fmt.Errorf("--match or CHECK_MATCH environment variable is required")
	}
	if len(plugin.StateFile) == 0 {
		return sensu.CheckStateWarning, fmt.Errorf("--statefile or CHECK_STATE_FILE environment variable is required")
	}
	return sensu.CheckStateOK, nil
}

// State represents the state file offset
type State struct {
	Offset json.Number `json:"offset"`
}

func getState(path string) (state State, err error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return state, fmt.Errorf("couldn't read state file: %s", err)
	}
	defer func() {
		err = f.Close()
	}()
	if err := json.NewDecoder(f).Decode(&state); err != nil {
		return state, fmt.Errorf("couldn't read state file: %s", err)
	}
	return state, nil
}

var matches []string
var matches_name []string
var matches_path []string
var matches_return []string

func WalkMatch(root, pattern string) ([]string, error) {
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if matched, err := filepath.Match(pattern, filepath.Base(path)); err != nil {
			return err
		} else if matched {
			matches = append(matches, path)
			matches_name = append(matches_name, info.Name())
			matches_path = append(matches_path, strings.Replace(path, info.Name(), "", -1))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return matches, nil
}

func setState(cur State, path string, file string) (err error) {

	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, os.ModePerm)
	}

	f, err := os.Create((path + file))
	if err != nil {
		return fmt.Errorf("couldn't write state file: %s", err)
	}
	defer func() {
		e := f.Close()
		if err == nil && e != nil {
			err = fmt.Errorf("couldn't close state file: %s", err)
		}
	}()
	if err := json.NewEncoder(f).Encode(cur); err != nil {
		return fmt.Errorf("couldn't write state file: %s", err)
	}
	return nil
}

func fatal(formatter string, args ...interface{}) {
	log.Printf(formatter, args...)
	os.Exit(2)
}

func executeCheck(event *types.Event) (int, error) {
	WalkMatch(plugin.LogPath, plugin.LogRegex)

	for log_element := range matches {

		f, err := os.Open(matches[log_element])
		if err != nil {
			fmt.Println("CRITICAL\nCouldn't open log files")
			return sensu.CheckStateCritical, nil
		}
		defer func() {
			if err := f.Close(); err != nil {
				fmt.Println("CRITICAL\nCouldn't close log files")
			}
		}()
		state, err := getState((plugin.StateFile + "\\" + (strings.Replace(matches[log_element], ":", "", -1))))

		offset, _ := state.Offset.Int64()
		if offset > 0 {
			if _, err := f.Seek(offset, io.SeekStart); err != nil {
				fmt.Println("CRITICAL\nCouldn't seek to offset")
				return sensu.CheckStateCritical, nil
			}

		}

		var reader io.Reader = f
		analyzer := Analyzer{
			Procs: plugin.NumProcs,
			Log:   reader,
			Func:  AnalyzeRegexp(plugin.Match),
		}

		results := analyzer.Go(context.Background())
		eventBuf := new(bytes.Buffer)
		enc := json.NewEncoder(eventBuf)

		for result := range results {
			if result.Err != nil {
				fmt.Println("CRITICAL\nError returning regex results")
				return sensu.CheckStateCritical, nil
			}
			if err := enc.Encode(result); err != nil {
				fmt.Println("CRITICAL\nError encoding result buffer")
				return sensu.CheckStateCritical, nil
			}
			matches_return = append(matches_return, result.Match)

		}

		bytesRead := analyzer.BytesRead()
		state.Offset = json.Number(fmt.Sprintf("%d", offset+bytesRead))

		if err := setState(state, (plugin.StateFile + "\\" + (strings.Replace(matches_path[log_element], ":", "", -1))), matches_name[log_element]); err != nil {
			fmt.Println("CRITICAL\nError setting state:", err)
			return sensu.CheckStateCritical, nil
		}
	}
	if len(matches_return) > 0 {
		fmt.Println("CRITICAL\nMatches found:")
		for matches_index := range matches_return {

			fmt.Println(matches_return[matches_index])

		}
		return sensu.CheckStateCritical, nil

	}

	fmt.Println("OK\nNo matches found in log files")
	return sensu.CheckStateOK, nil

}
