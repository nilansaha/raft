### Raft Consensus Protocol

POC to implement the election algorithms for Raft and reacting to leader failures

- Run the three commands in three different terminal tabs and watch them select a leader
- Kill the leader node and watch a new leader getting selected
- Run the killed node again and watching it join the network and accept the current leader

```
go run main.go --server-port :8080
go run main.go --server-port :8081
go run main.go --server-port :8082
```