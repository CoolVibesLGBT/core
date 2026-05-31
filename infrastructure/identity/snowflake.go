package identity

import "core/helpers"

type SnowflakePublicIDGenerator struct {
	node *helpers.Node
}

func NewSnowflakePublicIDGenerator(node *helpers.Node) *SnowflakePublicIDGenerator {
	return &SnowflakePublicIDGenerator{node: node}
}

func (g *SnowflakePublicIDGenerator) GeneratePublicID() int64 {
	return g.node.Generate().Int64()
}
