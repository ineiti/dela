package controller

import (
	"go.dedis.ch/dela/cli"
	"go.dedis.ch/dela/cli/node"
	"go.dedis.ch/dela/core/access"
)

// NewController returns a new controller initializer
func NewController() node.Initializer {
	return controller{}
}

// controller is an initializer with a set of commands.
//
// - implements node.Initializer
type controller struct{}

// Build implements node.Initializer.
func (m controller) SetCommands(builder node.Builder) {

	cmd := builder.SetCommand("e-voting")
	cmd.SetDescription("... ")

	// memcoin --config /tmp/node1 e-voting initHttpServer --portNumber 8080
	sub := cmd.SetSubCommand("initHttpServer")
	sub.SetDescription("Initialize the HTTP server")
	sub.SetFlags(cli.StringFlag{
		Name:     "portNumber",
		Usage:    "port number of the HTTP server",
		Required: true,
	})
	sub.SetAction(builder.MakeAction(&initHttpServerAction{
		ElectionIdNonce: 0,
		// TODO : should have the same client as pool controller
		client:          &client{nonce: 1},
	}))

	sub = cmd.SetSubCommand("createElectionTest")
	sub.SetDescription("createElectionTest")
	sub.SetAction(builder.MakeAction(&createElectionTestAction{}))
}

// OnStart implements node.Initializer. It creates and registers a pedersen DKG.
func (m controller) OnStart(ctx cli.Flags, inj node.Injector) error {
	return nil
}

// OnStop implements node.Initializer.
func (controller) OnStop(node.Injector) error {
	return nil
}

// client return monotically increasing nonce
//
// - implements signed.Client
type client struct {
	nonce uint64
}

// GetNonce implements signed.Client
func (c *client) GetNonce(access.Identity) (uint64, error) {
	res := c.nonce
	c.nonce++
	return res, nil
}