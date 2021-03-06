package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"time"
)

func main() {
	flag.IntVar(&maxConcurrent, "threads", 10, "Number of threads to use while scanning.")
	flag.IntVar(&delay, "delay", 0, "Delay between subsequent requests for the same host to avoid overwhelming the host.")
	flag.BoolVar(&useJson, "json", false, "Use body of type JSON in HTTP requests that can contain a body.")
	// flag.BoolVar(&randomScan, "random-scan", false, "Randomly scan IP addresses (Here we go pew pew pew).")
	flag.BoolVar(&useXML, "xml", false, "Use body of type XML in HTTP requests that can contain a body.")
	flag.StringVar(&canaryToken, "token", "", "Canary token payload to use in requests; if empty, a new token will be generated.")
	flag.StringVar(&email, "email", "", "Email to use for the receiving callback notifications.")
	flag.StringVar(&webhook, "webhook", "", "Webhook to use for receiving callback notifications.")
	flag.StringVar(&userAgent, "user-agent", "", "Custom user-agent string to use; if empty, payloads will be used.")
	flag.StringVar(&urlFile, "file", "", "Specify a file containing list of hosts to scan.")
	flag.StringVar(&commonHTTPPorts, "http-ports", "80,443,8080", "Comma separated list of HTTP ports to scan per target.")
	flag.StringVar(&commonFTPPorts, "ftp-ports", "21", "Comma separated list of HTTP ports to scan per target.")
	flag.StringVar(&commonIMAPPorts, "imap-ports", "143,993", "Comma separated list of IMAP ports to scan per target.")
	flag.StringVar(&commonSSHPorts, "ssh-ports", "22", "Comma separated list of SSH ports to scan per target.")
	flag.StringVar(&hMethods, "http-methods", "GET", "Comma separated list of HTTP methods to use while scanning.")
	flag.StringVar(&hHeaders, "headers", "", "Comma separated list of HTTP headers to use; if empty a default set of headers are used.")
	flag.StringVar(&hBody, "fbody", "", "Specify a format string to use as the body of the HTTP request.")
	flag.StringVar(&customServer, "custom-server", "", "Specify a custom callback server.")
	flag.StringVar(&headFile, "headers-file", "", "Specify a file containing custom set of headers to use in HTTP requests.")
	flag.StringVar(&customPayload, "payload", "", "Specify a single payload or a file containing list of payloads to use.")
	flag.StringVar(&proto, "protocol", "all", "Specify a protocol to test for vulnerabilities.")

	mainUsage := func() {
		fmt.Fprint(os.Stdout, lackofart, "\n")
		fmt.Fprintf(os.Stdout, "Usage:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stdout, "\nExamples:\n")
		fmt.Fprint(os.Stdout, "  ./lmp -email alerts@testing.site 1.2.3.4 1.1.1.1:8080\n")
		fmt.Fprint(os.Stdout, "  ./lmp -token xxxxxxxxxxxxxxxxxx -methods POST,PUT -fbody '<padding_here>%s<padding_here>' -headers X-Custom-Header\n")
		fmt.Fprint(os.Stdout, "  ./lmp -webhook https://webhook.testing.site -file internet-ranges.lst -ports 8000,8888\n")
		fmt.Fprint(os.Stdout, "  ./lmp -email alerts@testing.site -methods GET,POST,PUT,PATCH,DELETE 1.2.3.4:8880\n")
		fmt.Fprint(os.Stdout, "  ./lmp -protocol imap -custom-server alerts.testing.local 1.2.3.4:143\n\n")
	}
	flag.Usage = mainUsage
	flag.Parse()
	//fmt.Print(lackofart, "\n")

	allTargets = flag.Args()
	if len(allTargets) < 1 && len(urlFile) < 1 && !randomScan {
		flag.Usage()
		log.Println("You need to supply at least a valid target via arguments or '-file' to scan!")
		os.Exit(1)
	}

	if len(email) < 1 && len(webhook) < 1 && len(canaryToken) < 1 &&
		len(customServer) < 1 && len(customPayload) < 1 {
		flag.Usage()
		log.Println("You need to supply either a email or webhook or a custom callback server to receive notifications at!")
		os.Exit(1)
	}

	fmt.Print(lackofart, "\n\n")

	if proto == "all" {
		log.Println("Running for all protocols...")
	}
	for _, port := range strings.Split(commonHTTPPorts, ",") {
		allHTTPPorts = append(allHTTPPorts, strings.TrimSpace(port))
	}
	for _, port := range strings.Split(commonIMAPPorts, ",") {
		allIMAPPorts = append(allIMAPPorts, strings.TrimSpace(port))
	}
	for _, port := range strings.Split(commonSSHPorts, ",") {
		allSSHPorts = append(allSSHPorts, strings.TrimSpace(port))
	}
	for _, port := range strings.Split(commonFTPPorts, ",") {
		allFTPPorts = append(allFTPPorts, strings.TrimSpace(port))
	}
	for _, method := range strings.Split(hMethods, ",") {
		allMethods = append(allMethods, strings.TrimSpace(method))
	}

	log.Println("Pre-processing payloads to use...")
	if err := processPayloads(); err != nil {
		log.Fatalln("Failed processing payloads!")
	}

	// if user has supplied custom header file
	if len(headFile) > 0 {
		var tmpXHead []string
		file, err := os.Open(headFile)
		if err != nil {
			log.Fatalln(err.Error())
		}
		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			tmpXHead = append(tmpXHead, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Fatalln(err.Error())
		}
		hHeaders = strings.Join(tmpXHead, ",")
		file.Close()
	}

	_, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go handleInterrupt(c, &cancel)

	tnoe := time.Now()
	log.Println("Starting scan at:", tnoe.Local().String())
	go ProcessHosts()

	rand.Seed(time.Now().UnixNano())
	initDispatcher(maxConcurrent)
	dnoe := time.Now()
	fmt.Print("\n")
	log.Println("Please visit your email/webhook/custom callback server for seeing triggers.")
	log.Println("Scan finished at:", dnoe.Local().String())
	log.Println("Total time taken to scan:", time.Since(tnoe).String())
	log.Println("LogMePwn is exiting.")
}
