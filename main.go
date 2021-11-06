package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"
)

var serverPort = flag.String("server-port", "8080", "HTTP port")
var peerPorts = []string{"8080", "8081", "8082"}
var timeout = 2 * time.Second

type Raft struct {
	localAddr     string
	peers         []string
	currentTerm   int
	lastHeartbeat time.Time
	givenVotes    map[int]int
	isLeader      bool
}

func NewRaft() *Raft {
	peers := make([]string, 0)
	for _, port := range peerPorts {
		if port != *serverPort {
			peers = append(peers, fmt.Sprintf("http://0.0.0.0:%s", port))
		}
	}

	return &Raft{
		localAddr:   fmt.Sprintf("http://0.0.0.0:%s", *serverPort),
		peers:       peers,
		currentTerm: 0,
		givenVotes:  make(map[int]int),
		isLeader:    false,
	}
}

func (raft *Raft) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	requestServer := r.URL.Query()["server"][0]
	requestTerm, _ := strconv.Atoi(r.URL.Query()["term"][0])

	if requestTerm > raft.currentTerm {
		raft.currentTerm = requestTerm
		raft.isLeader = false
		fmt.Println("New Leader is", requestServer)
	}

	raft.lastHeartbeat = time.Now()
	fmt.Fprintf(w, "Ack. Heartbeat")
}

func (raft *Raft) handleRequestVotes(w http.ResponseWriter, r *http.Request) {
	requestServer := r.URL.Query()["server"][0]
	requestTerm, _ := strconv.Atoi(r.URL.Query()["term"][0])

	fmt.Println(requestServer, "Asking for vote on term", requestTerm)

	if _, exist := raft.givenVotes[requestTerm]; exist || raft.currentTerm > requestTerm {
		w.WriteHeader(500)
	} else {
		w.WriteHeader(200)
		fmt.Println("Vote given to", requestServer)
	}
	fmt.Fprintf(w, "Ack. RequestVotes")
}

func (raft *Raft) callElection() {
	votesRequired := int(math.Round(float64(len(raft.peers)) / 2))
	receivedVotes := 1
	raft.currentTerm++
	raft.givenVotes[raft.currentTerm] = 1

	for _, server := range raft.peers {
		query := fmt.Sprintf("%s/requestVote?term=%d&server=%s", server, raft.currentTerm, raft.localAddr)
		resp, err := http.Get(query)
		if err != nil {
			fmt.Println("Error on", server, err)
			continue
		}

		if resp.StatusCode == 200 {
			receivedVotes++
			fmt.Println("Received vote from", server)
		} else {
			fmt.Println("Didn't receive vote from", server)
		}

		if receivedVotes >= votesRequired {
			fmt.Println("I am Leader now")
			raft.isLeader = true
			raft.sendHeartbeats()
			break
		}
	}
}

func (raft *Raft) sendHeartbeats() {
	for _, server := range raft.peers {
		query := fmt.Sprintf("%s/heartbeat?server=%s&term=%d", server, raft.localAddr, raft.currentTerm)
		http.Get(query)
	}
}

func (raft *Raft) checkHeartbeat() {
	if raft.lastHeartbeat.IsZero() || time.Since(raft.lastHeartbeat) >= timeout {
		fmt.Println("Calling Election")
		raft.callElection()
	}
}

func (raft *Raft) runRoutine() {
	for range time.Tick(timeout) {
		if raft.isLeader {
			raft.sendHeartbeats()
		} else {
			raft.checkHeartbeat()
		}
	}
}

func main() {
	flag.Parse()

	raft := NewRaft()
	http.HandleFunc("/heartbeat", raft.handleHeartbeat)
	http.HandleFunc("/requestVote", raft.handleRequestVotes)

	go raft.runRoutine()

	fmt.Println("Web Server Started @", *serverPort)
	log.Fatal(http.ListenAndServe(":"+*serverPort, nil))
}
