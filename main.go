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

var serverPort = flag.String("server-port", ":8080", "HTTP port")
var allServers = []string{":8080", ":8081", ":8082"}
var timeout float64 = 2

type State struct {
	currentTerm   int
	lastHeartbeat time.Time
	givenVotes    map[int]int
	isLeader      bool
}

func NewState() *State {
	return &State{
		currentTerm: 0,
		givenVotes:  make(map[int]int),
		isLeader:    false,
	}
}

func (s *State) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	requestServer := r.URL.Query()["server"][0]
	requestTerm, _ := strconv.Atoi(r.URL.Query()["term"][0])

	if requestTerm > s.currentTerm {
		s.currentTerm = requestTerm
		s.isLeader = false
		fmt.Println("New Leader is", requestServer)
	}

	s.lastHeartbeat = time.Now()
	fmt.Fprintf(w, "Ack. Heartbeat")
}

func (s *State) handleRequestVotes(w http.ResponseWriter, r *http.Request) {
	requestServer := r.URL.Query()["server"][0]
	requestTerm, _ := strconv.Atoi(r.URL.Query()["term"][0])

	fmt.Println(requestServer, "Asking for vote on term", requestTerm)

	if _, exist := s.givenVotes[requestTerm]; exist || s.currentTerm > requestTerm {
		w.WriteHeader(500)
	} else {
		w.WriteHeader(200)
		fmt.Println("Vote given to", requestServer)
	}
	fmt.Fprintf(w, "Ack. RequestVotes")
}

func (s *State) callElection() {
	votesRequired := int(math.Round(float64(len(allServers)) / 2))
	receivedVotes := 1
	s.currentTerm++
	s.givenVotes[s.currentTerm] = 1

	for _, server := range allServers {
		if server != *serverPort {
			serverAddress := fmt.Sprintf("http://0.0.0.0%s", server)
			query := fmt.Sprintf("%s/requestVote?term=%d&server=%s", serverAddress, s.currentTerm, *serverPort)
			resp, err := http.Get(query)
			if err != nil {
				fmt.Println("Error on", serverAddress, err)
				continue
			}

			if resp.StatusCode == 200 {
				receivedVotes++
				fmt.Println("Received vote from", server)
			} else {
				fmt.Println("Didn't receive vote from", server)
			}
		}

		if receivedVotes >= votesRequired {
			fmt.Println("I am Leader now")
			s.isLeader = true
			s.sendHeartbeats()
			break
		}
	}
}

func (s *State) sendHeartbeats() {
	for _, server := range allServers {
		if server != *serverPort {
			serverAddress := fmt.Sprintf("http://0.0.0.0%s", server)
			query := fmt.Sprintf("%s/heartbeat?server=%s&term=%d", serverAddress, *serverPort, s.currentTerm)
			http.Get(query)
		}
	}
}

func (s *State) checkHeartbeat() {
	if s.lastHeartbeat.IsZero() || time.Since(s.lastHeartbeat).Seconds() >= timeout {
		fmt.Println("Calling Election")
		s.callElection()
	}
}

func (s *State) heartbeatRoutine() {
	for range time.Tick(time.Duration(timeout) * time.Second) {
		if s.isLeader {
			s.sendHeartbeats()
		} else {
			s.checkHeartbeat()
		}
	}
}

func main() {
	flag.Parse()

	state := NewState()
	http.HandleFunc("/heartbeat", state.handleHeartbeat)
	http.HandleFunc("/requestVote", state.handleRequestVotes)

	go state.heartbeatRoutine()

	fmt.Println("Web Server Started @", *serverPort)
	log.Fatal(http.ListenAndServe(*serverPort, nil))
}
