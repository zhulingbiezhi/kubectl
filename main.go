package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	args := os.Args[1:]
	var execFn = func(string, []string) {}
	switch args[0] {
	case "log", "logs":
		execFn = kubectlLogs
	case "po", "pod", "pods":
		execFn = kubctlPod
	case "env", "e":
		execFn = kubectlEnv
	}

	execFn(args[1], args[2:])
	//time.Sleep(time.Second*10)
}

type Command struct {
	Cmd          *exec.Cmd
	Arg          string
	CustomerArgs []string
	PreRun       func(cmd *Command) error
	Filters      []func(cmd *Command) error
	Run          func(cmd *Command) error
	AfterRun     func(cmd *Command) error
	Close        func(cmd *Command) error
}

func kubectlLogs(Arg string, customerArgs []string) {
	cmds := []*Command{
		{
			CustomerArgs: customerArgs,
			Arg:          Arg,
			PreRun:       preRun,
			Cmd:          exec.Command("/bin/sh", "-c", fmt.Sprintf("kubectl get pods --all-namespaces | grep %s", Arg)),
			Run:          run,
			AfterRun:     afterRun,
		},
		{
			CustomerArgs: customerArgs,
			Arg:          Arg,
			PreRun:       preRunLogs,
			Run:          run,
			AfterRun:     afterRun,
			Cmd:          exec.Command("kubectl", "logs", "-f"),
		},
	}
	if err := cmdPipe(cmds...); err != nil {
		return
	}
}

func kubctlPod(arg string, customerArgs []string) {

}

func kubectlEnv(arg string, customerArgs []string) {
	script := ""
	switch arg {
	case "stg", "stag", "staging", "stging":
		script = "stging.sh"
	case "prod", "prd":
		script = "prod.sh"
	}
	cmd := exec.Command("/bin/sh", fmt.Sprintf("/Users/huhai/scripts/%s", script))
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		fmt.Println("kubectlEnv error: ", err.Error())
	}
}

func cmdPipe(cmds ...*Command) error {
	l := len(cmds)
	for i, _ := range cmds[:l-1] {
		wc, _ := cmds[i+1].Cmd.StdinPipe()
		cmds[i].Cmd.Stderr = os.Stderr
		cmds[i].Cmd.Stdout = wc.(io.Writer)
		cmds[i+1].Close = func(*Command) error {
			return wc.Close()
		}
	}
	cmds[l-1].Cmd.Stdout = os.Stdout
	cmds[l-1].Cmd.Stderr = os.Stderr
	for i, cmd := range cmds {
		for _, f := range cmd.Filters {
			f(cmds[i])
		}
		if cmd.Run == nil {
			cmd.Run = func(cmd *Command) error {
				return cmd.Cmd.Run()
			}
		}
		type F func(*Command) error
		for _, f := range []F{cmd.PreRun, cmd.Run, cmd.AfterRun, cmd.Close} {
			if f != nil {
				if err := f(cmds[i]); err != nil {
					fmt.Printf("%s  error: %s\n", cmd.Cmd.Args, err.Error())
					return err
				}
			}
		}
	}
	return nil
}

func filter(cmd *Command) {
	//fmt.Println("---filters")
}

func run(cmd *Command) error {
	//fmt.Println("run start: ", cmd.Cmd.Args)
	//if cmd.Cmd.Stdin != nil {
	//	fmt.Println("---stdin:")
	//	bufio.NewReader(cmd.Cmd.Stdin).WriteTo(os.Stdout)
	//}
	if cmd.Cmd.Stdout != nil {
		//fmt.Println("---stdout:")
		w := cmd.Cmd.Stdout
		if w != os.Stdout {
			cmd.Cmd.Stdout = io.MultiWriter(os.Stdout, w)
		}
		return cmd.Cmd.Run()
	}
	return nil
}

func preRun(cmd *Command) error {
	//fmt.Println("---beforeRun")
	return nil
}

func afterRun(cmd *Command) error {
	//fmt.Println("---afterRun")
	return nil
}

func preRunLogs(cmd *Command) error {
	//fmt.Println("---preRunLogs")
	s := bufio.NewScanner(cmd.Cmd.Stdin)
	s.Split(bufio.ScanLines)
	pod := make(chan podStatus, 1)
	go func() {
		for s.Scan() {
			line := s.Text()
			ok, p := parsePodStatus(line, func(name string) bool {
				if ss := strings.Split(name, "-"); len(ss) > 0 {
					return strings.Contains(ss[0], cmd.Arg)
				}
				return false
			})
			if ok && p.Status == "Running" {
				pod <- p
				break
			}
		}
	}()
	defer func() {
		cmd.Close(cmd)
	}()
	select {
	case p := <-pod:
		cmd.Cmd.Args = append(cmd.Cmd.Args, p.Name, "-n", p.NameSpace)
		cmd.Cmd.Args = append(cmd.Cmd.Args, cmd.CustomerArgs...)
	case <-time.After(time.Second * 15):
		return fmt.Errorf("get pod  time out !")
	}
	return nil
}

type podStatus struct {
	NameSpace    string `json:"name_space"`
	Name         string `json:"name"`
	Ready        string `json:"ready"`
	Status       string `json:"status"`
	RestartTimes string `json:"restart_times"`
	LivedTime    string `json:"lived_time"`
}

func parsePodStatus(line string, match func(string) bool) (bool, podStatus) {
	var pod podStatus
	strs := strings.Fields(line)
	if len(strs) == 6 {
		pod.NameSpace = strs[0]
		pod.Name = strs[1]
		pod.Ready = strs[2]
		pod.Status = strs[3]
		pod.RestartTimes = strs[4]
		pod.LivedTime = strs[5]
		if match(pod.Name) {
			fmt.Printf("match: %s  %s  %s\n", pod.NameSpace, pod.Name, pod.Status)
			return true, pod
		}
	}
	return false, pod
}
