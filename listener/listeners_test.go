package listener

import (
	// "fmt"
	"testing"

	"github.com/davidwalter0/llb/pipe"
)

type TestPipe map[string][]pipe.Definition

var _testPipe1 = TestPipe{
	"Equal": []pipe.Definition{
		pipe.Definition{
			Source:    "0.0.0.0:8001",
			Sink:      "0.0.0.0:8002",
			Namespace: "test",
			Name:      "echo",
		},
		pipe.Definition{
			Source:    "0.0.0.0:8001",
			Sink:      "0.0.0.0:8002",
			Namespace: "test",
			Name:      "echo",
		},
	},
	"!Equal": []pipe.Definition{
		pipe.Definition{
			Source:    "0.0.0.0:8002",
			Sink:      "0.0.0.0:8003",
			Name:      "echo",
			Namespace: "test",
		},
		pipe.Definition{
			Source:    "0.0.0.0:8002",
			Sink:      "0.0.0.0:8004",
			Name:      "echo",
			Namespace: "test",
		},
	},
}

var _TestDefinitionEqual = []pipe.Definition{
	pipe.Definition{
		Source:    "0.0.0.0:8001",
		Sink:      "0.0.0.0:8002",
		Name:      "echo",
		Namespace: "test",
	},
	pipe.Definition{
		Source:    "0.0.0.0:8001",
		Sink:      "0.0.0.0:8002",
		Name:      "echo",
		Namespace: "test",
	},
}

var _TestDefinitionNotEqual = []pipe.Definition{
	pipe.Definition{
		Source:    "0.0.0.0:8002",
		Sink:      "0.0.0.0:8003",
		Name:      "echo",
		Namespace: "test",
	},
	pipe.Definition{
		Source:    "0.0.0.0:8002",
		Sink:      "0.0.0.0:8004",
		Name:      "echo",
		Namespace: "test",
	},
}

func TestDefinition(t *testing.T) {
	if !_TestDefinitionEqual[0].Equal(_TestDefinitionEqual[1]) {
		t.Errorf("%v %v", _TestDefinitionEqual[0], _TestDefinitionEqual[1])
	}
	if pipe := _TestDefinitionEqual[0].Copy(_TestDefinitionEqual[1]); !pipe.Equal(_TestDefinitionEqual[0]) || !pipe.Equal(_TestDefinitionEqual[1]) {
		t.Errorf("%v %v", _TestDefinitionEqual[0], _TestDefinitionEqual[1])
	}
	if _TestDefinitionNotEqual[0].Equal(_TestDefinitionNotEqual[1]) {
		t.Errorf("%v %v", _TestDefinitionNotEqual[0], _TestDefinitionNotEqual[1])
	}
}

func TestDefinitionMap(t *testing.T) {
	m := _testPipe1
	// fmt.Println(m)
	equal := m["Equal"]
	notequal := m["!Equal"]
	if !equal[0].Equal(equal[1]) {
		t.Errorf("%v %v", equal[0], equal[1])
	}
	if notequal[0].Equal(notequal[1]) {
		t.Errorf("%v %v", notequal[0], notequal[1])
	}

	if p1, p2 := equal[0].Copy(notequal[0]), equal[1].Copy(notequal[1]); !p1.Equal(notequal[0]) || !p2.Equal(notequal[1]) {
		t.Errorf("%v %v", p1, p2)
	}
	if p1 := equal[1].Copy(notequal[1]); !p1.Equal(notequal[1]) {
		t.Errorf("%v %v", p1, notequal[1])
	}
	if p2 := equal[1].Copy(notequal[1]); !p2.Equal(notequal[1]) {
		t.Errorf("%v %v", p2, notequal[1])
	}
	if !equal[0].Equal(equal[1]) {
		t.Errorf("Copy Value Modified Reference %v %v", equal[0], equal[1])
	}
	if notequal[0].Equal(notequal[1]) {
		t.Errorf("Copy Value Modified Reference %v %v", notequal[0], notequal[1])
	}
	p1, p2 := equal[0].Copy(notequal[0]), equal[1].Copy(notequal[1])
}