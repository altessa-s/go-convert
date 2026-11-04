// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package oneof_test

import (
	"fmt"

	"github.com/altessa-s/go-convert"
	"github.com/altessa-s/go-convert/proto/codec/oneof"
)

// The types below have the shape protoc-gen-go generates for
//
//	message Invitation {
//	  oneof role {
//	    Contractor contractor = 1;
//	    Agent agent = 2;
//	  }
//	}

type Contractor struct{ Company string }

type Agent struct{ Agency string }

type isInvitation_Role interface{ isInvitation_Role() } //nolint:revive // protoc naming

type Invitation_Contractor struct{ Contractor *Contractor } //nolint:revive // protoc naming

func (*Invitation_Contractor) isInvitation_Role() {}

type Invitation_Agent struct{ Agent *Agent } //nolint:revive // protoc naming

func (*Invitation_Agent) isInvitation_Role() {}

type Invitation struct{ Role isInvitation_Role }

func ExampleNew() {
	// The domain side holds one pointer field per oneof member.
	type Role struct {
		Contractor *Contractor
		Agent      *Agent
	}

	type InvitationModel struct{ Role Role }

	codec := oneof.New(oneof.WithWrapperRegistry(map[string]any{
		"Contractor": &Invitation_Contractor{},
		"Agent":      &Invitation_Agent{},
	}))

	msg := convert.Convert(InvitationModel{Role: Role{Agent: &Agent{Agency: "acme"}}}, &Invitation{}, convert.WithCodecs(codec))
	agent, ok := msg.Role.(*Invitation_Agent)
	fmt.Println(ok, agent.Agent.Agency)

	model := convert.Convert(msg, &InvitationModel{}, convert.WithCodecs(codec))
	fmt.Println(model.Role.Contractor == nil, model.Role.Agent.Agency)
	// Output:
	// true acme
	// true acme
}
