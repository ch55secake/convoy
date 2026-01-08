package loadbalancer

// Balancer interface for load distribution
type Balancer interface {
	Next() string
	AddServer(server string)
	RemoveServer(server string)
}
