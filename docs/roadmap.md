

# Main principles
- I want to write it robustly and clean with very simple solutions but great architecture, goal is to later iteratively improve some parts without big refactors (like storage engine or consensus algorithm)
- I choosed Go as it is VERY simple and has great stdlib for networking and concurrency, i want to learn more about distributed systems and not fight with language complexities
- I want to write a lot of tests to ensure correctness and robustness of the system
- I want to keep it simple and avoid overengineering

# Storage
Simple key value storage with basic operations.

## Features
- Set key-value pairs
- Get values by key
- Delete key-value pairs
- In-memory storage implementation
- Write-ahead logging for durability
- Pluggable storage backend interface
- Basic unit tests for storage functionalities


# Consensus
Implement consensus algorithm for distributed nodes to agree on the state of the system. I choosed raft algorithm.

## Features
- Leader election
- Log replication
- Handling node failures
- Unit tests for consensus functionalities
- Pluggable consensus backend interface
- Integration with storage layer for log persistence

# TCP/UDP resp2 layer

## Features
- RESP2 protocol parser and renderer
- TCP server that handles RESP2 commands
- UDP server for publish-subscribe messaging
- Support for basic Redis commands (PING, GET, SET, DEL, PUBLISH, SUBSCRIBE)


# Service Discovery

## Features
- simple static configuration for nodes
- maaaybe later i will add custom simple pub/sub based service discovery

# Load Balancing & sharding

## Features
- consistent hashing for sharding keys across multiple nodes
- simple round-robin load balancer for distributing requests among nodes
- automatic re-sharding and scaling (maybe later, it needs custom solution for service discovery)
