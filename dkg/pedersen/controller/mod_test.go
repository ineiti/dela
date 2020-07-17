package controller

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/dela/cli/node"
	"go.dedis.ch/dela/dkg/pedersen"
	"go.dedis.ch/dela/internal/testing/fake"
	"go.dedis.ch/dela/mino"
	"golang.org/x/xerrors"
)

func TestMinimal_Inject(t *testing.T) {
	minimal := NewMinimal()

	inj := newInjector(fake.Mino{})
	err := minimal.Inject(nil, inj)
	require.NoError(t, err)

	require.Len(t, inj.(*fakeInjector).history, 1)
	require.IsType(t, &pedersen.Pedersen{}, inj.(*fakeInjector).history[0])

	err = minimal.Inject(nil, newBadInjector())
	require.EqualError(t, err, "failed to resolve mino: oops")
}

// -----------------------------------------------------------------------------
// Utility functions

func newInjector(mino mino.Mino) node.Injector {
	return &fakeInjector{
		mino: mino,
	}
}

func newBadInjector() node.Injector {
	return &fakeInjector{
		isBad: true,
	}
}

// fakeInjector is a fake injector
//
// - implements node.Injector
type fakeInjector struct {
	isBad   bool
	mino    mino.Mino
	history []interface{}
}

// Resolve implements node.Injector
func (i fakeInjector) Resolve(el interface{}) error {
	if i.isBad {
		return xerrors.New("oops")
	}

	switch msg := el.(type) {
	case *mino.Mino:
		if i.mino == nil {
			return xerrors.New("oops")
		}
		*msg = i.mino
	default:
		return xerrors.Errorf("unkown message '%T", msg)
	}

	return nil
}

// Inject implements node.Injector
func (i *fakeInjector) Inject(v interface{}) {
	if i.history == nil {
		i.history = make([]interface{}, 0)
	}
	i.history = append(i.history, v)
}