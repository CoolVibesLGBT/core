package mcp

type Agent interface {
	Name() string
	Handle(msg Envelope) (Envelope, error)
}

type Registry struct {
	agents map[string]Agent
}

func NewRegistry() *Registry {
	return &Registry{
		agents: make(map[string]Agent),
	}
}

func (r *Registry) Register(agent Agent) {
	r.agents[agent.Name()] = agent
}

func (r *Registry) Get(name string) (Agent, bool) {
	agent, ok := r.agents[name]
	return agent, ok
}
