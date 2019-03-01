package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	env_key          = "kube_env"
	qfpay_path       = "services/qfpay"
	adyen_path       = "services/adyen"
	mpgs_path        = "services/mpgs"
	alipay_path      = "services/alipay"
	etone_path       = "services/etonepay"
	allinpay_path    = "services/allinpay"
	wechatpay_path   = "services/wechatpay"
	octopus_path     = "services/octopus"
	tapgo_path       = "services/tapgo"
	cybersource_path = "services/cybersource"
	sdk_path         = "services/payment_sdk"
	bea_path         = "services/payment-services/payment-bea"
	beacup_path      = "services/payment-services/payment-bea-cup"
	sic_path         = "services/payment-services/payment-sic"
	wlb_path         = "services/payment-services/payment-WLB"
	fake_path        = "services/payment-services/payment-fake"
	gateway_path     = "gateway"
)

var all_path = []string{
	qfpay_path,
	adyen_path,
	mpgs_path,
	alipay_path,
	etone_path,
	allinpay_path,
	wechatpay_path,
	octopus_path,
	tapgo_path,
	cybersource_path,
	sdk_path,
	gateway_path,
}

func main() {
	args := os.Args[1:]
	var execFn = func(string, []string) {}
	switch true {
	case strings.HasPrefix("logs", args[0]):
		execFn = kubectlLogs
	case strings.HasPrefix("pods", args[0]):
		execFn = kubctlPod
	case strings.HasPrefix("env", args[0]):
		execFn = kubectlEnv
	case strings.HasPrefix("replace", args[0]):
		execFn = kubectlReplace
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

func kubectlReplace(arg string, customerArgs []string) {
	if os.Getenv(env_key) == "prod" {
		log.Fatalf("the enviroment is prod  !!!")
		return
	}
	if len(customerArgs) == 0 || len(customerArgs[0]) != 40 {
		log.Fatalf("customerArgs is illegal  !!!")
		return
	}
	fileName := "/Users/huhai/develop/develop-scripts/"
	switch true {
	case strings.HasPrefix("qfpay", arg):
		fileName += qfpay_path
	case strings.HasPrefix("adyen", arg):
		fileName += adyen_path
	case strings.HasPrefix("mpgs", arg):
		fileName += mpgs_path
	case strings.HasPrefix("alipay", arg):
		fileName += alipay_path
	case strings.HasPrefix("etone", arg):
		fileName += etone_path
	case strings.HasPrefix("allinpay", arg):
		fileName += allinpay_path
	case strings.HasPrefix("wechatpay", arg):
		fileName += wechatpay_path
	case strings.HasPrefix("octopus", arg):
		fileName += octopus_path
	case strings.HasPrefix("tapgo", arg):
		fileName += tapgo_path
	case strings.HasPrefix("cybersource", arg):
		fileName += cybersource_path
	case strings.HasPrefix("sdk", arg):
		fileName += sdk_path
	case strings.HasPrefix("bea", arg):
		fileName += bea_path
	case strings.HasPrefix("beacup", arg):
		fileName += beacup_path
	case strings.HasPrefix("sic", arg):
		fileName += sic_path
	case strings.HasPrefix("wlb", arg):
		fileName += wlb_path
	case strings.HasPrefix("fake", arg):
		fileName += fake_path
	case strings.HasPrefix("gateway", arg):
		fileName += gateway_path
	case strings.HasPrefix("_all", arg):
		replaceByFileName(customerArgs[0], all_path...)
		return
	default:
		log.Fatalf("wrong replace name")
		return
	}
	replaceByFileName(customerArgs[0], fileName)
}

func replaceByFileName(arg string, names ...string) {
	for _, name := range names {
		name += "/deployment.yaml"
		//read lines
		lines, err := readLineFromFile(name, arg)
		if err != nil {
			log.Fatalf("readLineFromFile error : %s", err.Error())
			return
		}
		//write lines
		if err := writeLineToFile(name, lines); err != nil {
			log.Fatalf("writeLineToFile error : %s", err.Error())
			return
		}
		var s string
		for s == "" {
			fmt.Scanf("%s\n", &s)
		}
		if s == "quit" {
			continue
		}

		cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("kubectl replace -f %s", name))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			//log.Fatalf("exec.Command error : %s", err.Error())
			return
		}
		fmt.Println("replace success !")
	}
}

func readLineFromFile(fileName, dstStr string) ([]string, error) {
	var lines []string
	file, err := os.OpenFile(fileName, os.O_RDONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed opening file: %s", err)

	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		txt := scanner.Text()
		oriText := txt
		if strings.Contains(txt, "image:") && strings.Contains(txt, "bindo-staging-tw") {
			ss := strings.Split(txt, ":")
			if len(ss) == 3 {
				ss[2] = dstStr
			}
			txt = strings.Join(ss, ":")
			fmt.Println("-----------")
			fmt.Printf("%s\n", oriText)
			fmt.Println(txt)
			fmt.Println("-----------")
		}
		lines = append(lines, txt)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %s", err.Error())
	}
	return lines, nil
}

func writeLineToFile(fileName string, lines []string) error {
	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed opening file: %s", err)

	}
	defer file.Close()
	w := bufio.NewWriter(file)
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
	return w.Flush()
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
	} else {
		os.Setenv(env_key, "prod")
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
				if ss := strings.Split(name, "-"); len(ss) >= 3 {
					return strings.Contains(strings.Join(ss[:len(ss)-2], "-"), cmd.Arg)
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
		cmd.Cmd.Args = append(cmd.Cmd.Args, "--timestamps=true")
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
