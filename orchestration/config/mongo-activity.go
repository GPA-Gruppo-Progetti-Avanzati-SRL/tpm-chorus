package config

import (
	"encoding/json"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-mongo-common/jsonops"
	"github.com/mitchellh/mapstructure"
)

type MongoActivity struct {
	Activity
	OpType jsonops.MongoJsonOperationType    `yaml:"op-type,omitempty" mapstructure:"op-type,omitempty" json:"op-type,omitempty"`
	PII    PersonallyIdentifiableInformation `yaml:"pii,omitempty" mapstructure:"pii,omitempty" json:"pii,omitempty"`
}

func (c *MongoActivity) WithName(n string) *MongoActivity {
	c.Nm = n
	return c
}

func (c *MongoActivity) WithDescription(n string) *MongoActivity {
	c.Cm = n
	return c
}

func (c *MongoActivity) WithRefDefinition(n string) *MongoActivity {
	c.Definition = n
	return c
}

func (c *MongoActivity) WithOpType(n jsonops.MongoJsonOperationType) *MongoActivity {
	c.OpType = n
	return c
}

func (c *MongoActivity) WithExpressionContext(n string) *MongoActivity {
	c.ExprScope = n
	return c
}

func NewMongoActivity() *MongoActivity {
	s := MongoActivity{}
	s.Tp = MongoActivityType
	return &s
}

func NewMongoActivityFromJSON(message json.RawMessage) (Configurable, error) {
	i := NewMongoActivity()
	err := json.Unmarshal(message, i)
	if err != nil {
		return nil, err
	}

	i.PII.Initialize()
	return i, nil
}

func NewMongoActivityFromYAML(mp interface{}) (Configurable, error) {
	sa := NewMongoActivity()
	err := mapstructure.Decode(mp, sa)
	if err != nil {
		return nil, err
	}

	sa.PII.Initialize()
	return sa, nil
}
